package service

import (
	"bufio"
	"chat-system/pb"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

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
func UserLogout(client *ChatServiceClient) bool {
	user := client.clientstore.GetUser()
	req := &pb.LogoutRequest{
		User: &pb.Logout{
			User: user,
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
	log.Printf("User %v Logged out succesfully.", user.Name)
	return resp.Status
}

// experiment rpc
func (client *ChatServiceClient) JoinGroupChat() {

	ctx := context.Background()
	stream, err := client.chatservice.JoinGroupChat(ctx)
	if err != nil {
		log.Fatalf("open stream error %v", err)
	}

	done := make(chan bool)

	//go routine for send
	go func() error {
		for {
			log.Printf("Enter the message in the group:")
			msg, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				log.Fatalf("Cannot read the message, please enter again\n")
			}
			//preparing request
			req, command := client.ProcessMessage(msg)

			if err := stream.Send(req); err != nil {
				log.Println("Send request error")
			}
			if command == "q" {
				if err := stream.CloseSend(); err != nil {
					log.Println(err)
				}
			}

		}
	}()

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
				log.Fatalf("cannot receive %v", err)
			}

			//process response
			command := resp.Command
			if command == "p" {
				PrintAll(resp.Group)
			} else {
				PrintRecent(resp.Group)
			}

		}
	}()

	// go routine to close the done channel
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		// close(done)
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

	req := &pb.GroupChatRequest{}

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
						LikedBy:    make(map[uint32]string, 0),
					},
				},
			}
			req = &pb.GroupChatRequest{
				Action: appendchat,
			}
			log.Printf("appended a message in the group")

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
			req = &pb.GroupChatRequest{
				Action: likemessage,
			}

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
		req = &pb.GroupChatRequest{
			Action: unlikemessage,
		}

	case "p":
		{
			print := &pb.GroupChatRequest_Print{
				Print: &pb.PrintChat{
					User:      client.clientstore.GetUser(),
					Groupname: client.clientstore.GetGroup().Groupname,
				},
			}
			req = &pb.GroupChatRequest{
				Action: print,
			}
		}

	case "j":
		//join the group
		joinchat := &pb.GroupChatRequest_Joinchat{
			Joinchat: &pb.JoinChat{
				User:      client.clientstore.GetUser(),
				Newgroup:  msg,
				Currgroup: client.clientstore.GetGroup().Groupname,
			},
		}
		req = &pb.GroupChatRequest{
			Action: joinchat,
		}

	//quit the program
	case "q":
		logout := &pb.GroupChatRequest_Logout{
			Logout: &pb.Logout{
				User: client.clientstore.GetUser(),
			},
		}
		req = &pb.GroupChatRequest{
			Action: logout,
		}

	default:
		log.Printf("Cannot read the message, please enter again")
	}
	return req, cmd
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
