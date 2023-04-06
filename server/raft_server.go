//raft server for the raft consensus ,,odul. Exposes the Raft to the network and
//enables RPCs between Raft Peers

package chat_server

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

type Server struct {
	mu sync.Mutex

	serverId int
	peerIds  []int

	cm       *ConsensusModule
	rpcProxy *RPCProxy

	rpcServer *rpc.Server
	listener  net.Listener

	commitChan  chan<- CommitEntry
	peerClients map[int]*rpc.Client

	ready <-chan string
	quit  chan interface{}
	wg    sync.WaitGroup
}

func NewServer(serverId int, Listener net.Listener) *Server {

	s := new(Server)
	s.serverId = serverId

	peerIds := make([]int, 0)
	for p := 1; p <= 5; p++ {
		if p != serverId {
			peerIds = append(peerIds, p)
		}
	}
	s.peerIds = peerIds

	s.commitChan = make(chan CommitEntry, 10)
	s.peerClients = make(map[int]*rpc.Client)
	s.ConnectAllPeers(serverId)
	s.listener = Listener

	s.ready = make(chan string, 1)
	s.quit = make(chan interface{})

	s.cm = NewConsensusModule(s.serverId, s.peerIds, s, s.ready, s.commitChan)

	s.rpcServer = rpc.NewServer()
	s.rpcProxy = &RPCProxy{cm: s.cm}
	s.rpcServer.RegisterName("ConsensusModule", s.rpcProxy)

	return s
}

func (s *Server) Serve() {
	log.Printf("[%v] listening at %s", s.serverId, s.listener.Addr())
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		for {
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return
				default:
					log.Fatal("accept error:", err)
				}
			}
			s.wg.Add(1)
			go func() {
				s.rpcServer.ServeConn(conn)
				s.wg.Done()
			}()

		}
		// a = <-s.ready

	}()
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
func (s *Server) shutdown() {
	// s.cm.Stop()
	close(s.quit)
	s.listener.Close()
	s.wg.Wait()
}

func (s *Server) GetListenAddr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listener.Addr()
}

func (s *Server) ConnectToPeer(peerId int, addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.peerClients[peerId] == nil {
		client, err := rpc.Dial("tcp", addr)
		if err != nil {
			return err
		}
		log.Printf("Connected server %v to %v", s.serverId, peerId)
		s.peerClients[peerId] = client
	}
	return nil
}

func (s *Server) DisconnectPeer(peerId int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.peerClients[peerId] != nil {
		err := s.peerClients[peerId].Close()
		s.peerClients[peerId] = nil
		return err
	}
	return nil
}

func (s *Server) ConnectAllPeers(serverId int) error {
	BASE_IP := "localhost"
	for i := 1; i <= 5; i++ {
		for j := 1; j <= 5; j++ {
			if i != j {
				port := ":1200" + strconv.Itoa(j)
				addr := BASE_IP + port
				s.ConnectToPeer(j, addr)
			}
		}
	}
	return nil
}

func (s *Server) Call(id int, serviceMethod string, args interface{}, reply interface{}) error {
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

func (rpp *RPCProxy) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
		dice := rand.Intn(10)
		if dice == 9 {
			log.Printf("drop RequestVote")
			return fmt.Errorf("RPC failed")
		} else if dice == 8 {
			log.Printf("delay RequestVote")
			time.Sleep(75 * time.Millisecond)
		}
	} else {
		time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
	}
	return rpp.cm.RequestVote(args, reply)
}

func (rpp *RPCProxy) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	if len(os.Getenv("RAFT_UNRELIABLE_RPC")) > 0 {
		dice := rand.Intn(10)
		if dice == 9 {
			log.Printf("drop AppendEntries")
			return fmt.Errorf("RPC failed")
		} else if dice == 8 {
			log.Printf("delay AppendEntries")
			time.Sleep(75 * time.Millisecond)
		}
	} else {
		time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
	}
	return rpp.cm.AppendEntries(args, reply)
}
