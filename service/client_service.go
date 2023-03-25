package service

import (
	// "bufio"
	"bufio"
	"chat-system/pb"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	// "os"
	"strconv"
	"strings"

	// "time"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"
)

// client service
type ChatServiceClient struct {
	chatservice pb.ChatServiceClient
	authservice pb.AuthServiceClient
	clientstore ClientStore
}

func NewChatServiceClient(chatservice pb.ChatServiceClient, authservice pb.AuthServiceClient, store ClientStore) *ChatServiceClient {
	return &ChatServiceClient{
		chatservice: chatservice,
		authservice: authservice,
		clientstore: store,
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
			Groupname: groupname,
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

// experiment rpc
func (client *ChatServiceClient) JoinGroupChat(mesg string) {

	ctx := context.Background()

	stream, err := client.chatservice.JoinGroupChat(ctx)
	if err != nil {
		log.Fatalf("open stream error %v", err)
	}

	done := make(chan bool)
	req1, _ := client.ProcessMessage(mesg)
	if err := stream.Send(req1); err != nil {
		log.Println("Send request error")
		return
	}
	//go routine for receive
	go func() {
		for {
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
			command := resp.Command
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
			} else {
				log.Println("Logout Error.")
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
			}

		}
	}()

	<-done
	log.Println("Streaming Ended from client side")

}

func (client *ChatServiceClient) ProcessMessage(msg string) (*pb.GroupChatRequest, string) {
	log.Println("Processing Message..")
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
			log.Printf("appended a message in the group")
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
					User:      client.clientstore.GetUser(),
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
				User:      client.clientstore.GetUser(),
				Messageid: uint32(messagenumber),
				Group:     client.clientstore.GetGroup(),
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
		log.Println("Joining the group and starting the stream")
		currgroup := "None"
		if client.clientstore.GetGroup().GroupID != 0 {
			currgroup = client.clientstore.GetGroup().Groupname
		}
		joinchat := &pb.GroupChatRequest_Joinchat{
			Joinchat: &pb.JoinChat{
				User:      client.clientstore.GetUser(),
				Newgroup:  msg,
				Currgroup: currgroup,
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

	default:
		log.Printf("Cannot read the message, please enter again")

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
