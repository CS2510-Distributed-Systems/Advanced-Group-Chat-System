package main

import (
	"chat-system/pb"
	service "chat-system/server"
	"flag"
	"log"

	"google.golang.org/grpc"
)

func main() {
	// Process commandline arguments
	// address := flag.String("Address", "localhost", "server address")
	// portArg := flag.Int("port", 12000, "the server port")
	Id := flag.String("id", "0", "server ID")
	flag.Parse()

	IP_BASE := "localhost"
	port := ":1200" + *Id

	IP := IP_BASE + port

	//register the server
	grpcserver := grpc.NewServer()
	groupstore := service.NewInMemoryGroupStore()
	clients := service.NewInMemoryConnStore()
	userstore := service.NewInMemoryUserStore()
	server := new(service.Server)
	chatserver := service.NewChatServiceServer(groupstore, userstore, clients, server)

	pb.RegisterChatServiceServer(grpcserver, chatserver)
	pb.RegisterAuthServiceServer(grpcserver, chatserver)

	log.Printf("start server on port: %v", IP)
	server.Serve(IP)

}

// func main() {
// 	//commandline for the server ID
// 	// Id := flag.String("id", "0", "server ID")
// 	// flag.Parse()
// 	// serverId, _ := strconv.Atoi(*Id)

// 	IP_BASE := "localhost"
// 	port := ":1200"
// 	IP := IP_BASE + port

// 	// ip := address + ":" + strconv.Itoa(port)
// 	//get the peerIds

// 	server := new(raft.Server)
// 	err := server.StartCluster(5, IP)
// 	if err != nil {
// 		return
// 	}

// }
