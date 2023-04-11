package main

import (
	"chat-system/pb"
	service "chat-system/server"
	"flag"
	"log"
	"net"
	"strconv"
	"sync"

	"google.golang.org/grpc"
	//badger storag
)

func main() {
	// Process commandline argument
	// address := flag.String("Address", "localhost", "server address")
	// portArg := flag.Int("port", 12000, "the server port")
	Id := flag.Int("id", 1, "server ID")
	flag.Parse()

	//construct the IP
	serverId := *Id
	serverId_string := strconv.Itoa(serverId)
	IP_BASE := "0.0.0.0"
	port := ":1200" + serverId_string
	// port = ":12000"
	IP := IP_BASE + port

	var wg sync.WaitGroup

	//Initialize the Listener and the servers
	listener, err := net.Listen("tcp", IP)
	if err != nil {
		log.Fatal(err)
	}

	grpcserver := grpc.NewServer()

	//the go rpc (raft) server
	raftserver := service.NewServer(int64(serverId), listener)

	//register the services
	clients := service.NewInMemoryConnStore()
	chatserver := service.NewChatServiceServer(clients, raftserver)
	pb.RegisterChatServiceServer(grpcserver, chatserver)
	pb.RegisterAuthServiceServer(grpcserver, chatserver)
	pb.RegisterRaftServiceServer(grpcserver, raftserver)

	//Start Serving
	raftserver.Serve()
	grpcserver.Serve(listener)

	//wait for all go routines to end
	wg.Wait()

}
