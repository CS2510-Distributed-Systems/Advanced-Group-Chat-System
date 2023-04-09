//raft server for the raft consensus ,,odul. Exposes the Raft to the network and
//enables RPCs between Raft Peers

package chat_server

import (
	"chat-system/pb"
	"context"
	"log"
	"net"
	"strconv"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	pb.UnimplementedRaftServiceServer
	mu sync.Mutex

	serverId int64
	peerIds  []int64

	cm       *ConsensusModule
	listener net.Listener

	commitChan  chan<- CommitEntry
	peerClients map[int64]*grpc.ClientConn

	ready chan int
	quit  chan interface{}
}

func NewServer(serverId int64, Listener net.Listener) *Server {

	s := new(Server)
	s.serverId = serverId

	peerIds := make([]int64, 0)
	for p := 1; p <= 5; p++ {
		if p != int(serverId) {
			peerIds = append(peerIds, int64(p))
		}
	}
	s.peerIds = peerIds

	s.commitChan = make(chan CommitEntry, 10)
	s.peerClients = make(map[int64]*grpc.ClientConn)
	go s.ConnectAllPeers(serverId)
	s.listener = Listener

	s.ready = make(chan int, 10)
	s.quit = make(chan interface{})

	s.cm = NewConsensusModule(s.serverId, s.peerIds, s, s.ready, s.commitChan)

	return s
}

func (s *Server) Serve() {
	log.Printf("[%v] listening at %s", s.serverId, s.listener.Addr())
	// s.wg.Add(1)
	close(s.ready)
}

// closes all the client connections to peers for this server
func (s *Server) DisconnectAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.peerClients {
		if s.peerClients[id] != nil {
			s.peerClients[id].Close()
			s.peerClients[id] = nil
		}
	}
}

// shutdown the server and waits for it to shutdown gracefully
// func (s *Server) shutdown() {
// 	// s.cm.Stop()
// 	close(s.quit)
// 	s.listener.Close()
// 	s.wg.Wait()
// }

func (s *Server) GetListenAddr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listener.Addr()
}

func (s *Server) GetPeerAddr(peerId int64) string {
	BASE_IP := "0.0.0.0"
	port := ":1200" + strconv.Itoa(int(peerId))
	addr := BASE_IP + port
	return addr
}

func (s *Server) ConnectToPeer(peerId int64, addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	transportOption := grpc.WithTransportCredentials(insecure.NewCredentials())
	clientconn, err := grpc.Dial(addr, transportOption)
	if err != nil {
		log.Printf("Cannot Dail server %v", peerId)

	}
	log.Print(clientconn.GetState())
	log.Printf("Connected server %v to %v", s.serverId, peerId)
	s.peerClients[peerId] = clientconn
	return nil
}

func (s *Server) ReconnectToPeer(peerId int64) error {
	log.Printf("Reconnecting to server %v", peerId)
	addr := s.GetPeerAddr(peerId)
	s.ConnectToPeer(peerId, addr)
	return nil
}

func (s *Server) DisconnectPeer(peerId int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.peerClients[peerId] != nil {
		err := s.peerClients[peerId].Close()
		s.peerClients[peerId] = nil
		return err
	}
	return nil
}

func (s *Server) ConnectAllPeers(serverId int64) {
	for j := 1; j <= 5; j++ {
		if int(serverId) != j {
			k := int64(j)
			addr := s.GetPeerAddr(k)
			s.ConnectToPeer(k, addr)
		}
	}
}

type RPCProxy struct {
	cm *ConsensusModule
}

func (s *Server) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	return s.cm.RequestVoteHelper(req)
}

func (s *Server) ForwardLeader(ctx context.Context, req *pb.ForwardLeaderRequest) (*pb.ForwardLeaderResponse, error) {
	return s.cm.ForwardLeaderHelper(req)
}

func (s *Server) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	return s.cm.AppendEntriesHelper(req)
}
