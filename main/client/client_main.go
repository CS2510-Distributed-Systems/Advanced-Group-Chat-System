package main

import (
	"bufio"
	client "chat-system/chat_client"
	"chat-system/pb"
	"fmt"
	"log"
	"os"
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

		switch command {
		case "c":
			address := strings.TrimSpace(address)
			if !strings.Contains(address, ":") {
				log.Println("Incorrect server address.")
			} else {
				server_address := strings.Split(address, ":")
				ip := server_address[0]
				port := server_address[1]
				ConnectClient(ip, port)
				continue

			}
		case "q":
			log.Println("closed the program")
			return
		}

	}

}

// client
func ConnectClient(ip string, port string) {
	log.Printf("Dialing to server %s:%v", ip, port)

	// Connect to RPC server
	transportOption := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.Dial(ip+":"+port, transportOption)
	if err != nil {
		log.Fatal("cannot dial the server", err)
		return
	}
	serverId := int(ip[len(ip)-1])
	clientstore := client.NewInMemoryClientStore()
	chatclient := client.NewChatServiceClient(pb.NewChatServiceClient(conn), pb.NewAuthServiceClient(conn), clientstore, serverId)
	time.Sleep(3 * time.Second)
	Event := "Login: "
	for {
		state := conn.GetState().String()
		if state != "READY" {
			log.Printf("Cannot connect to the server.")
			return

		}

		//Handling client crashing amd server crashing
		go chatclient.ConnectionHealthCheck(conn)

		//read input
		log.Printf("%v", Event)
		msg, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Println("Client crashed. Performing graceful shutdown")
			chatclient.UserLogout()

			os.Exit(1)
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
				Event = "Enter the Group :"
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
