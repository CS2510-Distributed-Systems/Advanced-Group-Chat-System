//raft server for the raft consensus ,,odul. Exposes the Raft to the network and
//enables RPCs between Raft Peers

package chat_server

import (
	"chat-system/pb"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"strconv"
	"sync"
)


type Server struct {
	pb.UnimplementedRaftServiceServer
	mu sync.Mutex

	serverId int64
	peerIds  []int64

	cm       *ConsensusModule
	rpcProxy *RPCProxy

	rpcServer *rpc.Server
	listener  net.Listener

	commitChan  chan<- CommitEntry
	peerClients map[int64]*rpc.Client

	ready chan int
	quit  chan interface{}
	wg    sync.WaitGroup
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
	s.peerClients = make(map[int64]*rpc.Client)
	go s.ConnectAllPeers(serverId)
	s.listener = Listener

	s.ready = make(chan int, 10)
	s.quit = make(chan interface{})

	s.cm = NewConsensusModule(s.serverId, s.peerIds, s, s.ready, s.commitChan)

	s.rpcServer = rpc.NewServer()
	s.rpcProxy = &RPCProxy{cm: s.cm}
	s.rpcServer.RegisterName("ConsensusModule", s.rpcProxy)

	//Initialize the Diskstore persistance storage
	//config for app data

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
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("error occured : %v", err)
	}
	log.Printf("Connected server %v to %v", s.serverId, peerId)
	s.peerClients[peerId] = client
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
	for {
		for j := 1; j <= 5; j++ {
			if int(serverId) != j {
				k := int64(j)
				if s.peerClients[k] == nil {
					addr := s.GetPeerAddr(k)
					s.ConnectToPeer(k, addr)
				}

			}
		}
	}

}

func (s *Server) Call(id int64, serviceMethod string, args interface{}, reply interface{}) error {
	s.mu.Lock()
	peer := s.peerClients[id]
	s.mu.Unlock()

	if peer == nil {
		return fmt.Errorf("call client %d after its closes", id)
	} else {
		return peer.Call(serviceMethod, args, reply)
	}
}

type RPCProxy struct {
	cm *ConsensusModule
}

// func (rpp *RPCProxy) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
// 	if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
// 		dice := rand.Intn(10)
// 		if dice == 9 {
// 			log.Println("drop RequestVote")
// 			return fmt.Errorf("RPC failed")
// 		} else if dice == 8 {
// 			log.Println("delay RequestVote")
// 			time.Sleep(75 * time.Millisecond)
// 		}
// 	} else {
// 		time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
// 	}
// 	return rpp.cm.RequestVote(args, reply)
// }

// func (rpp *RPCProxy) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
// 	if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
// 		dice := rand.Intn(10)
// 		if dice == 9 {
// 			log.Println("drop AppendEntries")
// 			return fmt.Errorf("RPC failed")
// 		} else if dice == 8 {
// 			log.Println("delay AppendEntries")
// 			time.Sleep(75 * time.Millisecond)
// 		}
// 	} else {
// 		time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
// 	}
// 	return rpp.cm.AppendEntries(args, reply)
// }

