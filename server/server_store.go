package chat_server

import (
	"chat-system/pb"
	"log"
	"sync"
)

// stores the incoming connection so as to braodcast later
type ConnStore interface {
	BroadCast(groupname string, resp *pb.GroupChatResponse) error
	AddConn(stream pb.ChatService_JoinGroupChatServer, client [2]string)
	RemoveConn(client [2]string)
}

type InMemoryConnStore struct {
	mutex   sync.RWMutex
	clients map[pb.ChatService_JoinGroupChatServer][2]string
}

func NewInMemoryConnStore() *InMemoryConnStore {
	return &InMemoryConnStore{
		clients: make(map[pb.ChatService_JoinGroupChatServer][2]string),
	}
}

// adds an incoming conn in the server connstore
func (conn *InMemoryConnStore) AddConn(stream pb.ChatService_JoinGroupChatServer, client [2]string) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if conn.clients == nil {
		conn.clients = make(map[pb.ChatService_JoinGroupChatServer][2]string)
	}
	currclient, found := conn.clients[stream]
	if found && currclient == client {
		log.Printf("Client already present")
		return
	}
	conn.clients[stream] = client
	log.Printf("Client added")
}

// removes an incoming conn in the server connstore
func (conn *InMemoryConnStore) RemoveConn(removeclient [2]string) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if conn.clients == nil {
		log.Printf("No connection present")
	}
	for stream, client := range conn.clients {
		if removeclient == client {
			delete(conn.clients, stream)
			return
		}
	}
	log.Println("No record found in connection store")
}
