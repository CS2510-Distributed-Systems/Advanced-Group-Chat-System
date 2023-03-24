package service

import (
	"chat-system/pb"
	"context"
	"io"
	"log"

	"github.com/google/uuid"
)

// chat service
type ChatServiceServer struct {
	pb.UnimplementedChatServiceServer
	pb.UnimplementedAuthServiceServer
	groupstore GroupStore
	UserStore  UserStore
	clients    ConnStore
}

func NewChatServiceServer(groupstore GroupStore, userstore UserStore, clients ConnStore) *ChatServiceServer {
	return &ChatServiceServer{
		groupstore: groupstore,
		UserStore:  userstore,
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
	s.UserStore.SaveUser(newUser)
	res := &pb.LoginResponse{
		User: req.GetUser(),
	}

	return res, nil
}

// logout rpc
func (s *ChatServiceServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	s.UserStore.DeleteUser(req.User.User.Id)
	resp := &pb.LogoutResponse{
		Status: true,
	}
	log.Println("User deleted")
	return resp, nil
}

// Experiment to merge JoinGroup and Groupchat rpc's
func (s *ChatServiceServer) JoinGroupChat(stream pb.ChatService_JoinGroupChatServer) error {
	log.Println("Received Join Group Request")

	ctx := stream.Context()

	go func() error {
		for {

			//exit if context is done
			select {
			case <-ctx.Done():
				log.Println("Context Error")
				return ctx.Err()
			default:
			}

			//receive data
			req, err := stream.Recv()
			if err == io.EOF {
				log.Println("Stream Ended in the server side")
				return nil
			}
			if err != nil {
				log.Println("Error in Receive.")
				continue
			}

			//processing received request
			groupname, command := s.ProcessRequest(req)

			//adding the new client
			if command == "j" {
				user := req.GetJoinchat().User
				//add the requested stream
				newclient := [2]string{groupname, user.Name}
				s.clients.AddConn(stream, newclient)
			}

			//prepare a response
			resp := &pb.GroupChatResponse{
				Group:   s.groupstore.GetGroup(groupname),
				Command: command,
			}

			//braodcast to clients who are in same group
			s.clients.BroadCast(groupname, resp)
			if err := stream.Send(resp); err != nil {
				log.Printf("send error %v", err)
			}

		}
	}()

	return nil

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
		// currclient := [2]string{currgroupname, user.Name}
		//remove the current client stream
		// s.clients.RemoveConn(currclient)
		//remove the current client group from store
		s.groupstore.RemoveUser(user.Id, currgroupname)
		return newgroupname, command

	default:
		log.Printf("let the client enter the command")
	}

	return "", ""
}

// joining a group rpc
// func (s *ChatServiceServer) JoinGroup(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {

// 	md, ok := metadata.FromIncomingContext(ctx)
// 	if !ok {
// 		log.Printf("didnt receive the context from the client")
// 	}

// 	groupname := md.Get("groupname")[0]
// 	userid := md.Get("userid")[0]
// 	client := [2]string{groupname, userid}
// 	//removing the client from current group
// 	s.clients.RemoveConn(client)

// 	currentchat := req.GetJoinchat()
// 	log.Printf("receive a group join request with name: %s", currentchat.Newgroup)
// 	// remove if user is already in a group
// 	s.groupstore.RemoveUser(currentchat.User, groupname)
// 	s.clients.BroadCast(groupname, &pb.GroupChatResponse{Group: s.groupstore.GetGroup(groupname)})
// 	// join a group
// 	group_details, err := s.groupstore.JoinGroup(currentchat.Newgroup, currentchat.User)
// 	if err != nil {
// 		log.Printf("Failed to join group %v", err)
// 	}

// 	log.Printf("Joined group %s", currentchat.Newgroup)
// 	res := &pb.JoinResponse{
// 		Group: group_details,
// 	}

// 	return res, nil

// }

// bi-directional streaming rpc
// func (s *ChatServiceServer) GroupChat(stream pb.ChatService_GroupChatServer) error {
// 	//fetching the current group of the client from the rpc context.
// 	_, cancel := context.WithCancel(stream.Context())
// 	defer cancel()
// 	md, ok := metadata.FromIncomingContext(stream.Context())
// 	if !ok {
// 		log.Printf("didnt receive the context properly from the client..")
// 	}

// 	groupname := md.Get("groupname")[0]
// 	userid := md.Get("userid")[0]
// 	client := [2]string{groupname, userid}
// 	//add the conn to server
// 	s.clients.AddConn(stream, client)

// 	errch := make(chan error)
// 	listener := make(chan string)
// 	resp := &pb.GroupChatResponse{
// 		Group:   s.groupstore.GetGroup(groupname),
// 		Command: "j",
// 	}
// 	//broadcast the change
// 	s.clients.BroadCast(groupname, resp)

// 	wg.Add(2)
// 	//receive go routine
// 	go receivestream(stream, s, groupname, listener)
// 	//send go routine
// 	go func() error {
// 		defer wg.Done()
// 		defer log.Println("send server stream ended")
// 		for {
// 			command := <-listener
// 			if command == "q" {
// 				resp := &pb.GroupChatResponse{
// 					Group:   s.groupstore.GetGroup(groupname),
// 					Command: command,
// 				}
// 				s.clients.BroadCast(groupname, resp)
// 				err := errors.New("Graceful shutdown requested")
// 				stream.SendMsg(err)
// 				return err
// 			}

// 			resp := &pb.GroupChatResponse{
// 				Group:   s.groupstore.GetGroup(groupname),
// 				Command: command,
// 			}
// 			s.clients.BroadCast(groupname, resp)

// 		}
// 	}()

// 	wg.Wait()
// 	log.Printf("Stream ended for %v. Please join other group", groupname)
// 	// go refreshgroup(stream, s, resp)

// 	return <-errch
// }

// var mutex = &sync.RWMutex{}

// func receivestream(stream pb.ChatService_GroupChatServer, s *ChatServiceServer, groupname string, listener chan string) error {
// 	defer wg.Done()
// 	defer log.Println("Receive server  stream ended")
// 	for {
// 		err := contextError(stream.Context())
// 		if err != nil {
// 			return err
// 		}
// 		req, err := stream.Recv()
// 		if err == io.EOF {
// 			log.Print("no more data")
// 			break
// 		}
// 		if err != nil {
// 			log.Printf("error in receiving from client")
// 			return logError(status.Errorf(codes.Unknown, "cannot receive stream request: %v", err))
// 		}
// 		switch req.GetAction().(type) {
// 		case *pb.GroupChatRequest_Append:
// 			command := "a"
// 			appendchat := &pb.AppendChat{
// 				Group:       req.GetAppend().GetGroup(),
// 				Chatmessage: req.GetAppend().GetChatmessage(),
// 			}
// 			err := s.groupstore.AppendMessage(appendchat)
// 			if err != nil {
// 				log.Printf("cannot save the message %s", err)
// 			}
// 			//sending to the channel
// 			listener <- command

// 		case *pb.GroupChatRequest_Like:
// 			command := "l"
// 			group := req.GetLike().Group
// 			msgId := req.GetLike().Messageid
// 			user := req.GetLike().User
// 			likemessage := &pb.LikeMessage{
// 				Group:     group,
// 				Messageid: msgId,
// 				User:      user,
// 			}
// 			err := s.groupstore.LikeMessage(likemessage)
// 			if err != nil {
// 				log.Printf("%s", err)
// 			}
// 			//sending to the channel
// 			listener <- command

// 		case *pb.GroupChatRequest_Unlike:
// 			command := "r"
// 			group := req.GetUnlike().Group
// 			msgId := req.GetUnlike().Messageid
// 			user := req.GetUnlike().User
// 			unlikemessage := &pb.UnLikeMessage{
// 				Group:     group,
// 				Messageid: msgId,
// 				User:      user,
// 			}
// 			s.groupstore.UnLikeMessage(unlikemessage)
// 			if err != nil {
// 				log.Printf("some error occured in unliking the message: %s", err)
// 			}
// 			listener <- command

// 		case *pb.GroupChatRequest_Print:
// 			command := "p"
// 			resp := &pb.GroupChatResponse{
// 				Group:   s.groupstore.GetGroup(groupname),
// 				Command: command,
// 			}
// 			stream.Send(resp)

// 		case *pb.GroupChatRequest_Logout:
// 			command := "q"
// 			user := req.GetLogout().User
// 			s.UserStore.DeleteUser(user)
// 			s.groupstore.RemoveUser(user, groupname)
// 			client := [2]string{groupname, strconv.Itoa(int(user.Id))}
// 			listener <- command
// 			s.clients.RemoveConn(client)
// 			log.Printf("user : %v left the chat", user.Name)
// 			return nil

// 		default:
// 			log.Printf("let the client enter the command")
// 		}

// 	}
// 	return nil
// }

// func contextError(ctx context.Context) error {
// 	switch ctx.Err() {
// 	case context.Canceled:
// 		return logError(status.Error(codes.Canceled, "request is canceled"))
// 	case context.DeadlineExceeded:
// 		return logError(status.Error(codes.DeadlineExceeded, "deadline is exceeded"))
// 	default:
// 		return nil
// 	}
// }

// func logError(err error) error {
// 	if err != nil {
// 		log.Print(err)
// 	}
// 	return err
// }
