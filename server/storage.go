package chat_server

import (
	"bytes"
	"chat-system/pb"
	"encoding/gob"
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

	//raft replication log storage methods
	SetState(int, int, []LogEntry) error
	GetState() (int, int, []LogEntry)
	//has data returns true if any sets were made on this storage
	HasData() bool
}

type DiskStore struct {
	mu            sync.Mutex
	diskstore     *raftbadger.Storage
	replicatedlog *raftbadger.Storage
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

func (store *DiskStore) Set(newlog *raft.Log, votedFor uint64) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.replicatedlog.StoreLog(newlog)

	store.replicatedlog.SetUint64([]byte("votedFor"), votedFor)
	return nil
}

func (store *DiskStore) GetState() (int, int, []LogEntry) {
	store.mu.Lock()
	defer store.mu.Unlock()

	var term int
	if termData, err := store.replicatedlog.Get([]byte("currentTerm")); err == nil {
		d := gob.NewDecoder(bytes.NewBuffer(termData))
		if err := d.Decode(term); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("currentTerm not found in storage")
	}
	var votedFor int
	if votedData, err := store.replicatedlog.Get([]byte("votedFor")); err == nil {
		d := gob.NewDecoder(bytes.NewBuffer(votedData))
		if err := d.Decode(votedFor); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("votedFor not found in storage")
	}
	var logs []LogEntry
	if logData, err := store.replicatedlog.Get([]byte("log")); err == nil {
		d := gob.NewDecoder(bytes.NewBuffer(logData))
		if err := d.Decode(logs); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("log not found in storage")
	}

	return term, votedFor, logs
}

func (store *DiskStore) SetState(currentTerm int, votedFor int, logs []LogEntry) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	//store term data
	var termData bytes.Buffer
	if err := gob.NewEncoder(&termData).Encode(currentTerm); err != nil {
		log.Fatal(err)
	}
	err := store.replicatedlog.Set([]byte("currentTerm"), termData.Bytes())
	checkError(err)
	//store votedFor data
	var votedData bytes.Buffer
	if err := gob.NewEncoder(&votedData).Encode(votedFor); err != nil {
		log.Fatal(err)
	}
	store.replicatedlog.Set([]byte("votedFor"), votedData.Bytes())
	//store the entire raftlog
	var logData bytes.Buffer
	if err := gob.NewEncoder(&logData).Encode(logs); err != nil {
		log.Fatal(err)
	}
	store.replicatedlog.Set([]byte("log"), logData.Bytes())

	return nil
}
func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
