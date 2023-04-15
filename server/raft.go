// core raft implementation
package chat_server

import (
	"chat-system/pb"
	"context"
	"log"
	"math/rand"
	"sync"
	"time"
)

const DebugCM = 1

// Command struct for the chat service application
type Command struct {
	Event   string
	Command *pb.Command
}

// Data reported by the raft to the commit channel. Each commit notifies that the consensus is achieved
// on a command and it can be applied to the clients state machine.
type CommitEntry struct {
	//command being commited
	Command *pb.Command

	//Log index where the client command is being commited
	Index int64

	//Term at which the client command is commited
	Term int64
}

type LogEntry struct {
	Command Command
	Term    int64
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
	id int64

	//keep track of leader
	leaderid int64

	// peerID's list of our peers in the cluster
	peerIds []int64

	//server of this CM
	server *Server

	//persistance storage of app data and raft data
	storage *DiskStore

	commitChan chan CommitEntry

	//internal notification channel used by go routines to notify that newly commited entries may be sent
	//on commitChan
	newCommitReadyChan chan int64

	// triggerAEChan is an internal notification channel used to trigger
	// sending new AEs to followers when interesting changes occurred.
	triggerAEChan chan int

	//Raft state persistence on all the servers
	currentTerm int64
	votedFor    int64
	log         []*pb.LogEntry

	//temp storage persistance when no quorum
	tempstorage        *TempDiskStore
	localraft          bool
	tempcommitchan     chan CommitEntry
	templog            []*pb.LogEntry
	templogterm        int64
	triggerLocalAEChan chan int
	tempcommitindex    int64
	nexttempindex      int64

	//raft state volatile on all servers
	commitIndex        int64
	lastApplied        int64
	state              CMState
	electionResetEvent time.Time

	//volatile raft state on the leaders
	nextIndex  map[int64]int64
	matchIndex map[int64]int64
}

// new consensus module creation with the given ID, list of peerID's and server.
// Ready channel signals CM that all the peers are connected and its safe to start
func NewConsensusModule(id int64, peerIds []int64, server *Server, ready <-chan int, commitChan chan CommitEntry) *ConsensusModule {
	log.Printf("Initialising raft consensus..")
	cm := new(ConsensusModule)
	cm.id = id
	cm.leaderid = -1
	cm.peerIds = peerIds
	cm.server = server
	cm.commitChan = commitChan
	cm.tempcommitchan = make(chan CommitEntry, 10)
	cm.newCommitReadyChan = make(chan int64, 16)
	cm.triggerAEChan = make(chan int)
	cm.triggerLocalAEChan = make(chan int)
	cm.tempcommitindex = 0
	cm.nexttempindex = 0
	cm.localraft = false
	cm.state = Follower
	cm.votedFor = -1
	cm.lastApplied = -1
	cm.nextIndex = make(map[int64]int64)
	cm.matchIndex = make(map[int64]int64)
	cm.commitIndex = -1
	cm.storage = NewDiskStore(cm.id)
	cm.tempstorage = NewTempDiskStore(cm.id)

	//restore the state from the persisted storage
	if cm.storage.HasData() {
		log.Printf("Found Persistant data, Retreiving..")
		cm.restoreFromStorage()
	}
	log.Printf("No Persistant data found")
	go func() {
		//the CM will be inactive until ready is signaled ,then it starts a countdown
		<-ready
		go cm.commitEntries()
		for {
			cm.mu.Lock()
			cm.electionResetEvent = time.Now()
			cm.mu.Unlock()
			cm.runElectionTimer()
		}

	}()

	go cm.commitChanSender()
	return cm
}

// Reprts the current state of the CM
func (cm *ConsensusModule) Report() (id int64, term int64, isLeader bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.id, cm.currentTerm, cm.state == Leader
}

func (cm *ConsensusModule) Submit(command *pb.Command) bool {
	cm.mu.Lock()

	log.Printf("Submit received by %v", cm.state)
	if cm.state == Leader {
		cm.log = append(cm.log, &pb.LogEntry{Command: command, Term: cm.currentTerm})
		cm.mu.Unlock()
		cm.persistToStorage()
		// log.Printf("... log=%v", cm.log)
		cm.triggerAEChan <- 1

		return true
	} else if cm.localraft {
		log.Printf("Theere is no leader yet. Still trying to progess")
		if len(cm.server.activepeerIds) > 0 && cm.server.activepeerIds[0] < cm.id {
			cm.mu.Unlock()
			cm.forwardToLeader(command)
		} else {
			cm.templog = append(cm.templog, &pb.LogEntry{Command: command, Term: cm.currentTerm})
			cm.mu.Unlock()
			cm.persistToTempStorage()
			log.Printf("Setting the local AE chan ..")
			cm.triggerLocalAEChan <- 1
		}

		return true

	} else {
		cm.mu.Unlock()
		cm.forwardToLeader(command)
		return true

	}
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

func (cm *ConsensusModule) RequestVoteHelper(req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	reply := &pb.RequestVoteResponse{}
	if cm.state == Dead {
		return reply, nil
	}
	lastLogIndex, LastLogTerm := cm.lastLogIndexAndTerm()
	// log.Printf("Working with GRPC")
	// log.Printf("RequestVote: %v [currentTerm = %v, votedFor = %v, log index/term= (%d,%d)]", req, cm.currentTerm, cm.votedFor, lastLogIndex, LastLogTerm)

	//if a term received is greater than the current term..become a follower
	if req.Term > cm.currentTerm {
		// log.Printf("Current term out of date..")
		cm.becomeFollower(req.Term)
	}

	//if term matches
	//if cm didnot vote for any candidate yet..vote
	//if already voted..ignore
	if cm.currentTerm == req.Term &&
		(cm.votedFor == -1 || cm.votedFor == req.CandidateId) &&
		(req.LastLogTerm > LastLogTerm ||
			(req.LastLogTerm == LastLogTerm && req.LastLogIndex >= lastLogIndex)) {
		reply.VoteGranted = true
		cm.votedFor = req.CandidateId
		cm.electionResetEvent = time.Now()
	} else {
		reply.VoteGranted = false
	}

	reply.Term = cm.currentTerm
	cm.persistToStorage()
	// log.Printf("Requestvote Reply of %v: %v", cm.server.serverId, reply)
	return reply, nil
}

func (cm *ConsensusModule) ForwardLeaderHelper(req *pb.ForwardLeaderRequest) (*pb.ForwardLeaderResponse, error) {
	reply := &pb.ForwardLeaderResponse{}
	if cm.state == Dead {
		return reply, nil
	}

	log.Printf("Got the forwarded Command: %v", cm.leaderid)

	reply.Success = false
	//if cm is leader
	if cm.state == Leader || cm.localraft {
		log.Printf("I'm the leader/virtual leader, so Commiting to log and sending to followers")
		if cm.Submit(req.Command) {
			reply.Success = true
			reply.LeaderId = cm.id
		}
	} else {
		reply.Success = false
		reply.LeaderId = cm.leaderid
	}

	return reply, nil
}

func (cm *ConsensusModule) AppendEntriesHelper(req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	reply := &pb.AppendEntriesResponse{}
	if cm.state == Dead {
		return reply, nil
	}
	// log.Printf("AppendEntries: %v", req)
	//update the current leader details
	cm.leaderid = req.LeaderId

	//if a term received is greater than the current term..become a follower
	if req.Term > cm.currentTerm {
		// log.Printf("Current term out of date..")
		cm.becomeFollower(req.Term)
	}
	reply.Success = false
	if req.Term == cm.currentTerm {
		//it may happen that two leaders exists in same term..make later as follower
		if cm.state != Follower {
			cm.becomeFollower(req.Term)
		}
		cm.electionResetEvent = time.Now()

		//Does our log contains an Entry at PrevLogIndex whose term matches
		if req.PrevLogIndex == -1 ||
			(int(req.PrevLogIndex) < len(cm.log) && req.PrevLogTerm == cm.log[req.PrevLogIndex].Term) {
			reply.Success = true
			//find an insertion point - where theres a term mismatch between
			//the existing log starting at PrevLogIndex+1 and the new entries sent in the RPC
			logInsertIndex := req.PrevLogIndex + 1
			newEntriesIndex := 0

			for {
				if int(logInsertIndex) >= len(cm.log) || newEntriesIndex >= len(req.Entries) {
					break
				}
				if cm.log[logInsertIndex].Term != req.Entries[newEntriesIndex].Term {
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
			if newEntriesIndex < len(req.Entries) {
				// log.Printf("... inserting %v from index %d", req.Entries[newEntriesIndex:], logInsertIndex)
				cm.log = append(cm.log[:logInsertIndex], req.Entries[newEntriesIndex:]...)
				// log.Printf("... log is now: %v", cm.log)
			}

			//set commit index
			if req.LeaderCommit > cm.commitIndex {
				cm.commitIndex = int64(intMin(int(req.LeaderCommit), len(cm.log)-1))
				log.Printf("... setting commitIndex=%d", cm.commitIndex)
				cm.newCommitReadyChan <- 1
			}
		}
	}

	reply.Term = cm.currentTerm
	cm.persistToStorage()
	// log.Printf("AppendEntries reply : %v", reply)
	return reply, nil
}

func (cm *ConsensusModule) GetTempLogsHelper(req *pb.MergeRequest) (*pb.MergeResponse, error) {
	reply := &pb.MergeResponse{}
	if cm.state == Dead {
		return reply, nil
	}

	log.Printf("Seems like a leader is elected. Checking my templog to send to leader")
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.leaderid = req.Leaderid
	var tempentries []*pb.LogEntry
	lowestpeerIdnumber := cm.id
	for _, peerid := range cm.server.activepeerIds {
		if lowestpeerIdnumber > peerid {
			lowestpeerIdnumber = peerid
		}
	}
	if cm.localraft && lowestpeerIdnumber == cm.id {
		log.Printf("Seems like Im the smallest Peer. Sending my temp log")
		tempentries = cm.templog
		reply.Entries = tempentries
		reply.IsTempLog = true
	} else {
		log.Printf("I dont have to send. My smaller active peer might have sent it")
		reply.IsTempLog = false
	}
	cm.closeLocalRaft()
	return reply, nil

}

func (cm *ConsensusModule) SendLocalAEsHelper(req *pb.LocalAERequest) (*pb.LocalAEResponse, error) {
	reply := &pb.LocalAEResponse{}
	if cm.state == Dead {
		return reply, nil
	}

	log.Printf("Got the Local AE from my virtual leader. Honoring it")

	//directly append the log
	cm.templog = append(cm.templog, req.Entries...)

	//get the commit index
	savedtempcommitindex := cm.tempcommitindex

	log.Printf("commit index is %v", cm.tempcommitindex)

	commitentries := cm.templog[savedtempcommitindex:]
	log.Printf("commitentries %v ", commitentries)
	for _, entry := range commitentries {
		cm.tempcommitchan <- CommitEntry{
			Command: entry.Command,
			Index:   cm.tempcommitindex + 1,
			Term:    cm.currentTerm,
		}
		cm.tempcommitindex++
	}

	reply.Success = true
	return reply, nil

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
			// log.Printf("in election timer, state= %s, bailing out as Im the leader already", cm.state)
			cm.mu.Unlock()
			return
		}

		//if terms dont match. stop the timer
		if termStarted != cm.currentTerm {
			// log.Printf("in election timer, term changed from %d to %d, bailing out", termStarted, cm.currentTerm)
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
	cm.server.activepeerIds = cm.server.GetActivePeers()
	if len(cm.server.activepeerIds) <= 2 && !cm.localraft {

		//if no quorum is present
		log.Printf("Checking connected peers number..")
		log.Printf("number of active peers : %v", len(cm.server.activepeerIds))

		log.Printf("Found the active peer less than majority. Starting local instance.")
		cm.templogterm = savedCurrentTerm
		cm.InitialiseLocalRaft()
	}

	//send RequestVote RPC's to all the servers Concurrently, to collect the votes
	for _, peerId := range cm.peerIds {
		go func(peerId int64) {
			cm.mu.Lock()
			savedLastLogIndex, savedLastLogTerm := cm.lastLogIndexAndTerm()
			cm.mu.Unlock()

			args := &pb.RequestVoteRequest{
				Term:         savedCurrentTerm,
				CandidateId:  cm.id,
				LastLogIndex: savedLastLogIndex,
				LastLogTerm:  savedLastLogTerm,
			}
			var reply pb.RequestVoteResponse

			// log.Printf("Sending RequestVote to %d: %+v", peerId, args)
			// cm.server.ReconnectToPeer(peerId)

			err := cm.server.peerClients[peerId].Invoke(context.Background(), "/chat.RaftService/RequestVote", args, &reply)
			if err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				// log.Printf("received RequestVoteReply %+v", &reply)
				//As we got a response check if the peer is connected to the cm.
				if cm.server.peerClients[peerId] == nil {
					log.Printf("Peer %v connected back.", peerId)
					cm.server.ReconnectToPeer(peerId)
				}
				//when other RV calls received a majority, then election might have already been done
				if cm.state == Leader {
					// log.Printf("While waiting for reply, State changed to %v", cm.state)
					cm.startLeader()
					return
				}
				if cm.state == Follower {
					// log.Printf("While waiting for reply, State changed to Follower")
					return
				}

				//if cm finds that its term is stale, become follower. else proceed with voting
				if reply.Term > savedCurrentTerm {
					// log.Printf("term out of date in RequestVoteReply")
					cm.becomeFollower(reply.Term)
					return
				} else if reply.Term == savedCurrentTerm {
					if reply.VoteGranted {
						votesReceived += 1
						if votesReceived*2 > len(cm.peerIds)+1 {
							// log.Printf("Wins election with %d other votes", votesReceived-1)
							go cm.startLeader()
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
func (cm *ConsensusModule) becomeFollower(term int64) {
	// log.Printf("Become follower with term = %v, log = %v", term, cm.log)
	cm.state = Follower
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()

	go cm.runElectionTimer()
}

func (cm *ConsensusModule) startLeader() {
	cm.state = Leader
	cm.leaderid = cm.id
	// log.Printf("Becomes Leader; term = %d, log=%v", cm.currentTerm, cm.log)

	//As I'm the leader, merging all the temp logs
	cm.MergeTempLogs()

	for _, peerId := range cm.peerIds {
		cm.nextIndex[peerId] = int64(len(cm.log))
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
			case ok := <-cm.triggerAEChan:
				if ok == 1 {
					// log.Printf("AE channnel triggered..")
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

func (cm *ConsensusModule) MergeTempLogs() {
	log.Printf("Leader is elected. Merging all the temp logs")
	//closing my localraft
	templogslist := make([][]*pb.LogEntry, 6)
	templogslist[cm.id] = cm.templog
	for _, peerId := range cm.peerIds {
		go func(peerId int64) {
			cm.mu.Lock()
			args := &pb.MergeRequest{
				Leaderid: cm.id,
			}
			cm.mu.Unlock()
			var reply pb.MergeResponse
			err := cm.server.peerClients[peerId].Invoke(context.Background(), "/chat.RaftService/GetTempLogs", args, &reply)
			if err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				if reply.IsTempLog {
					templogslist[peerId] = reply.Entries
				}
			}
		}(peerId)
	}
	log.Printf("Picked all the templogs")
	cm.MergetoRaftLog(templogslist)

	cm.closeLocalRaft()

}

func (cm *ConsensusModule) MergetoRaftLog(templogslist [][]*pb.LogEntry) {
	for _, templogs := range templogslist {
		cm.log = append(cm.log, templogs...)
	}
	log.Printf("Merged raft log : %v", cm.log)
	// cm.newCommitReadyChan <- int64(1)

}

func (cm *ConsensusModule) closeLocalRaft() {
	log.Printf("Closing the local raft as leader is back")
	log.Printf("Local Raft succesfully closed. Into the main Raft again")
	cm.localraft = false
	cm.commitIndex = 0
	cm.nexttempindex = 0
}

func (cm *ConsensusModule) forwardToLeader(command *pb.Command) {
	if cm.state != Leader {
		go func() {
			log.Printf("Sorry, I'm not the leader . forwarding ")

			args := &pb.ForwardLeaderRequest{
				Command: command,
			}
			var reply pb.ForwardLeaderResponse
			cm.mu.Lock()
			peerid := cm.leaderid
			if cm.localraft {
				peerid = cm.server.activepeerIds[0]
			}

			cm.mu.Unlock()
			// log.Printf("Sending RequestVote to %d: %+v", peerId, args)
			log.Printf("forwarding to Leader server %v", cm.leaderid)
			err := cm.server.peerClients[peerid].Invoke(context.Background(), "/chat.RaftService/ForwardLeader", args, &reply)
			if err == nil {

				log.Printf("received LeaderReply %v", &reply)

				if reply.Success {
					log.Printf("Succesfully forwarded the request to leader")
					cm.newCommitReadyChan <- int64(1)
				} else {
					cm.leaderid = reply.LeaderId
					cm.forwardToLeader(command)
				}
			}
		}()

	}
}

// sends a round of AEs to all peers, collects their
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
		go func(peerId int64) {
			cm.mu.Lock()
			ni := cm.nextIndex[int64(peerId)]
			prevLogIndex := ni - 1
			prevLogTerm := int64(-1)
			if prevLogIndex >= 0 {
				prevLogTerm = cm.log[prevLogIndex].Term
			}
			entries := cm.log[ni:]
			// log.Printf("Entries : %v",entries)

			args := &pb.AppendEntriesRequest{
				Term:         savedCurrentTerm,
				LeaderId:     cm.id,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries:      entries,
				LeaderCommit: cm.commitIndex,
			}
			cm.mu.Unlock()
			// log.Printf("sending AppendEntries to %v: ni=%d, args=%+v", peerId, ni, args)
			var reply pb.AppendEntriesResponse
			err := cm.server.peerClients[peerId].Invoke(context.Background(), "/chat.RaftService/AppendEntries", args, &reply)
			if err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				//As we got a response check if the peer is connected to the cm.
				// log.Printf("AppendEntries reply from %d !success: nextIndex := %d", peerId, ni-1)

				if reply.Term > cm.currentTerm {
					// log.Printf("term out of date in heartbeat reply")
					cm.becomeFollower(reply.Term)
					return
				}

				if cm.state == Leader && savedCurrentTerm == reply.Term {
					//if the append entry is accepted by the follower
					if reply.Success {
						cm.nextIndex[peerId] = ni + int64(len(entries))
						cm.matchIndex[peerId] = cm.nextIndex[peerId] - 1

						savedCommitIndex := cm.commitIndex
						for i := int(cm.commitIndex) + 1; i < len(cm.log); i++ {
							if cm.log[i].Term == cm.currentTerm {
								matchCount := 1
								for _, peerId := range cm.peerIds {
									if cm.matchIndex[peerId] >= int64(i) {
										matchCount++
									}
								}
								if matchCount*2 > len(cm.peerIds)+1 {
									cm.commitIndex = int64(i)
								}
							}
						}
						// log.Printf("AppendEntries reply from %d success: nextIndex := %v, matchIndex := %v; commitIndex := %d", peerId, cm.nextIndex, cm.matchIndex, cm.commitIndex)
						if cm.commitIndex != savedCommitIndex {
							log.Printf("leader sets commitIndex := %d", cm.commitIndex)
							// Commit index changed: the leader considers new entries to be
							// committed. Send new entries on the commit channel to this
							// leader's clients, and notify followers by sending them AEs.
							cm.newCommitReadyChan <- int64(1)
							cm.triggerAEChan <- 1
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
								cm.nextIndex[peerId] = int64(lastIndexOfTerm + 1)
							} else {
								cm.nextIndex[peerId] = reply.ConflictIndex
							}
						} else {
							cm.nextIndex[peerId] = reply.ConflictIndex
						}

					}
				}
			}
		}(peerId)
	}
	// log.Printf("Leader Send AE's finished")
}

func (cm *ConsensusModule) commitChanSender() {
	for {
		<-cm.newCommitReadyChan
		//find which entries we need to apply
		cm.mu.Lock()
		savedTerm := cm.currentTerm
		//get last applied index
		savedLastApplied := cm.lastApplied
		var entries []*pb.LogEntry
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
				Index:   savedLastApplied + int64(i) + 1,
				Term:    savedTerm,
			}
		}
		log.Printf("CommitChanSender done")
	}

}

func (cm *ConsensusModule) commitEntries() {
	for {
		log.Printf("It is in the CommitEntries function")
		entry := <-cm.commitChan
		cm.server.persistData(entry)
	}
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
func (cm *ConsensusModule) lastLogIndexAndTerm() (int64, int64) {
	if len(cm.log) > 0 {
		lastIndex := len(cm.log) - 1
		return int64(lastIndex), cm.log[lastIndex].Term
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
	// log.Printf("Trying to persist data1..")
	err := cm.storage.SetState(cm.currentTerm, cm.votedFor, cm.log)
	if err == nil {
		return
		// log.Printf("State successFully persisted in the term %v", cm.currentTerm)
	}

}

// Saves all the CM's persistant state
func (cm *ConsensusModule) persistToTempStorage() {
	log.Printf("Trying to persist tempdata1..")
	err := cm.tempstorage.SetState(cm.currentTerm, cm.votedFor, cm.log)
	if err == nil {
		log.Printf("State successFully temp persisted in the term %v", cm.currentTerm)
		return

	}

}
func (cm *ConsensusModule) tempcommitEntries() {
	for {
		log.Printf("It is in the Temp CommitEntries function")
		entry := <-cm.tempcommitchan
		cm.server.persistTempData(entry)
	}
}

func (cm *ConsensusModule) pullRaftState() {

	cm.tempstorage.tempdiskstore = cm.storage.diskstore
	// cm.tempstorage.templog = cm.storage.replicatedlog

	log.Printf("Complete raft status is pulled to temp successfully")

}

func (cm *ConsensusModule) InitialiseLocalRaft() {
	go cm.tempcommitEntries()
	//get the latest state of the raft
	cm.pullRaftState()
	//initialise templog
	cm.templog = make([]*pb.LogEntry, 0)
	cm.tempcommitindex = 0

	log.Printf("Local Raft succesfully initialised")

	cm.localraft = true

	//if there is one more server connected to it
	//if this is the smaller server id . Then behanve as a virtual leader

	go func(heartbeatTimeout time.Duration) {
		// Immediately send AEs to peers.

		t := time.NewTimer(heartbeatTimeout)
		defer t.Stop()
		for {
			doSend := false
			ok := <-cm.triggerLocalAEChan
			if ok == 1 {
				// log.Printf("AE channnel triggered..")
				doSend = true
			} else {
				return
			}

			if doSend {
				// If this isn't a leader any more, stop the heartbeat loop.
				log.Printf("Received a send local AE's request. Serving")
				cm.SendLocalAEs()
			}
		}
	}(50 * time.Millisecond)

}

func (cm *ConsensusModule) SendLocalAEs() {
	cm.mu.Lock()
	tempcommmitindex := cm.tempcommitindex

	// log.Printf("sending AppendEntries to %v: ni=%d, args=%+v", peerId, ni, args)
	cm.mu.Unlock()
	activepeers := cm.server.GetActivePeers()
	log.Printf("Active peers : %v", activepeers)
	if len(activepeers) > 0 && activepeers[0] > cm.id {
		ni := cm.nexttempindex
		// pick logs which are not present in peer templog.
		entries := cm.templog[ni:]
		log.Printf("Testing1 %v", entries)
		args := &pb.LocalAERequest{
			Entries:     entries,
			Commitindex: tempcommmitindex,
			Nextindex:   ni,
		}
		// log.Printf("Found a peer to whom we need to submit the localAE")
		var reply pb.AppendEntriesResponse
		err := cm.server.peerClients[activepeers[0]].Invoke(context.Background(), "/chat.RaftService/LocalAEs", args, &reply)
		if err == nil {
			log.Printf("sent the localAE to follower. received success")
			cm.nexttempindex = ni + int64(len(entries))
		}
	}
	entries := cm.templog[tempcommmitindex:]
	log.Printf("Testing2 %v", entries)
	for i, entry := range entries {

		cm.tempcommitchan <- CommitEntry{
			Command: entry.Command,
			Index:   tempcommmitindex + int64(i) + 1,
			Term:    cm.currentTerm,
		}
	}
	cm.tempcommitindex = int64(int(cm.tempcommitindex) + len(entries))
	log.Printf("Appended into the temp log successfully")

}

func (cm *ConsensusModule) GetGroup(groupname string) *pb.Group {

	if cm.localraft {
		log.Printf("Fetcing the group details from the temporary storage..")
		group, _ := cm.tempstorage.GetGroup(groupname)
		return group
	} else {
		group, _ := cm.storage.GetGroup(groupname)
		return group
	}
}
