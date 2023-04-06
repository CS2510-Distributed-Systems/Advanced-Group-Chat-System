// core raft implementation
package chat_server

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

const DebugCM = 1

// Data reported by the raft to the commit channel. Each commit notifies that the consensus is achieved
// on a command and it can be applied to the clients state , machine.
type CommitEntry struct {
	//command being commited
	Command interface{}

	//Log index where the client command is being commited
	Index int

	//Term at which the client command is commited
	Term int
}

type LogEntry struct {
	Command interface{}
	Term    int
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
	server *Server

	commitChan chan<- CommitEntry

	//internal notification channel used by go routines to notify that newly commited entries may be sent
	//on commitChan
	newCommitReadyChan chan struct{}

	//Raft state persistence on all the servers
	currentTerm int
	votedFor    int
	log         []LogEntry

	//raft state volatile on all servers
	commitIndex        int
	lastApplied        int
	state              CMState
	electionResetEvent time.Time

	//volatile raft state on the leaders
	nextIndex  map[int]int
	matchIndex map[int]int
}

// new consensus module creation with the given ID, list of peerID's and server.
// Ready channel signals CM that all the peers are connected and its safe to start
func NewConsensusModule(id int, peerIds []int, server *Server, ready <-chan int, commitChan chan<- CommitEntry) *ConsensusModule {
	cm := new(ConsensusModule)
	cm.id = id
	cm.peerIds = peerIds
	cm.server = server
	cm.commitChan = commitChan
	cm.newCommitReadyChan = make(chan struct{}, 16)
	cm.state = Follower
	cm.votedFor = -1
	cm.lastApplied = -1
	cm.nextIndex = make(map[int]int)
	cm.matchIndex = make(map[int]int)

	go func() {
		//the CM will be inactive until ready is signaled ,then it starts a countdown
		<-ready
		cm.mu.Lock()
		cm.electionResetEvent = time.Now()
		cm.mu.Unlock()
		cm.runElectionTimer()

	}()

	go cm.commitChanSender()
	return cm
}

// Reprts the current state of the CM
func (cm *ConsensusModule) Report() (id int, term int, isLeader bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.id, cm.currentTerm, cm.state == Leader
}

func (cm *ConsensusModule) Submit(command interface{}) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	log.Println("Submit received by %v: %v", cm.state, command)
	if cm.state == Leader {
		cm.log = append(cm.log, LogEntry{Command: command, Term: cm.currentTerm})
		log.Println(".. log=%v", cm.log)
		return true
	}
	return false
}

type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

func (cm *ConsensusModule) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.state == Dead {
		return nil
	}
	lastLogIndex, LastLogTerm := cm.lastLogIndexAndTerm()
	log.Println("RequestVote: %v [currentTerm = %v, votedFor = %v, log index/term= (%d,%d)]", args, cm.currentTerm, cm.votedFor, lastLogIndex, LastLogTerm)

	//if a term received is greater than the current term..become a follower
	if args.Term > cm.currentTerm {
		log.Println("Current term out of date..")
		cm.becomeFollower(args.Term)
	}

	if cm.currentTerm == args.Term &&
		(cm.votedFor == -1 || cm.votedFor == args.CandidateId) &&
		(args.LastLogTerm > LastLogTerm ||
			(args.LastLogTerm == LastLogTerm && args.LastLogIndex >= lastLogIndex)) {
		reply.VoteGranted = true
		cm.votedFor = args.CandidateId
		cm.electionResetEvent = time.Now()
	} else {
		reply.VoteGranted = false
	}

	reply.Term = cm.currentTerm
	log.Println("Requestvote Reply : %v", reply)
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
	log.Println("AppendEntries: %v", args)

	//if a term received is greater than the current term..become a follower
	if args.Term > cm.currentTerm {
		log.Println("Current term out of date..")
		cm.becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == cm.currentTerm {
		if cm.state != Follower {
			cm.becomeFollower(args.Term)
		}
		cm.electionResetEvent = time.Now()

		//Does our Entry contains an Entry at PrevLogIndex whose term matches
		if args.PrevLogIndex == -1 ||
			(args.PrevLogIndex < len(cm.log) && args.PrevLogTerm == cm.log[args.PrevLogIndex].Term) {
			reply.Success = true
			//find an insertion point - where theres a term mismatch between
			//the  existing log starting at PrevLogIndex+1 and the new entries sent in the RPC
			logInsertIndex := args.PrevLogIndex + 1
			newEntriesIndex := 0

			for {
				if logInsertIndex >= len(cm.log) || newEntriesIndex >= len(args.Entries) {
					break
				}
				if cm.log[logInsertIndex].Term != args.Entries[newEntriesIndex].Term {
					break
				}
				logInsertIndex++
				newEntriesIndex++
			}
			//by the end of the loop:
			// - logInsertIndex points to the end of the log, or an index
			//where the tem mismatches wit an entry from the leader
			// -newEntriesIndex points to the End of the entries, or an index
			//where the term mismatches with the corresponding log entry
			if newEntriesIndex < len(args.Entries) {
				log.Println("... inserting %v from index %d", args.Entries[newEntriesIndex:], logInsertIndex)
				cm.log = append(cm.log[:logInsertIndex], args.Entries[newEntriesIndex:]...)
				log.Println("... log is now: %v", cm.log)
			}

			//set commit index
			if args.LeaderCommit > cm.commitIndex {
				cm.commitIndex = intMin(args.LeaderCommit, len(cm.log)-1)
				log.Println("... setting commitIndex=%d", cm.commitIndex)
				cm.newCommitReadyChan <- struct{}{}
			}
		}
	}

	reply.Term = cm.currentTerm
	log.Println("AppedEntries reply : %v", *reply)
	return nil
}

func (cm *ConsensusModule) electionTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}

func (cm *ConsensusModule) runElectionTimer() {
	timeoutDuration := cm.electionTimeout()
	cm.mu.Lock()
	termStarted := cm.currentTerm
	cm.mu.Unlock()
	log.Println("Election time started (%v), term=%d", timeoutDuration, termStarted)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C

		cm.mu.Lock()
		if cm.state != Candidate && cm.state != Follower {
			log.Println("in election timer state= %s, bailing out", cm.state)
			return
		}

		if termStarted != cm.currentTerm {
			log.Println("in election timer term changed from %d to %d, bailing out", termStarted, cm.currentTerm)
			cm.mu.Unlock()
			return
		}

		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
			cm.startElection()
			cm.mu.Unlock()
			return
		}
		cm.mu.Unlock()
	}
}

func (cm *ConsensusModule) startElection() {
	cm.state = Candidate
	cm.currentTerm += 1
	savedCurrentTerm := cm.currentTerm
	cm.electionResetEvent = time.Now()
	cm.votedFor = cm.id
	log.Println("becomes Candidate (currentTerm=%d); log= %v", savedCurrentTerm, cm.log)

	votesReceived := 1

	//send RequestVote RPC's to all the servers Concurrently
	for _, peerId := range cm.peerIds {
		go func(peerId int) {
			cm.mu.Lock()
			savedLastLogIndex, savedLastLogTerm := cm.lastLogIndexAndTerm()
			cm.mu.Unlock()

			args := RequestVoteArgs{
				Term:         savedCurrentTerm,
				CandidateId:  cm.id,
				LastLogIndex: savedLastLogIndex,
				LastLogTerm:  savedLastLogTerm,
			}
			var reply RequestVoteReply

			log.Println("Sending RequestVote to %d: %+v", peerId, args)
			if err := cm.server.Call(peerId, "ConsensusModule.RequestVote", args, &reply); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				log.Println("received RequestVoteReply %+v", reply)

				if cm.state != Candidate {
					log.Println("While waiting for reply, State = %v", cm.state)
					return
				}

				if reply.Term > savedCurrentTerm {
					log.Println("term out of date in RequestVoteReply")
					cm.becomeFollower(reply.Term)
					return
				} else if reply.Term == savedCurrentTerm {
					if reply.VoteGranted {
						votesReceived += 1
						if votesReceived*2 > len(cm.peerIds)+1 {
							log.Println("Wins election with %d votes", votesReceived)
							cm.startLeader()
							return
						}
					}
				}
			}
		}(peerId)
	}

	//run another electiontimer if this election is not successful
	go cm.runElectionTimer()
}

// becomes a floower ans changes its state
func (cm *ConsensusModule) becomeFollower(term int) {
	log.Println("Become follower with term = %v, log = %v", term, cm.log)
	cm.state = Follower
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()

	go cm.runElectionTimer()
}

func (cm *ConsensusModule) startLeader() {
	cm.state = Leader
	log.Println("Becomes Leader; term = %d, log=%v", cm.currentTerm, cm.log)

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		//send periodic hearbeats, as long as still leader
		for {
			cm.leaderSendHeartBeats()
			<-ticker.C

			cm.mu.Lock()
			if cm.state != Leader {
				cm.mu.Unlock()
				return
			}
			cm.mu.Unlock()
		}
	}()
}

// sends a roundof heartbeats to all the peers, collects their replies and adjusts cm's state.
func (cm *ConsensusModule) leaderSendHeartBeats() {
	cm.mu.Lock()
	if cm.state != Leader {
		cm.mu.Unlock()
		return
	}
	savedCurrentTerm := cm.currentTerm
	cm.mu.Unlock()

	for _, peerId := range cm.peerIds {
		go func(peerId int) {
			cm.mu.Lock()
			ni := cm.nextIndex[peerId]
			prevLogIndex := ni - 1
			prevLogTerm := -1
			if prevLogIndex >= 0 {
				prevLogTerm = cm.log[prevLogIndex].Term
			}
			entries := cm.log[ni:]

			args := AppendEntriesArgs{
				Term:         savedCurrentTerm,
				LeaderId:     cm.id,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries:      entries,
				LeaderCommit: cm.commitIndex,
			}
			cm.mu.Unlock()

			log.Println("sending AppendEntries to %v: ni=%d, args=%+v", peerId, ni, args)
			var reply AppendEntriesReply
			if err := cm.server.Call(peerId, "ConsensusModule.AppendEntries", args, &reply); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				if reply.Term > savedCurrentTerm {
					log.Println("term out of date in heartbeat reply")
					cm.becomeFollower(reply.Term)
					return
				}

				if cm.state == Leader && savedCurrentTerm == reply.Term {
					if reply.Success {
						cm.nextIndex[peerId] = ni + len(entries)
						cm.matchIndex[peerId] = cm.nextIndex[peerId] - 1
						log.Println("AppendEntries reply from %d success: nextIndex := %v, matchIndex := %v", peerId, cm.nextIndex)

						savedCommitIndex := cm.commitIndex
						for i := cm.commitIndex + 1; i < len(cm.log); i++ {
							if cm.log[i].Term == cm.currentTerm {
								matchCount := 1
								for _, peerId := range cm.peerIds {
									if cm.matchIndex[peerId] >= i {
										matchCount++
									}
								}
								if matchCount*2 > len(cm.peerIds)+1 {
									cm.commitIndex = i
								}
							}
						}
						if cm.commitIndex != savedCommitIndex {
							log.Println("leader sets commmitIndex := %d", cm.commitIndex)
							cm.newCommitReadyChan <- struct{}{}
						}
					} else {
						cm.nextIndex[peerId] = ni - 1
						log.Println("AppendEntries reply from %d !sucess: nextIndex := %d", peerId, ni-1)
					}
				}
			}
		}(peerId)
	}
}

func (cm *ConsensusModule) commitChanSender() {
	for range cm.newCommitReadyChan {
		//find which entries we need to apply
		cm.mu.Lock()
		savedTerm := cm.currentTerm
		savedLastApplied := cm.lastApplied
		var entries []LogEntry
		if cm.commitIndex > cm.lastApplied {
			entries = cm.log[cm.lastApplied+1 : cm.commitIndex+1]
			cm.lastApplied = cm.commitIndex
		}
		cm.mu.Unlock()
		log.Println("commitChanSender entries = %v, savedLastApplied = %d", entries, savedLastApplied)

		for i, entry := range entries {
			cm.commitChan <- CommitEntry{
				Command: entry.Command,
				Index:   savedLastApplied + i + 1,
				Term:    savedTerm,
			}
		}
	}
	log.Println("CommitChanSender done")
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// lastLogIndexAndTerm returns the last log index and the last log entry's term
// (or -1 if there's no log) for this server.
// Expects cm.mu to be locked.
func (cm *ConsensusModule) lastLogIndexAndTerm() (int, int) {
	if len(cm.log) > 0 {
		lastIndex := len(cm.log) - 1
		return lastIndex, cm.log[lastIndex].Term
	} else {
		return -1, -1
	}
}
