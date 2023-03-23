package main

import (
	"chat-system/pb"
	"chat-system/service"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"
)

func main() {
	// Process commandline arguments
	address := flag.String("Address", "0.0.0.0", "server address")
	portArg := flag.Int("port", 12000, "the server port")
	flag.Parse()
	port := *portArg

	//register the server
	grpcserver := grpc.NewServer()
	groupstore := service.NewInMemoryGroupStore()
	clients := service.NewInMemoryConnStore()
	userstore := service.NewInMemoryUserStore()
	chatserver := service.NewChatServiceServer(groupstore, userstore, clients)

	pb.RegisterChatServiceServer(grpcserver, chatserver)
	pb.RegisterAuthServiceServer(grpcserver, chatserver)

	log.Printf("start server on port: %d", port)
	Listener, err := net.Listen("tcp", *address+":"+strconv.Itoa(port))
	if err != nil {
		log.Fatal("cannot start server: %w", err)
	}
	log.Printf("Start GRPC server at %s", Listener.Addr())

	err = grpcserver.Serve(Listener)
	fmt.Println(err)
	if err != nil {
		return
	}

}
