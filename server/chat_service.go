package chat_server

import (
	"chat-system/pb"
	"context"
	"io"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// chat service
type ChatServiceServer struct {
	pb.UnimplementedChatServiceServer
	pb.UnimplementedAuthServiceServer
	connstore *InMemoryConnStore
	raft      *Server
}

func NewChatServiceServer(clients *InMemoryConnStore, raft *Server) *ChatServiceServer {
	chatserver := &ChatServiceServer{
		connstore: clients,
		raft:      raft,
	}
	go chatserver.BroadCast()
	return chatserver
}

func (s *ChatServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	//get the request payload
	user := req.User
	//construct raft command
	command := &pb.Command{
		Event: "u",
		Action: &pb.Command_Login{
			Login: user,
		},
	}

	resp := &pb.LoginResponse{}
	//submit to raft
	if s.raft.cm.Submit(command) {
		log.Printf("User %v logged in", user.Name)
		resp = &pb.LoginResponse{
			User: user,
		}
	}

	return resp, nil
}

// logout rpc
func (s *ChatServiceServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	//get the request payload
	user := req.Logout.User
	groupname := req.Logout.Currgroup

	//construct raft command
	command := &pb.Command{
		Event: "q",
		Action: &pb.Command_Logout{
			Logout: &pb.Logout{
				User:      user,
				Currgroup: groupname,
			},
		},
	}

	resp := &pb.LogoutResponse{}
	//submit to raft
	if s.raft.cm.Submit(command) {
		log.Printf("User %v logged out", user.Name)
		resp = &pb.LogoutResponse{
			Status: true,
		}
		s.raft.broadcast <- groupname
	}

	return resp, nil
}

// Group Join Bidirectional RPC
func (s *ChatServiceServer) JoinGroupChat(stream pb.ChatService_JoinGroupChatServer) error {
	log.Println("Received Join Group Request")

	for {
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		//receive data
		log.Println("Stream open to client requests")
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream ended.")
			return err
		}
		if err != nil {
			log.Println("Error occured. Closing the stream")
			return err
		}

		//processing received request
		groupname, event := s.ProcessRequest(req)

		//adding the new client
		//change1
		if event == "j" {
			user := req.GetJoinchat().Joineduser

			//store the stream details
			newclient := [2]string{groupname, user.Name}
			s.connstore.AddConn(stream, newclient)
		}

		//prepare a response
		time.Sleep(200 * time.Millisecond)
		group, _ := s.raft.cm.storage.GetGroup(groupname)
		resp := &pb.GroupChatResponse{
			Group: group,
			Event: event,
		}

		//braodcast the change
		s.raft.broadcast <- groupname
		if err := stream.Send(resp); err != nil {
			log.Printf("Error in send stream: %v", err)
		}
	}

}

// API to process the request received
func (s *ChatServiceServer) ProcessRequest(req *pb.GroupChatRequest) (string, string) {
	log.Println("Processing the request")
	switch req.GetAction().(type) {

	case *pb.GroupChatRequest_Append:
		event := "a"
		//get the request payload
		action := &pb.GroupChatRequest_Append{
			Append: req.GetAppend(),
		}

		//construct raft command
		command := &pb.Command{
			Event: event,
			Action: &pb.Command_Append{
				Append: action.Append,
			},
		}
		//submit to raft
		if s.raft.cm.Submit(command) {
			return command.GetAppend().Group.Groupname, event
		}

	case *pb.GroupChatRequest_Like:
		event := "l"
		//get the request payload
		action := &pb.GroupChatRequest_Like{
			Like: req.GetLike(),
		}

		//construct raft command
		command := &pb.Command{
			Event: event,
			Action: &pb.Command_Like{
				Like: action.Like,
			},
		}
		//submit to raft
		if s.raft.cm.Submit(command) {
			return command.GetLike().Group.Groupname, event
		}

	case *pb.GroupChatRequest_Unlike:
		event := "r"
		//get the request payload
		action := &pb.GroupChatRequest_Unlike{
			Unlike: req.GetUnlike(),
		}

		//construct raft command
		command := &pb.Command{
			Event: event,
			Action: &pb.Command_Unlike{
				Unlike: action.Unlike,
			},
		}
		//submit to raft
		if s.raft.cm.Submit(command) {
			return command.GetUnlike().Group.Groupname, event
		}

	case *pb.GroupChatRequest_Joinchat:
		event := "j"
		//get the request payload
		action := &pb.GroupChatRequest_Joinchat{
			Joinchat: req.GetJoinchat(),
		}

		//construct raft command
		command := &pb.Command{
			Event: event,
			Action: &pb.Command_Joinchat{
				Joinchat: action.Joinchat,
			},
		}

		user := action.Joinchat.Joineduser
		currgroupname := action.Joinchat.Currgroup
		newgroupname := action.Joinchat.Newgroup
		currclient := [2]string{currgroupname, user.Name}
		//remove the current client stream
		s.connstore.RemoveConn(currclient)

		if s.raft.cm.Submit(command) {
			return newgroupname, event
		}

	case *pb.GroupChatRequest_Print:
		event := "p"
		//get the request payload
		action := &pb.GroupChatRequest_Print{
			Print: req.GetPrint(),
		}

		return action.Print.Groupname, event
	}

	return "", ""
}

// Broadcasts the respecitve group infos to the connected clients
func (s *ChatServiceServer) BroadCast() {
	for {
		groupname := <-s.raft.broadcast
		for stream, client := range s.connstore.clients {
			if client[0] == groupname {
				if stream.Context().Err() == context.Canceled || stream.Context().Err() == context.DeadlineExceeded {
					delete(s.connstore.clients, stream)
					continue
				}
				group, _ := s.raft.cm.storage.GetGroup(groupname)
				resp := &pb.GroupChatResponse{
					Group: group,
					Event: "b",
				}
				stream.Send(resp)
				log.Printf("Broadcasted succesfully to %v of %v group", client[1], client[0])
			}
		}
	}

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
