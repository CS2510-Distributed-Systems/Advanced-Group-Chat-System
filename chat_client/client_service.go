package chat_client

import (
	// "bufio"
	"bufio"
	"chat-system/pb"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	// "os"
	"strconv"
	"strings"

	// "time"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// client service
type ChatServiceClient struct {
	chatservice pb.ChatServiceClient
	authservice pb.AuthServiceClient
	clientstore ClientStore
	serverId    int
}

func NewChatServiceClient(chatservice pb.ChatServiceClient, authservice pb.AuthServiceClient, store ClientStore, serverId int) *ChatServiceClient {
	return &ChatServiceClient{
		chatservice: chatservice,
		authservice: authservice,
		clientstore: store,
		serverId:    serverId,
	}
}

// login rpc
func UserLogin(user_name string, client *ChatServiceClient) error {

	user := &pb.User{
		Id:   uuid.New().ID(),
		Name: user_name,
	}
	user_details := &pb.LoginRequest{
		User: user,
	}
	_, err := client.authservice.Login(context.Background(), user_details)
	if err != nil {
		return err
	}
	client.clientstore.SetUser(user)
	log.Printf("User %v Logged in succesfully.", client.clientstore.GetUser().Name)

	return nil
}

// logoutrpc to remove the user from the server
func (client *ChatServiceClient) UserLogout() bool {

	user := client.clientstore.GetUser()
	groupname := client.clientstore.GetGroup().Groupname

	if user.Id == 0 {
		return true
	}
	req := &pb.LogoutRequest{
		Logout: &pb.Logout{
			User:      user,
			Currgroup: groupname,
		},
	}
	resp, err := client.authservice.Logout(context.Background(), req)
	if err != nil {
		log.Println("cannot Logout.Please Try again")
	}
	client.clientstore.SetUser(
		&pb.User{
			Name: "",
			Id:   0,
		},
	)
	log.Printf("User %v Logged out from server succesfully.", user.Name)
	return resp.Status
}

func (client *ChatServiceClient) ServerView() {
	req := &pb.ServerViewRequest{
		Event: "v",
	}
	resp, err := client.chatservice.ServerView(context.Background(), req)
	if err != nil {
		log.Println("cannot Logout.Please Try again")
	}
	peerservers := resp.Peerservers
	log.Printf("Client is currently connected to server%v whose peer servers are %v", client.serverId, peerservers)
}

// Biirectional rpc
func (client *ChatServiceClient) JoinGroupChat(mesg string) {

	stream, err := client.chatservice.JoinGroupChat(context.Background())
	if err != nil {
		log.Fatalf("open stream error %v", err)
		return
	}

	done := make(chan bool)

	//to join the group
	req1, _ := client.ProcessMessage(mesg)
	if err := stream.Send(req1); err != nil {
		log.Println("Send request error")
		return
	}

	//go routine for receive
	go func() {
		for {
			err := contextError(stream.Context())
			if err != nil {
				return
			}
			resp, err := stream.Recv()
			if err == io.EOF {
				log.Println("EOF error")
				close(done)
				return
			}
			if err != nil {
				return
			}

			//process response
			command := resp.Event
			if command == "j" {
				_ = client.clientstore.SetGroup(resp.GetGroup())
			}

			if command == "p" {
				PrintAll(resp.Group)
			} else {
				PrintRecent(resp.Group)
			}

		}
	}()

	//go routine for send
	go func() {
		for {
			log.Printf("Enter the message in the group:")
			msg, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				log.Println("Client crashed inside the stream. Performing graceful shutdown")
				return
			}

			//preparing request
			req, command := client.ProcessMessage(msg)
			if command == "q" {
				if err := stream.CloseSend(); err != nil {
					log.Println(err)
				}
				close(done)
				return
			}
			if err := stream.Send(req); err != nil {
				log.Println("Send request error")
				return
			}
		}
	}()

	<-done
	log.Println("Streaming Ended. Thank you")
	return
}

func (client *ChatServiceClient) ProcessMessage(msg string) (*pb.GroupChatRequest, string) {
	msg = strings.Trim(msg, "\r\n")
	args := strings.Split(msg, " ")
	cmd := strings.TrimSpace(args[0])
	msg = strings.Join(args[1:], " ")

	switch cmd {
	case "a":
		if client.clientstore.GetUser().GetName() == "" {
			log.Println("Please login to join a group.")
		} else if client.clientstore.GetGroup().Groupname == "" {
			log.Println("Please join a group to send a message")
		} else {
			//appending a message
			appendchat := &pb.GroupChatRequest_Append{
				Append: &pb.AppendChat{
					Group: client.clientstore.GetGroup(),
					Chatmessage: &pb.ChatMessage{
						MessagedBy: client.clientstore.GetUser(),
						Message:    msg,
						LikedBy:    make(map[string]string, 0),
					},
				},
			}
			req := &pb.GroupChatRequest{
				Action: appendchat,
			}
			return req, cmd

		}

	//like message
	case "l":
		if client.clientstore.GetUser().GetName() == "" {
			log.Println("Please login to join a group.")
		} else if client.clientstore.GetGroup().Groupname == "" {
			log.Println("Please join a group to send a message")
		} else {
			messagenumber, err := strconv.ParseUint(msg, 10, 32)
			if err != nil {
				log.Printf("please provide a valid number to like")
			}
			likemessage := &pb.GroupChatRequest_Like{
				Like: &pb.LikeMessage{
					Likeduser: client.clientstore.GetUser(),
					Messageid: uint32(messagenumber),
					Group:     client.clientstore.GetGroup(),
				},
			}
			req := &pb.GroupChatRequest{
				Action: likemessage,
			}
			return req, cmd

		}

	case "r":
		messagenumber, err := strconv.ParseUint(msg, 10, 32)
		if err != nil {
			log.Printf("please provide a valid number to unlike")
		}
		unlikemessage := &pb.GroupChatRequest_Unlike{
			Unlike: &pb.UnLikeMessage{
				Unlikeduser: client.clientstore.GetUser(),
				Messageid:   uint32(messagenumber),
				Group:       client.clientstore.GetGroup(),
			},
		}
		req := &pb.GroupChatRequest{
			Action: unlikemessage,
		}
		return req, cmd

	case "p":
		{
			print := &pb.GroupChatRequest_Print{
				Print: &pb.PrintChat{
					User:      client.clientstore.GetUser(),
					Groupname: client.clientstore.GetGroup().Groupname,
				},
			}
			req := &pb.GroupChatRequest{
				Action: print,
			}
			return req, cmd
		}

	case "j":
		//join the group
		currgroup := "None"
		if client.clientstore.GetGroup().GroupID != 0 {
			currgroup = client.clientstore.GetGroup().Groupname
		}
		joinchat := &pb.GroupChatRequest_Joinchat{
			Joinchat: &pb.JoinChat{
				Joineduser: client.clientstore.GetUser(),
				Newgroup:   msg,
				Currgroup:  currgroup,
			},
		}
		req := &pb.GroupChatRequest{
			Action: joinchat,
		}
		return req, cmd

	//quit the program
	case "q":
		client.UserLogout()
		req := &pb.GroupChatRequest{}
		return req, cmd
	case "v":
		client.ServerView()
		req := &pb.GroupChatRequest{}
		return req, cmd
	default:
		log.Printf("Cannot read the message, please enter again")
		return &pb.GroupChatRequest{}, cmd

	}
	return &pb.GroupChatRequest{}, cmd

}

// print only 10 recent messages
func PrintRecent(group *pb.Group) {
	groupname := group.GetGroupname()
	participants := maps.Values(group.GetParticipants())
	chatmessages := group.GetMessages()
	chatlength := len(chatmessages)
	print_recent := 10
	fmt.Printf("Group: %v\n", groupname)
	fmt.Printf("Participants: %v\n", participants)
	for print_recent > 0 {
		i := uint32(chatlength - print_recent)

		chatmessage, found := chatmessages[i]
		if found {
			fmt.Printf("%v. %v: %v                     likes: %v\n", i, chatmessage.MessagedBy.Name, chatmessage.Message, len(chatmessage.LikedBy))
		}
		print_recent--
	}

}

// printall
func PrintAll(group *pb.Group) {
	groupname := group.GetGroupname()
	participants := maps.Values(group.GetParticipants())
	chatmessages := group.GetMessages()
	chatlength := len(chatmessages)
	fmt.Printf("Group: %v\n", groupname)
	fmt.Printf("Participants: %v\n", participants)
	count := 0
	for count < chatlength {
		i := uint32(count)

		chatmessage, found := chatmessages[i]
		if found {
			fmt.Printf("%v. %v: %v                     likes: %v\n", i, chatmessage.MessagedBy.Name, chatmessage.Message, len(chatmessage.LikedBy))
		}
		count++
	}

}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return logError(status.Error(codes.Canceled, "request is canceled"))
	case context.DeadlineExceeded:
		return logError(status.Error(codes.DeadlineExceeded, "deadline is exceeded"))
	default:
		return nil
	}
}

func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}

func (client *ChatServiceClient) ConnectionHealthCheck(conn *grpc.ClientConn) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		client.UserLogout()
		os.Exit(1)
	}()
	go func() {
		for {
			state := conn.GetState().String()
			if state != "READY" {
				log.Printf("Server Disconnected")
				os.Exit(1)
			}

		}
	}()
}
