package service

import (
	"chat-system/pb"
	"context"
	"io"
	"log"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	// "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// chat service
type ChatServiceServer struct {
	pb.UnimplementedChatServiceServer
	pb.UnimplementedAuthServiceServer
	groupstore GroupStore
	userstore  UserStore
	clients    ConnStore
}

func NewChatServiceServer(groupstore GroupStore, userstore UserStore, clients ConnStore) *ChatServiceServer {
	return &ChatServiceServer{
		groupstore: groupstore,
		userstore:  userstore,
		clients:    clients,
	}
}

func (s *ChatServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user_name := req.User.Name
	log.Printf("Logging as: %v", user_name)
	newUser := &pb.User{
		Id:   uuid.New().ID(),
		Name: user_name,
	}
	s.userstore.SaveUser(newUser)
	res := &pb.LoginResponse{
		User: req.GetUser(),
	}

	return res, nil
}

// logout rpc
func (s *ChatServiceServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	username := req.Logout.User.Name
	userID := req.Logout.User.Id
	groupname := req.Logout.Groupname
	//delete user from userlist
	s.userstore.DeleteUser(userID)

	//delete user from grouplist
	if groupname != "" {
		s.groupstore.RemoveUser(userID, groupname)
		log.Printf("Removed %v from %v group", username, groupname)
		currclient := [2]string{groupname, username}

		//remove the current client stream
		s.clients.RemoveConn(currclient)
		
		//braodcast the change
		broadcastresp := &pb.GroupChatResponse{
			Group:   s.groupstore.GetGroup(groupname),
			Command: "q",
		}

		s.clients.BroadCast(groupname, broadcastresp)
	}
	resp := &pb.LogoutResponse{
		Status: true,
	}

	log.Printf("Removed %v from active-users", username)
	return resp, nil
}

// Experiment to merge JoinGroup and Groupchat rpc's
func (s *ChatServiceServer) JoinGroupChat(stream pb.ChatService_JoinGroupChatServer) error {
	log.Println("Received Join Group Request")

	for {
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		//receive data
		log.Println("stream waiting to receive data....")
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream ended in the server side")
			return err
		}
		if err != nil {
			log.Println("Error in Receive.")
			return err
		}

		//processing received request
		groupname, command := s.ProcessRequest(req)

		//adding the new client
		if command == "j" {
			user := req.GetJoinchat().User

			//store the stream details
			newclient := [2]string{groupname, user.Name}
			s.clients.AddConn(stream, newclient)
		}

		//prepare a response
		resp := &pb.GroupChatResponse{
			Group:   s.groupstore.GetGroup(groupname),
			Command: command,
		}

		//braodcast the change
		s.clients.BroadCast(groupname, resp)
		if err := stream.Send(resp); err != nil {
			log.Printf("send error %v", err)
		}

	}

}

// api to process the request received
func (s *ChatServiceServer) ProcessRequest(req *pb.GroupChatRequest) (string, string) {
	log.Println("Request received. Processing..")
	switch req.GetAction().(type) {
	case *pb.GroupChatRequest_Append:
		command := "a"
		appendchat := &pb.AppendChat{
			Group:       req.GetAppend().GetGroup(),
			Chatmessage: req.GetAppend().GetChatmessage(),
		}
		err := s.groupstore.AppendMessage(appendchat)
		if err != nil {
			log.Printf("cannot save the message %s", err)
		}
		return appendchat.Group.Groupname, command

	case *pb.GroupChatRequest_Like:
		command := "l"
		group := req.GetLike().Group
		msgId := req.GetLike().Messageid
		user := req.GetLike().User
		likemessage := &pb.LikeMessage{
			Group:     group,
			Messageid: msgId,
			User:      user,
		}
		err := s.groupstore.LikeMessage(likemessage)
		if err != nil {
			log.Printf("%s", err)
		}
		return likemessage.Group.Groupname, command

	case *pb.GroupChatRequest_Unlike:
		command := "r"
		group := req.GetUnlike().Group
		msgId := req.GetUnlike().Messageid
		user := req.GetUnlike().User
		unlikemessage := &pb.UnLikeMessage{
			Group:     group,
			Messageid: msgId,
			User:      user,
		}
		err := s.groupstore.UnLikeMessage(unlikemessage)
		if err != nil {
			log.Printf("some error occured in unliking the message: %s", err)
		}
		return unlikemessage.Group.Groupname, command

	case *pb.GroupChatRequest_Joinchat:
		command := "j"
		user := req.GetJoinchat().User
		currgroupname := req.GetJoinchat().Currgroup
		newgroupname := req.GetJoinchat().Newgroup
		currclient := [2]string{currgroupname, user.Name}

		//remove the current client stream
		s.clients.RemoveConn(currclient)

		//remove the user from the current group
		s.groupstore.RemoveUser(user.Id, currgroupname)

		// join a group
		_, err := s.groupstore.JoinGroup(newgroupname, user)
		if err != nil {
			log.Printf("Failed to join group %v", err)
		}
		log.Printf("Joined group %s", newgroupname)

		return newgroupname, command

	default:
		log.Printf("let the client enter the command")
	}

	return "", ""
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return logError(status.Error(codes.Canceled, "request is canceled"))
	case context.DeadlineExceeded:
		return logError(status.Error(codes.DeadlineExceeded, "deadline is exceeded"))
	default:
		return nil
	}
}

func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}
