package main

import (
	"chat-system/pb"
	service "chat-system/server"
	"flag"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
)

func main() {
	// Process commandline arguments
	// address := flag.String("Address", "localhost", "server address")
	// portArg := flag.Int("port", 12000, "the server port")
	Id := flag.Int("id", 1, "server ID")
	flag.Parse()

	//construct the IP
	serverId := *Id
	IP_BASE := "0.0.0.0"
	port := ":1200" + strconv.Itoa(serverId)
	port = ":12000"
	IP := IP_BASE + port

	//

	//create the main listener
	listener, err := net.Listen("tcp", IP)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	//create a cmux
	mux := cmux.New(listener)

	//match connections in order
	//first grpc, then go rpc
	// grpcL := mux.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	trpcL := mux.Match(cmux.Any())

	//The gRPC server
	grpcserver := grpc.NewServer()

	//the go rpc (raft) server
	raftserver := service.NewServer(serverId, trpcL)

	//initialize and register the chatserver
	groupstore := service.NewInMemoryGroupStore()
	clients := service.NewInMemoryConnStore()
	userstore := service.NewInMemoryUserStore()
	chatserver := service.NewChatServiceServer(groupstore, userstore, clients, raftserver)
	pb.RegisterChatServiceServer(grpcserver, chatserver)
	pb.RegisterAuthServiceServer(grpcserver, chatserver)

	//use the muxed listeners for your servers
	go grpcserver.Serve(listener)
	raftserver.Serve()

	//start serving
	mux.Serve()

	//wait for go routines to end
	wg.Wait()
}
