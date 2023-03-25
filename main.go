package main

import (
	// "bufio"
	client "chat-system/cmd/client"
	"log"
	// "os"
	"strings"
)

func main() {
	log.Println("Please specify server address to connect: ")
	for {
		msg := "c localhost:12000"
		argument := strings.Split(msg, " ")
		command := argument[0]
		address := ""
		if len(argument) > 1 {
			address = argument[1]
		}
		switch command {
		case "c":
			if strings.TrimSpace(address) == "" {
				log.Println("Please specify address to connect")
			} else {
				server_address := strings.Split(address, ":")
				address := server_address[0]
				port := server_address[1]
				client.CallClient(address, port)
			}
		case "q":
			log.Println("closed the program")
			return
		default:
			log.Println("Please provide correct command to proceed.")
		}

	}
}
