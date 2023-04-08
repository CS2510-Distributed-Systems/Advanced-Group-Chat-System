// core raft implementation
package chat_server

import (
	"chat-system/pb"
	"log"
	"math/rand"
	"sync"
	"time"
)

const DebugCM = 1

// Command struct for the chat service application
type Command struct {
	Event          string
	Triggeredby    string
	Triggeredgroup string
	Message        *pb.ChatMessage
}

func NewCommand() *Command {
	return &Command{
		Event:          "",
		Triggeredby:    "",
		Triggeredgroup: "",
		Message: &pb.ChatMessage{
			MessagedBy: &pb.User{
				Id:   0,
				Name: "",
			},
			Message: "",
			LikedBy: make(map[string]string),
		},
	}
}

// Data reported by the raft to the commit channel. Each commit notifies that the consensus is achieved
// on a command and it can be applied to the clients state machine.
type CommitEntry struct {
	//command being commited
	Command Command

	//Log index where the client command is being commited
	Index int

	//Term at which the client command is commited
	Term int
}

type LogEntry struct {
	Command Command
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

	//keep track of leader
	leaderid int

	// peerID's list of our peers in the cluster
	peerIds []int

	//server of this CM
	server *Server

	//persistance storage of app data and raft data
	storage *DiskStore

	commitChan chan<- CommitEntry

	//internal notification channel used by go routines to notify that newly commited entries may be sent
	//on commitChan
	newCommitReadyChan chan struct{}

	// triggerAEChan is an internal notification channel used to trigger
	// sending new AEs to followers when interesting changes occurred.
	triggerAEChan chan struct{}

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
	log.Printf("Initialising raft consensus..")
	cm := new(ConsensusModule)
	cm.id = id
	cm.leaderid = -1
	cm.peerIds = peerIds
	cm.server = server
	cm.commitChan = commitChan
	cm.newCommitReadyChan = make(chan struct{}, 16)
	cm.triggerAEChan = make(chan struct{}, 1)
	cm.state = Follower
	cm.votedFor = -1
	cm.lastApplied = -1
	cm.nextIndex = make(map[int]int)
	cm.matchIndex = make(map[int]int)
	cm.commitIndex = -1
	cm.storage = NewDiskStore(cm.id)

	//restore the state from the persisted storage
	if cm.storage.HasData() {
		log.Printf("Found Persistant data, Retreiving..")
		cm.restoreFromStorage()
	}
	log.Printf("No Persistant data found")
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

func (cm *ConsensusModule) Submit(command Command) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	log.Printf("Submit received by %v: %v", cm.state, command)
	if cm.state == Leader {
		cm.log = append(cm.log, LogEntry{Command: command, Term: cm.currentTerm})
		cm.persistToStorage()
		log.Printf("... log=%v", cm.log)
		cm.mu.Unlock()
		cm.triggerAEChan <- struct{}{}
		return true
	} else {
		cm.forwardToLeader(command)

	}
	return false
}

// Stop stops this CM, cleaning up its state. This method returns quickly, but
// it may take a bit of time (up to ~election timeout) for all goroutines to
// exit.
func (cm *ConsensusModule) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.state = Dead
	log.Printf("becomes Dead")
	close(cm.newCommitReadyChan)
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
	log.Printf("RequestVote: %v [currentTerm = %v, votedFor = %v, log index/term= (%d,%d)]", args, cm.currentTerm, cm.votedFor, lastLogIndex, LastLogTerm)

	//if a term received is greater than the current term..become a follower
	if args.Term > cm.currentTerm {
		log.Printf("Current term out of date..")
		cm.becomeFollower(args.Term)
	}

	//if term matches
	//if cm didnot vote for any candidate yet..vote
	//if already voted..ignore
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
	cm.persistToStorage()
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

	// Faster conflict resolution optimization (described near the end of section
	// 5.3 in the paper.)
	ConflictIndex int
	ConflictTerm  int
}

type ForwardToLeaderArgs struct {
	Command Command
}

type LeaderReply struct {
	Success bool
	//Receive a leaderid if its sent to follower
	Leaderid int
}

func (cm *ConsensusModule) ForwardRequest(args ForwardToLeaderArgs, reply *LeaderReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.state == Dead {
		return nil
	}
	log.Printf("Forwarding the command to leader which is server: %v", cm.leaderid)

	reply.Success = false
	//if cm is leader
	if cm.state == Leader {
		if cm.Submit(args.Command) {
			reply.Success = true
			reply.Leaderid = cm.id
		}
	}

	return nil
}

func (cm *ConsensusModule) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.state == Dead {
		return nil
	}
	log.Printf("AppendEntries: %v", args)
	//update the current leader details
	cm.leaderid = args.LeaderId
	//if a term received is greater than the current term..become a follower
	if args.Term > cm.currentTerm {
		log.Printf("Current term out of date..")
		cm.becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == cm.currentTerm {
		//it may happen that two leaders exists in same term..make later as follower
		if cm.state != Follower {
			cm.becomeFollower(args.Term)
		}
		cm.electionResetEvent = time.Now()

		//Does our log contains an Entry at PrevLogIndex whose term matches
		if args.PrevLogIndex == -1 ||
			(args.PrevLogIndex < len(cm.log) && args.PrevLogTerm == cm.log[args.PrevLogIndex].Term) {
			reply.Success = true
			//find an insertion point - where theres a term mismatch between
			//the existing log starting at PrevLogIndex+1 and the new entries sent in the RPC
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
				log.Printf("... inserting %v from index %d", args.Entries[newEntriesIndex:], logInsertIndex)
				cm.log = append(cm.log[:logInsertIndex], args.Entries[newEntriesIndex:]...)
				log.Printf("... log is now: %v", cm.log)
			}

			//set commit index
			if args.LeaderCommit > cm.commitIndex {
				cm.commitIndex = intMin(args.LeaderCommit, len(cm.log)-1)
				log.Printf("... setting commitIndex=%d", cm.commitIndex)
				cm.newCommitReadyChan <- struct{}{}
			}
		}
	}

	reply.Term = cm.currentTerm
	cm.persistToStorage()
	log.Printf("AppedEntries reply : %v", *reply)
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
	// log.Printf("Election time started (%v), term=%d", timeoutDuration, termStarted)

	//This loops until
	//1. When election timer is no longer needed
	//2. Election timer expires and CM becomes a condidate
	//for a follower, this keeps ticking for lifetime
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C

		cm.mu.Lock()
		//if its a leader . stop the timer
		if cm.state != Candidate && cm.state != Follower {
			log.Printf("in election timer state= %s, bailing out", cm.state)
			return
		}

		//if terms dont match. stop the timer
		if termStarted != cm.currentTerm {
			log.Printf("in election timer term changed from %d to %d, bailing out", termStarted, cm.currentTerm)
			cm.mu.Unlock()
			return
		}

		//if cm is follower and didnt hear from leader for the duration of timeout
		//start the Election
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
	// log.Printf("becomes Candidate (currentTerm=%d); log= %v", savedCurrentTerm, cm.log)

	votesReceived := 1

	//send RequestVote RPC's to all the servers Concurrently, to collect the votes
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

			// log.Printf("Sending RequestVote to %d: %+v", peerId, args)
			if err := cm.server.Call(peerId, "ConsensusModule.RequestVote", args, &reply); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				log.Printf("received RequestVoteReply %+v", reply)
				//As we got a response check if the peer is connected to the cm.
				if cm.server.peerClients[peerId] == nil {
					log.Printf("Peer %v connected back.", peerId)
					cm.server.ReconnectToPeer(peerId)
				}
				//when other RV calls received a majority, then election might have already been done
				if cm.state != Candidate {
					log.Printf("While waiting for reply, State changed to %v", cm.state)
					return
				}

				//if cm finds that its term is stale, become follower. else proceed with voting
				if reply.Term > savedCurrentTerm {
					log.Printf("term out of date in RequestVoteReply")
					cm.becomeFollower(reply.Term)
					return
				} else if reply.Term == savedCurrentTerm {
					if reply.VoteGranted {
						votesReceived += 1
						if votesReceived*2 > len(cm.peerIds)+1 {
							log.Printf("Wins election with %d votes", votesReceived)
							cm.startLeader()
							return
						}
					}
				}
			} else if cm.server.peerClients[peerId] != nil {
				log.Printf("Seems like Peer %v disconnected. Removing from peer client list.", peerId)
				cm.server.peerClients[peerId] = nil
			}
		}(peerId)
	}

	//run another electiontimer if this election is not successful
	go cm.runElectionTimer()
}

// becomes a floower ans changes its state
func (cm *ConsensusModule) becomeFollower(term int) {
	log.Printf("Become follower with term = %v, log = %v", term, cm.log)
	cm.state = Follower
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()

	go cm.runElectionTimer()
}

func (cm *ConsensusModule) startLeader() {
	cm.state = Leader
	cm.leaderid = cm.id
	log.Printf("Becomes Leader; term = %d, log=%v", cm.currentTerm, cm.log)

	for _, peerId := range cm.peerIds {
		cm.nextIndex[peerId] = len(cm.log)
		cm.matchIndex[peerId] = -1
	}

	// This goroutine runs in the background and sends AEs to peers:
	// * Whenever something is sent on triggerAEChan
	// * ... Or every 50 ms, if no events occur on triggerAEChan
	go func(heartbeatTimeout time.Duration) {
		// Immediately send AEs to peers.
		cm.leaderSendAEs()

		t := time.NewTimer(heartbeatTimeout)
		defer t.Stop()
		for {
			doSend := false
			select {
			case <-t.C:
				doSend = true

				// Reset timer to fire again after heartbeatTimeout.
				t.Stop()
				t.Reset(heartbeatTimeout)
			case _, ok := <-cm.triggerAEChan:
				if ok {
					doSend = true
				} else {
					return
				}

				// Reset timer for heartbeatTimeout.
				if !t.Stop() {
					<-t.C
				}
				t.Reset(heartbeatTimeout)
			}

			if doSend {
				// If this isn't a leader any more, stop the heartbeat loop.
				cm.mu.Lock()
				if cm.state != Leader {
					cm.mu.Unlock()
					return
				}
				cm.mu.Unlock()
				cm.leaderSendAEs()
			}
		}
	}(50 * time.Millisecond)
}

func (cm *ConsensusModule) forwardToLeader(command Command) {
	if cm.state != Leader {
		go func() {
			log.Printf("Sorry, I'm not the leader rn.")

			args := ForwardToLeaderArgs{
				Command: command,
			}
			var reply LeaderReply
			cm.mu.Lock()
			leader := cm.leaderid

			// log.Printf("Sending RequestVote to %d: %+v", peerId, args)
			log.Printf("forwarding to Leader server %v", cm.leaderid)
			if err := cm.server.Call(leader, "ConsensusModule.ForwardRequest", args, &reply); err == nil {

				defer cm.mu.Unlock()
				log.Printf("received LeaderReply %v", reply)
				//As we got a response check if the peer is connected to the cm.
				if cm.server.peerClients[leader] == nil {
					cm.server.ReconnectToPeer(leader)
				}

				if reply.Success {
					log.Printf("Succesfully forwarded the request to leader")
					cm.leaderid = reply.Leaderid
				} else {
					cm.forwardToLeader(command)
				}
			} else {
				log.Printf("Seems like Peer disconnected. Removing from peer client list.")
				cm.server.peerClients[leader] = nil
			}
		}()

	}
}

// leaderSendAEs sends a round of AEs to all peers, collects their
// replies and adjusts cm's state.
func (cm *ConsensusModule) leaderSendAEs() {
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
			log.Printf("sending AppendEntries to %v: ni=%d, args=%+v", peerId, ni, args)
			var reply AppendEntriesReply
			if err := cm.server.Call(peerId, "ConsensusModule.AppendEntries", args, &reply); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				//As we got a response check if the peer is connected to the cm.
				if cm.server.peerClients[peerId] == nil {
					log.Printf("Peer %v connected back.", peerId)
					cm.server.ReconnectToPeer(peerId)
				}
				if reply.Term > cm.currentTerm {
					log.Printf("term out of date in heartbeat reply")
					cm.becomeFollower(reply.Term)
					return
				}

				if cm.state == Leader && savedCurrentTerm == reply.Term {
					if reply.Success {
						cm.nextIndex[peerId] = ni + len(entries)
						cm.matchIndex[peerId] = cm.nextIndex[peerId] - 1

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
						log.Printf("AppendEntries reply from %d success: nextIndex := %v, matchIndex := %v; commitIndex := %d", peerId, cm.nextIndex, cm.matchIndex, cm.commitIndex)
						if cm.commitIndex != savedCommitIndex {
							log.Printf("leader sets commitIndex := %d", cm.commitIndex)
							// Commit index changed: the leader considers new entries to be
							// committed. Send new entries on the commit channel to this
							// leader's clients, and notify followers by sending them AEs.
							cm.newCommitReadyChan <- struct{}{}
							cm.triggerAEChan <- struct{}{}
						}
					} else {
						if reply.ConflictTerm >= 0 {
							lastIndexOfTerm := -1
							for i := len(cm.log) - 1; i >= 0; i-- {
								if cm.log[i].Term == reply.ConflictTerm {
									lastIndexOfTerm = i
									break
								}
							}
							if lastIndexOfTerm >= 0 {
								cm.nextIndex[peerId] = lastIndexOfTerm + 1
							} else {
								cm.nextIndex[peerId] = reply.ConflictIndex
							}
						} else {
							cm.nextIndex[peerId] = reply.ConflictIndex
						}
						log.Printf("AppendEntries reply from %d !success: nextIndex := %d", peerId, ni-1)
					}
				}
			} else if cm.server.peerClients[peerId] != nil {
				log.Printf("Seems like Peer %v disconnected. Removing from peer client list.", peerId)
				cm.server.peerClients[peerId] = nil
			}
		}(peerId)
	}
}

func (cm *ConsensusModule) commitChanSender() {
	for range cm.newCommitReadyChan {
		//find which entries we need to apply
		cm.mu.Lock()
		savedTerm := cm.currentTerm
		//get last applied index
		savedLastApplied := cm.lastApplied
		var entries []LogEntry
		//if commit index is greater.. get the remaining log entries that need to be applied
		if cm.commitIndex > cm.lastApplied {
			entries = cm.log[cm.lastApplied+1 : cm.commitIndex+1]
			cm.lastApplied = cm.commitIndex
		}
		cm.mu.Unlock()
		log.Printf("commitChanSender entries = %v, savedLastApplied = %d", entries, savedLastApplied)

		//send them to commit channel
		for i, entry := range entries {
			cm.commitChan <- CommitEntry{
				Command: entry.Command,
				Index:   savedLastApplied + i + 1,
				Term:    savedTerm,
			}
		}
	}
	log.Printf("CommitChanSender done")
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

// restores the persistant state of this CM from storage
func (cm *ConsensusModule) restoreFromStorage() {

	term, votedFor, logs := cm.storage.GetState()
	cm.currentTerm = term
	cm.votedFor = votedFor
	cm.log = logs
	log.Printf("State Restored succesfully from the storage")
}

// Saves all the CM's persistant state
func (cm *ConsensusModule) persistToStorage() {
	err := cm.storage.SetState(cm.currentTerm, cm.votedFor, cm.log)
	if err == nil {
		log.Printf("State successFully persisted in the term %v", cm.currentTerm)
	}

}

// sends a roundof heartbeats to all the peers, collects their replies and adjusts cm's state.
// func (cm *ConsensusModule) leaderSendHeartBeats() {
// cm.mu.Lock()
// if cm.state != Leader {
// 	cm.mu.Unlock()
// 	return
// }
// savedCurrentTerm := cm.currentTerm
// cm.mu.Unlock()
// for _, peerId := range cm.peerIds {
// 	go func(peerId int) {
// 		cm.mu.Lock()
// 		//cm's view of last index of the peer
// 		ni := cm.nextIndex[peerId]

// 		prevLogIndex := ni - 1
// 		prevLogTerm := -1
// 		if prevLogIndex >= 0 {
// 			prevLogTerm = cm.log[prevLogIndex].Term
// 		}
// 		//pick all entries of log after ni.
// 		entries := cm.log[ni:]

// 		args := AppendEntriesArgs{
// 			Term:         savedCurrentTerm,
// 			LeaderId:     cm.id,
// 			PrevLogIndex: prevLogIndex,
// 			PrevLogTerm:  prevLogTerm,
// 			Entries:      entries,
// 			LeaderCommit: cm.commitIndex,
// 		}
// 		cm.mu.Unlock()

// 		log.Printf("sending new AppendEntries to %v: ni=%d, args=%+v", peerId, ni, args)
// 		var reply AppendEntriesReply
// 		if err := cm.server.Call(peerId, "ConsensusModule.AppendEntries", args, &reply); err == nil {
// 			cm.mu.Lock()
// 			defer cm.mu.Unlock()
// 			//if peer's term is higher, become a follower
// 			if reply.Term > savedCurrentTerm {
// 				log.Printf("term out of date in heartbeat reply")
// 				cm.becomeFollower(reply.Term)
// 				return
// 			}
// 			//if cm is the leader and is in the same term of peer
// 			if cm.state == Leader && savedCurrentTerm == reply.Term {
// 				if reply.Success {
// 					//update the cm with new next index of the peer
// 					cm.nextIndex[peerId] = ni + len(entries)
// 					cm.matchIndex[peerId] = cm.nextIndex[peerId] - 1
// 					log.Printf("AppendEntries reply from %d success: nextIndex := %v, matchIndex := %v", peerId, cm.nextIndex, cm.matchIndex)
// 					//get the commit index of the cm
// 					savedCommitIndex := cm.commitIndex
// 					//iterate over new log entries
// 					for i := cm.commitIndex + 1; i < len(cm.log); i++ {
// 						//if term matches
// 						if cm.log[i].Term == cm.currentTerm {
// 							matchCount := 1
// 							//get the count of peers having the log entry
// 							for _, peerId := range cm.peerIds {
// 								if cm.matchIndex[peerId] >= i {
// 									matchCount++
// 								}
// 							}
// 							//if majority..bring the commit index to i
// 							if matchCount*2 > len(cm.peerIds)+1 {
// 								cm.commitIndex = i
// 							}
// 						}
// 					}
// 					//if commit index is changed..send the entries to the commitchannel
// 					if cm.commitIndex != savedCommitIndex {
// 						log.Printf("leader sets commmitIndex := %d", cm.commitIndex)
// 						cm.newCommitReadyChan <- struct{}{}
// 					}
// 				} else {
// 					cm.nextIndex[peerId] = ni - 1
// 					log.Printf("AppendEntries reply from %d !sucess: nextIndex := %d", peerId, ni-1)
// 				}
// 			}
// 		}
// 	}(peerId)
// }
// }
