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
	//badger storage
)

func main() {
	// Process commandline arguments
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
	//create a cmux to multiplex between GRPC and net/rpc requests
	mux := cmux.New(listener)
	//match connections in order
	// grpcL := mux.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	//first grpc, then go rpc
	trpcL := mux.Match(cmux.Any())
	//The gRPC server
	grpcserver := grpc.NewServer()
	//the go rpc (raft) server
	raftserver := service.NewServer(serverId, trpcL)

	//register the services
	groupstore := service.NewInMemoryGroupStore()
	clients := service.NewInMemoryConnStore()
	userstore := service.NewInMemoryUserStore()
	chatserver := service.NewChatServiceServer(groupstore, userstore, clients, raftserver)
	pb.RegisterChatServiceServer(grpcserver, chatserver)
	pb.RegisterAuthServiceServer(grpcserver, chatserver)

	//Start Serving

	go grpcserver.Serve(trpcL)
	go raftserver.Serve()

	mux.Serve()
	//wait for all go routines to end
	wg.Wait()

}
