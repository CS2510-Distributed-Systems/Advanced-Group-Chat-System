package chat_server

import (
	"chat-system/pb"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/hashicorp/raft"
	"github.com/jinzhu/copier"
	raftbadger "github.com/rfyiamcool/raft-badger"
	"google.golang.org/protobuf/proto"
)

type Storage interface {
	//user data disk storage methods
	SaveUser(user *pb.User) error
	DeleteUser(userID uint32)

	//group data disk storage methods
	GetGroup(groupname string) *pb.Group
	JoinGroup(groupname string, user *pb.User) (*pb.Group, error)
	AppendMessage(appendchat *pb.AppendChat) error
	LikeMessage(like *pb.LikeMessage) error
	UnLikeMessage(unlike *pb.UnLikeMessage) error
	RemoveUser(userID uint32, groupname string)
}

type ReplicatedLog interface {
	Set(key string, value []byte)

	Get(key string) ([]byte, bool)

	//has data returns true if any sets were made on this storage
	HasData() bool
}

type DiskStore struct {
	mu        sync.Mutex
	diskstore *raftbadger.Storage
}

type replicatedlog struct {
	mu            sync.Mutex
	replicatedlog *raft.Log
}

func (store *DiskStore) SaveUser(user *pb.User) error {
	usercopy := &pb.User{}
	err := copier.Copy(usercopy, user)
	if err != nil {
		return fmt.Errorf("error while deepcopy user: %w", err)
	}

	newuser, err := proto.Marshal(usercopy)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}

	Id := usercopy.GetId()
	err = store.diskstore.Set([]byte(strconv.Itoa(int(Id))), newuser)
	checkError(err)

	log.Printf("user %v logged in the server.User stored in the disk", user.GetName())
	return nil
}

func (log *replicatedlog) Set(key string, value []byte) {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.Set(key, value)
}

func (log *replicatedlog) Get(key string) ([]byte, bool) {
	log.mu.Lock()
	defer log.mu.Unlock()
	v, found := log.Get(key)
	return v, found
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
