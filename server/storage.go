package chat_server

import (
	"bytes"
	"chat-system/pb"
	"encoding/gob"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	raftbadger "github.com/rfyiamcool/raft-badger"
)

type Storage interface {
	//user data disk storage methods
	SaveUser(user *pb.User) error
	DeleteUser(userID uint32)

	//group data disk storage methods
	GetGroup(string) (*pb.Group, bool)
	JoinGroup(string, user *pb.User)
	AppendMessage(*pb.AppendChat)
	LikeMessage(*pb.LikeMessage)
	UnLikeMessage(*pb.UnLikeMessage)
	RemoveUser(uint32, string)
	RemoveUserInGroup(uint32, string) bool
	UpdateGroup(string, *pb.Group) bool

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

func NewDiskStore(serverId int64) *DiskStore {
	cfg_appdata := raftbadger.Config{
		DataPath: "server/diskstore/server" + strconv.Itoa(int(serverId)),
	}
	diskstore, err := raftbadger.New(cfg_appdata, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create raft badger storage, err: %s", err.Error()))
	}
	//config for raft
	cfg_raftdata := raftbadger.Config{
		DataPath: "server/diskstore/server" + strconv.Itoa(int(serverId)) + "/raft",
	}
	raftdiskstore, err := raftbadger.New(cfg_raftdata, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create raft badger storage, err: %s", err.Error()))
	}
	RegisterCommands()
	return &DiskStore{
		diskstore:     diskstore,
		replicatedlog: raftdiskstore,
	}
}

func RegisterCommands() {
	gob.Register(&pb.Command{})
	gob.Register(&pb.Command_Append{})
	gob.Register(&pb.Command_Joinchat{})
	gob.Register(&pb.Command_Like{})
	gob.Register(&pb.Command_Unlike{})
	gob.Register(&pb.Command_Print{})
	gob.Register(&pb.Command_Login{})
	gob.Register(&pb.Command_Logout{})

}

func (store *DiskStore) SaveUser(user *pb.User) error {
	usercopy := &pb.User{}
	err := copier.Copy(usercopy, user)
	if err != nil {
		return fmt.Errorf("error while deepcopy user: %w", err)
	}

	var encoder bytes.Buffer
	// gob.Register(pb.User)
	if err := gob.NewEncoder(&encoder).Encode(usercopy); err != nil {
		log.Fatal(err)
	}
	err = store.diskstore.Set([]byte(strconv.Itoa(int(usercopy.Id))), encoder.Bytes())
	checkError(err)

	log.Printf("user %v logged in the server.User stored in the disk", user.GetName())
	return nil
}

func (store *DiskStore) GetUser(userId uint32) *pb.User {
	store.mu.Lock()
	defer store.mu.Unlock()
	var user *pb.User
	if usr, err := store.diskstore.Get([]byte(strconv.Itoa(int(userId)))); err == nil {
		d := gob.NewDecoder(bytes.NewBuffer(usr))
		if err := d.Decode(&user); err != nil {
			return user
		}
	}
	return user
}

func (store *DiskStore) DeleteUser(userId uint32) error {
	err := store.diskstore.Delete([]byte(strconv.Itoa(int(userId))))
	checkError(err)

	return nil
}

func (store *DiskStore) GetGroup(groupname string) (*pb.Group, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	var group *pb.Group
	if grp, err := store.diskstore.Get([]byte(groupname)); err == nil {
		d := gob.NewDecoder(bytes.NewBuffer(grp))
		if err := d.Decode(&group); err != nil {
			return group, false
		}
	} else {
		log.Println("group not found in storage")
		return nil, false
	}
	return group, true
}

func (store *DiskStore) UpdateGroup(groupname string, group *pb.Group) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	groupcopy := &pb.Group{}
	err := copier.Copy(groupcopy, group)
	if err != nil {
		return false
	}
	log.Printf("Deep copy of group is %v", groupcopy)
	var encoder bytes.Buffer
	if err := gob.NewEncoder(&encoder).Encode(groupcopy); err != nil {
		log.Fatal(err)
	}
	err = store.diskstore.Set([]byte(groupname), encoder.Bytes())
	checkError(err)
	log.Println("Group info updated successfully in the disk")
	return true
}

func (store *DiskStore) JoinGroup(groupname string, user *pb.User) {

	group, found := store.GetGroup(groupname)
	if found {
		group.Participants[user.GetId()] = user.GetName()
		log.Printf("updated group with the participants %v", group)
		if store.UpdateGroup(groupname, group) {
			log.Printf("user %v joined %v group", user.GetName(), groupname)
			return
		}

	}
	//if not found create one
	new_group := &pb.Group{
		GroupID:      uuid.New().ID(),
		Groupname:    groupname,
		Participants: make(map[uint32]string),
		Messages:     make(map[uint32]*pb.ChatMessage),
	}
	new_group.Participants[user.GetId()] = user.GetName()
	log.Printf("Added participant to %v", new_group)
	if store.UpdateGroup(groupname, new_group) {
		log.Printf("user %v joined %v group", user.GetName(), groupname)
		return
	} else {
		log.Println(" Failed to update the group")
	}
}

func (store *DiskStore) RemoveUserInGroup(userID uint32, groupname string) bool {
	group, found := store.GetGroup(groupname)
	if found {
		if group.Participants == nil || group.Participants[userID] == "" {
			return true
		}
		delete(group.Participants, userID)
		if store.UpdateGroup(groupname, group) {
			log.Printf("removed user from current group if any")
			return true

		}
	} else {
		log.Printf("Group is not present in disk yet.")
		return true
	}
	return false
}

func (store *DiskStore) SetState(currentTerm int64, votedFor int64, logs []*pb.LogEntry) error {
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
	// log.Printf("Trying to persist data2..")
	gob.Register(logs)
	if err := gob.NewEncoder(&logData).Encode(logs); err != nil {
		log.Fatal(err)
	}
	// log.Printf("Trying to persist data3..")
	store.replicatedlog.Set([]byte("log"), logData.Bytes())

	return nil
}

func (store *DiskStore) GetState() (int64, int64, []*pb.LogEntry) {
	store.mu.Lock()
	defer store.mu.Unlock()

	var term int64
	if termData, err := store.replicatedlog.Get([]byte("currentTerm")); err == nil {
		d := gob.NewDecoder(bytes.NewBuffer(termData))
		if err := d.Decode(&term); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("currentTerm not found in storage")
	}
	var votedFor int64
	if votedData, err := store.replicatedlog.Get([]byte("votedFor")); err == nil {
		d := gob.NewDecoder(bytes.NewBuffer(votedData))
		if err := d.Decode(&votedFor); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("votedFor not found in storage")
	}
	var logs []*pb.LogEntry
	if logData, err := store.replicatedlog.Get([]byte("log")); err == nil {
		d := gob.NewDecoder(bytes.NewBuffer(logData))
		if err := d.Decode(&logs); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("log not found in storage")
	}

	return term, votedFor, logs
}

func (store *DiskStore) HasData() bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	var logs []LogEntry
	if logData, err := store.replicatedlog.Get([]byte("log")); err == nil {
		d := gob.NewDecoder(bytes.NewBuffer(logData))
		if err := d.Decode(&logs); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("log not found in storage")
	}

	return len(logs) > 0
}

func (store *DiskStore) AppendMessage(appendchat *pb.AppendChat) {
	groupname := appendchat.Group.Groupname
	message := appendchat.Chatmessage
	group, found := store.GetGroup(groupname)
	if found {
		appendmessagenumber := len(group.Messages)
		group.Messages[uint32(appendmessagenumber)] = message
		log.Printf("group %v: ", group)
		if store.UpdateGroup(groupname, group) {
			log.Printf("Appended new message in the group %v", group.Groupname)
			return
		}

	}
}

func (store *DiskStore) LikeMessage(likemessage *pb.LikeMessage) {
	groupname := likemessage.Group.Groupname
	group, found := store.GetGroup(groupname)
	if found {

		likemessagenumber := likemessage.Messageid
		//validate the message
		message, found := group.Messages[likemessagenumber]
		if !found {
			log.Printf("Message not found")
			return
		}
		if message.MessagedBy.Name == likemessage.Likeduser.Name {
			log.Printf("Cannot Like own message")
			return
		}
		likedusername, found := message.LikedBy[likemessage.Likeduser.Name]
		if found {
			log.Printf("already Liked the message")
			return
		}

		//like
		if len(message.LikedBy) == 0 {
			message.LikedBy = make(map[string]string)
		}
		message.LikedBy[likemessage.Likeduser.Name] = likedusername

		if store.UpdateGroup(groupname, group) {
			log.Printf("Liked message in the group %v", group.Groupname)
			return
		}

	}
}

func (store *DiskStore) UnLikeMessage(unlikemessage *pb.UnLikeMessage) {
	groupname := unlikemessage.Group.Groupname
	group, found := store.GetGroup(groupname)
	if found {
		unlikemessagenumber := unlikemessage.Messageid
		//validate the message
		message, found := group.Messages[unlikemessagenumber]
		if !found {
			log.Printf("Message not found")
			return
		}
		if message.MessagedBy.Name == unlikemessage.Unlikeduser.Name {
			log.Printf("Cannot unLike own message")
			return
		}
		_, found = message.LikedBy[unlikemessage.Unlikeduser.Name]
		if !found {
			log.Printf("its not liked")
			return
		}

		//unlike
		delete(message.LikedBy, unlikemessage.Unlikeduser.Name)
		if store.UpdateGroup(groupname, group) {
			log.Printf("Unliked message in the group %v", group.Groupname)
			return
		}

	}
}
func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
