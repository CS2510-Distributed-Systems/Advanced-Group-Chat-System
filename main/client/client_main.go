package main

import (
	"bufio"
	client "chat-system/chat_client"
	"chat-system/pb"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	// "time"
	"strings"

	"google.golang.org/grpc"

	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	for {
		log.Println("Please specify server address to connect: ")
		msg, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Println("Error while reading command. Please try again.")
			continue
		}
		msg = strings.Trim(msg, "\r\n")
		argument := strings.Split(msg, " ")
		command := argument[0]
		address := ""
		if len(argument) > 1 {
			address = argument[1]
		}
		s := make(chan int, 1)
		
		switch command {
		case "c":
			if strings.TrimSpace(address) == "" {
				log.Println("Please specify address to connect")
			} else {
				server_address := strings.Split(address, ":")
				address := server_address[0]
				port := server_address[1]
				connectClient(address, port)
				continue

			}
		case "q":
			log.Println("closed the program")
			return
		default:
			log.Println("Please provide correct command to proceed.")
			continue
		}

	}

}

// client
func connectClient(server_address string, port string) {
	log.Printf("Dialing to server %s:%v", server_address, port)

	// Connect to RPC server
	transportOption := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.Dial(server_address+":"+port, transportOption)
	if err != nil {
		log.Fatal("cannot dial the server", err)
		return
	}
	serverId := int(server_address[len(server_address)-1])

	log.Printf("Dialing to server %s:%v", server_address, port)
	log.Printf("Target : %v :%v ", conn.GetState(), conn.Target())
	clientstore := client.NewInMemoryClientStore()
	chatclient := client.NewChatServiceClient(pb.NewChatServiceClient(conn), pb.NewAuthServiceClient(conn), clientstore, serverId)
	time.Sleep(20 * time.Millisecond)
	//Handling client crashing
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		chatclient.UserLogout()
		os.Exit(1)
	}()
	go func() {
		for {
			state := conn.GetState().String()
			log.Printf("state %v:", state)
			if state != "READY" {
				log.Printf("Iside the state shutdown")
				break

			}

		}
	}()

	for {
		//read input
		log.Printf("Enter the message:")
		msg, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Println("Client crashed. Performing graceful shutdown")
			return
		}

		//parsing input
		msg = strings.Trim(msg, "\r\n")
		args := strings.Split(msg, " ")
		cmd := strings.TrimSpace(args[0])
		arg := strings.Join(args[1:], " ")

		switch cmd {
		case "u":
			username := strings.TrimSpace(arg)
			if username == "" {
				log.Println("Please provide a valid user name")
			} else {
				err := client.UserLogin(username, chatclient)
				if err != nil {
					fmt.Println(err)
					return
				}
			}

		case "j":
			if clientstore.GetUser().GetName() == "" {
				log.Println("Please login to join a group.")
			} else {
				//join the group
				groupname := strings.TrimSpace(arg)
				if groupname == "" {
					log.Println("Please provide a valid group name")
				}
				chatclient.JoinGroupChat(msg)
				return

			}
		case "a", "l", "r":
			log.Println("please Enter the chat group to perform chat operations")

		case "q":
			if chatclient.UserLogout() {
				log.Println("Connection closed.")
				conn.Close()
				return
			} else {
				log.Println("Logout Error.")
			}
		case "v":
			chatclient.ServerView()
		default:
			log.Printf("incorrect command, please enter again\n")
		}
	}

}
