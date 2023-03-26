// core raft implementation
package raft

import (
	"log"
	"sync"
	"time"
)

const DebugCM = 1

type LogEntry struct {
	Command interface{}
	Term    uint32
}

type CMState int

const (
	Follower CMState = iota
	Candidate
	Leader
	Dead
)

func (s CMState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Dead:
		return "Dead"
	default:
		panic("unreachable")
	}
}

// consensus module (CM) that runs on every server in the cluster
type ConsensusModule struct {
	//lock for the concurrent access to CM
	mu sync.Mutex

	//id of the server of this CM
	id int

	// peerID's list of our peers in the cluster
	peerIds []int

	//server of this CM
	server *server

	//Raft state persistence on all the servers
	currentTerm int
	votedFor    int
	log         []LogEntry

	//raft state volatile on all servers
	state              CMState
	electionResetEvent time.Time
}

// new consensus module creation with the given ID, list of peerID's and server.
// Ready channel signals CM that all the peers are connected and its safe to start
func NewConsensusModule(id int, peerIds []int, server *server, ready <-chan interface{}) *ConsensusModule {
	cm := new(ConsensusModule)
	cm.id = id
	cm.peerIds = peerIds
	cm.server = server
	cm.state = Follower
	cm.votedFor = -1

	go func() {
		//the CM will be inactive until ready is signaled ,then it starts a countdown
		<-ready
		cm.mu.Lock()
		cm.electionResetEvent = time.Now()
		cm.mu.Unlock()
		cm.runElectionTimer()

	}()

	return cm
}

// Reprts the current state of the CM
func (cm *ConsensusModule) Report() (id int, term int, isLeader bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.id, cm.currentTerm, cm.state == Leader
}

type RequestVoreArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

func (cm *ConsensusModule) Requestvote(args RequestVoreArgs, reply *RequestVoteReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.state == Dead {
		return nil
	}
	log.Printf("RequestVote: %v [currentTerm = %v, votedFor = %v]", args, cm.currentTerm, cm.votedFor)

	//if a term received is greater than the current term..become a follower
	if args.Term > cm.currentTerm {
		log.Printf("Current term out of date..")
		cm.becomeFollower(args.Term)
	}

	if cm.currentTerm == args.Term && (cm.votedFor == -1 || cm.votedFor == args.CandidateId) {
		reply.VoteGranted = true
		cm.votedFor = args.CandidateId
		cm.electionResetEvent = time.Now()
	} else {
		reply.VoteGranted = false
	}

	reply.Term = cm.currentTerm
	log.Printf("Requestvote Reply : %v", reply)
	return nil
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (cm *ConsensusModule) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.state == Dead {
		return nil
	}
	log.Printf("AppendEntries: %v", args)

	//if a term received is greater than the current term..become a follower
	if args.Term > cm.currentTerm {
		log.Printf("Current term out of date..")
		cm.becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == cm.currentTerm {
		if cm.state != Follower {
			cm.becomeFollower(args.Term)
		}
		cm.electionResetEvent = time.Now()
		reply.Success = true
	}

	reply.Term = cm.currentTerm
	log.Printf("AppedEntries reply : %v", *reply)
	return nil
}

// becomes a floower ans changes its state
func (cm *ConsensusModule) becomeFollower(term int) {
	log.Printf("Become follower with term = %v, log = %v", term, cm.log)
	cm.state = Follower
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()

	go cm.runElectiontimer()
}
