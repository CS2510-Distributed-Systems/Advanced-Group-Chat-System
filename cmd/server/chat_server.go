package main

import (
	"chat-system/pb"
	"chat-system/raft"
	"chat-system/service"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"
)

// func main() {
// 	// Process commandline arguments
// 	address := flag.String("Address", "localhost", "server address")
// 	portArg := flag.Int("port", 12000, "the server port")
// 	flag.Parse()
// 	port := *portArg

// 	//register the server
// 	grpcserver := grpc.NewServer()
// 	groupstore := service.NewInMemoryGroupStore()
// 	clients := service.NewInMemoryConnStore()
// 	userstore := service.NewInMemoryUserStore()
// 	chatserver := service.NewChatServiceServer(groupstore, userstore, clients)

// 	pb.RegisterChatServiceServer(grpcserver, chatserver)
// 	pb.RegisterAuthServiceServer(grpcserver, chatserver)

// 	log.Printf("start server on port: %d", port)
// 	Listener, err := net.Listen("tcp", *address+":"+strconv.Itoa(port))
// 	if err != nil {
// 		log.Fatal("cannot start server: %w", err)
// 	}
// 	log.Printf("Start GRPC server at %s", Listener.Addr())

// 	err = grpcserver.Serve(Listener)
// 	fmt.Println(err)
// 	if err != nil {
// 		return
// 	}

// }

func main() {
	//commandline for the server ID
	Id := flag.String("id","0","server ID" )
	flag.Parse()
	serverId,_ := strconv.Atoi(*Id)

	address := "0.0.0.0"
	port := 12000

	//get the peerIds
	peerIds := make([]int, 0)
	for p:= 1; p <= 5; p++ {
		if p != serverId {
			peerIds = append(peerIds, p)
		}
	}
	ready := make(chan interface{})
	
	server := raft.NewServer(serverId, peerIds,ready )
	server.Serve()
}