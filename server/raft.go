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
// on a command and it can be applied to the all state machines.
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

	//temp storage persistance when no quorum (1 or 2 servers)
	tempstorage        *TempDiskStore
	localraft          bool
	tempcommitchan     chan CommitEntry
	templog            []*pb.LogEntry
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

// The randamized timeout for elections
func (cm *ConsensusModule) electionTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}

// ticking the election timeout. if timed out, the election starts
// and cm becomes candidate
func (cm *ConsensusModule) runElectionTimer() {
	timeoutDuration := cm.electionTimeout()
	cm.mu.Lock()
	termStarted := cm.currentTerm
	cm.mu.Unlock()

	//This loops until
	//1. When election timer is no longer needed
	//2. Election timer expires and CM becomes a condidate
	//for a follower, this keeps ticking for lifetime
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C

		cm.mu.Lock()

		//if its a leader,stop the timer
		if cm.state != Candidate && cm.state != Follower {
			cm.mu.Unlock()
			return
		}

		//if terms dont match. stop the timer
		if termStarted != cm.currentTerm {
			cm.mu.Unlock()
			return
		}

		//if cm is follower election timer timedout
		//start the Election
		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
			cm.startElection()
			cm.mu.Unlock()
			return
		}
		cm.mu.Unlock()
	}
}

// start the election and collects votes through RequestVote RPC
// restarts until there is a leader.
func (cm *ConsensusModule) startElection() {
	cm.state = Candidate
	cm.currentTerm += 1
	savedCurrentTerm := cm.currentTerm
	cm.electionResetEvent = time.Now()
	cm.votedFor = cm.id
	log.Printf("Elections Started. Became Candidate. current term: %v", savedCurrentTerm)
	votesReceived := 1

	//check if there is a quorum. if there is no quorum
	//start local raft to make progress
	cm.server.activepeerIds = cm.server.GetActivePeers()
	if len(cm.server.activepeerIds) < 2 && !cm.localraft {
		log.Printf("Found the active peers (%v) less than majority. Starting local instance", cm.server.activepeerIds)
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

			err := cm.server.peerClients[peerId].Invoke(context.Background(), "/chat.RaftService/RequestVote", args, &reply)
			if err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				//when other RV calls received a majority, then election might have already been done
				if cm.state == Leader {
					cm.startLeader()
					return
				}
				if cm.state == Follower {
					return
				}

				//if cm finds that its term is stale, become follower. else proceed with voting
				if reply.Term > savedCurrentTerm {
					cm.becomeFollower(reply.Term)
					return
				} else if reply.Term == savedCurrentTerm {
					if reply.VoteGranted {
						votesReceived += 1
						if votesReceived*2 > len(cm.peerIds)+1 {
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

// If this cm has been elected as leader. It starts leader duties.
func (cm *ConsensusModule) startLeader() {
	cm.state = Leader
	cm.leaderid = cm.id
	log.Printf("Becomes Leader; term = %d", cm.currentTerm)

	//more info near its definition
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
		log.Printf("Leader is starts heart beats..")
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
			//if any of the above case is true
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
	//send AE's to all the peers
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
			//construct AE request
			args := &pb.AppendEntriesRequest{
				Term:         savedCurrentTerm,
				LeaderId:     cm.id,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries:      entries,
				LeaderCommit: cm.commitIndex,
			}
			cm.mu.Unlock()
			var reply pb.AppendEntriesResponse
			err := cm.server.peerClients[peerId].Invoke(context.Background(), "/chat.RaftService/AppendEntries", args, &reply)
			if err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()

				//if reply term exceeded become follower
				if reply.Term > cm.currentTerm {
					cm.becomeFollower(reply.Term)
					return
				}
				//if the append entry is accepted by the follower
				if cm.state == Leader && savedCurrentTerm == reply.Term {
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
}

// every chat client command is submitted here to the raft or local raft
func (cm *ConsensusModule) Submit(command *pb.Command) bool {
	cm.mu.Lock()
	log.Printf("New command received")
	//if the cm is leader, honor the request
	if cm.state == Leader {
		log.Printf("Main raft honoring the request")
		cm.log = append(cm.log, &pb.LogEntry{Command: command, Term: cm.currentTerm})
		cm.mu.Unlock()
		//persist to storage immediately
		cm.persistToStorage()
		//notify others of the change
		cm.triggerAEChan <- 1
		return true
		// if the cm is in localraft state. cm continues to serve the clients
	} else if cm.localraft {
		log.Printf("Local raft honoring the request")
		//if 1 more peer is connected and its server id is smaller than this one
		//forward the request as small server ID peer is virtual leader in local raft state
		if len(cm.server.activepeerIds) > 0 && cm.server.activepeerIds[0] < cm.id {
			log.Printf("have an active peer. Peer id is %v where as our id is %v", cm.server.activepeerIds, cm.id)
			cm.mu.Unlock()
			cm.forwardToLeader(command)
			//if our id is smaller. then honor it
		} else {
			cm.templog = append(cm.templog, &pb.LogEntry{Command: command, Term: cm.currentTerm})
			cm.mu.Unlock()
			//persist the state immediately
			cm.persistToStorage()
			cm.triggerLocalAEChan <- 1
		}
		return true
		//if in raft state and is a follower, forward it to leader
	} else {
		cm.mu.Unlock()
		cm.forwardToLeader(command)
		return true
	}
}

// becomes a floower ans changes its state
func (cm *ConsensusModule) becomeFollower(term int64) {
	// log.Printf("Become follower with term = %v, log = %v", term, cm.log)
	cm.state = Follower
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()
	log.Printf("Became follower. Starting the election timer.")
	go cm.runElectionTimer()
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

// func which checks the last applied entry and sends the remaining
// entries to the commitchannel
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

// once the commit chan sender comes up with the entries that need to be commited
// commit the entries
func (cm *ConsensusModule) commitEntries() {
	for {
		log.Printf("CommitEntries function waits for entries.")
		entry := <-cm.commitChan
		cm.server.persistData(entry)
	}
}

// helper function for the RequestVote RPC
func (cm *ConsensusModule) RequestVoteHelper(req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	reply := &pb.RequestVoteResponse{}
	if cm.state == Dead {
		return reply, nil
	}
	lastLogIndex, LastLogTerm := cm.lastLogIndexAndTerm()
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
	return reply, nil
}

// helper function for ForwardLeader RPC
func (cm *ConsensusModule) ForwardLeaderHelper(req *pb.ForwardLeaderRequest) (*pb.ForwardLeaderResponse, error) {
	reply := &pb.ForwardLeaderResponse{}
	if cm.state == Dead {
		return reply, nil
	}

	log.Printf("Received the forwarded Command from other peer.")

	reply.Success = false
	//if cm is leader of is progressing with localraft
	if cm.state == Leader || cm.localraft {
		log.Printf("I'm the leader/virtual leader. Processing the Request.")
		if cm.Submit(req.Command) {
			reply.Success = true
			reply.LeaderId = cm.id
		}
		//if the cm is not leader . reply with correct leader id
	} else {
		log.Printf("I'm not the Leader. Sending back the correct leader info")
		reply.Success = false
		reply.LeaderId = cm.leaderid
	}

	return reply, nil
}

// helper function for AppendEntries RPC
// executed by followers
func (cm *ConsensusModule) AppendEntriesHelper(req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	reply := &pb.AppendEntriesResponse{}
	if cm.state == Dead {
		return reply, nil
	}
	//update the current leader details
	cm.leaderid = req.LeaderId

	//if a term received is greater than the current term..become a follower
	if req.Term > cm.currentTerm {
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
			//where the term mismatches with an entry from the leader
			// -newEntriesIndex points to the End of the entries, or an index
			//where the term mismatches with the corresponding log entry
			if newEntriesIndex < len(req.Entries) {
				cm.log = append(cm.log[:logInsertIndex], req.Entries[newEntriesIndex:]...)
			}

			//set commit index
			if req.LeaderCommit > cm.commitIndex {
				cm.commitIndex = int64(intMin(int(req.LeaderCommit), len(cm.log)-1))
				log.Printf("setting commitIndex=%d", cm.commitIndex)
				cm.newCommitReadyChan <- 1
			}
		}
	}

	reply.Term = cm.currentTerm
	cm.persistToStorage()
	return reply, nil
}

// followers diverting the client requests to the leader
func (cm *ConsensusModule) forwardToLeader(command *pb.Command) {
	if cm.state != Leader {
		go func() {

			args := &pb.ForwardLeaderRequest{
				Command: command,
			}
			var reply pb.ForwardLeaderResponse
			cm.mu.Lock()
			//if no quorum, send the request to lower peer Id server.
			peerid := cm.leaderid
			if cm.localraft {
				peerid = cm.server.activepeerIds[0]
			}

			cm.mu.Unlock()
			log.Printf("As I'm not the leader, forwarding to leader server %v", peerid)
			err := cm.server.peerClients[peerid].Invoke(context.Background(), "/chat.RaftService/ForwardLeader", args, &reply)
			if err == nil {
				if reply.Success {
					log.Printf("Success forwarding the request to leader")
					cm.newCommitReadyChan <- int64(1)
				} else {
					//if sent to wrong server, forward to right leader.
					cm.leaderid = reply.LeaderId
					cm.forwardToLeader(command)
				}
			}
		}()
	}
}

// helper function for GetTempLogs RPC
func (cm *ConsensusModule) GetTempLogsHelper(req *pb.MergeRequest) (*pb.MergeResponse, error) {
	reply := &pb.MergeResponse{}
	if cm.state == Dead {
		return reply, nil
	}
	log.Printf("Received a Merge Request from leader. Checking if any temp log to merge.")
	cm.mu.Lock()
	defer cm.mu.Unlock()
	//update the leader
	cm.leaderid = req.Leaderid
	//get the lowest server id connected peer.
	lowestpeerIdnumber := cm.id
	for _, peerid := range cm.server.activepeerIds {
		if lowestpeerIdnumber > peerid {
			lowestpeerIdnumber = peerid
		}
	}
	//if I am the lowest, submit my temp log to leader to merge.
	if cm.localraft && lowestpeerIdnumber == cm.id {
		log.Printf("I am the lowest server Id peer. Sending my temp log to leader")
		reply.Entries = cm.templog
		reply.IsTempLog = true
		//if active peer have the lowest server Id, it might have already sent the temp log
		//as both the logs are identical.Ignore this request.
	} else {
		log.Printf("My peer have the lowest server id. Ignoring the merge request")
		reply.IsTempLog = false
	}
	//close the local raft to shift back to raft mode.
	cm.closeLocalRaft()
	return reply, nil

}

func (cm *ConsensusModule) SendLocalAEsHelper(req *pb.LocalAERequest) (*pb.LocalAEResponse, error) {
	reply := &pb.LocalAEResponse{}
	if cm.state == Dead {
		return reply, nil
	}
	log.Printf("Got the Local AE from my virtual leader. Processing it")
	//directly append the log
	cm.templog = append(cm.templog, req.Entries...)
	//get the commit index
	savedtempcommitindex := cm.tempcommitindex
	commitentries := cm.templog[savedtempcommitindex:]
	log.Printf("Entries to be committed to local raft: %v ", commitentries)
	for _, entry := range commitentries {
		cm.tempcommitchan <- CommitEntry{
			Command: entry.Command,
			Index:   cm.tempcommitindex,
			Term:    cm.currentTerm,
		}
		cm.tempcommitindex++
	}
	reply.Success = true
	return reply, nil

}

// func to merge all the temp logs from other peers, when there was no leader.
// this way we can progress with no quorum
func (cm *ConsensusModule) MergeTempLogs() {
	log.Printf("Merging all the temp logs from the peers")
	templogslist := make([][]*pb.LogEntry, 6)
	cm.mu.Lock()
	templogslist[cm.id] = cm.templog
	cm.mu.Unlock()
	//send GetTempLogs RPC to all the peers and get their templogs if any.
	for _, peerId := range cm.peerIds {
		go func(peerId int64) {
			args := &pb.MergeRequest{
				Leaderid: cm.id,
			}
			var reply pb.MergeResponse
			err := cm.server.peerClients[peerId].Invoke(context.Background(), "/chat.RaftService/GetTempLogs", args, &reply)
			if err == nil {
				if reply.IsTempLog {
					log.Printf("Received a temp log from %v", peerId)
					templogslist[peerId] = reply.Entries
				}
			}
		}(peerId)
	}
	//append all the temp logs to raft log
	cm.mu.Lock()
	for _, templogs := range templogslist {
		cm.log = append(cm.log, templogs...)
	}
	log.Printf("Merged raft log : %v", cm.log)
	cm.mu.Unlock()

	//once the templogs are collected, close local raft
	//to switch back to raft mode.
	cm.closeLocalRaft()

}

// when no leader, we initialize the local raft mode which ensure progress
// and eventual consistency
func (cm *ConsensusModule) InitialiseLocalRaft() {
	go cm.tempcommitEntries()
	//get the latest state of the raft
	cm.tempstorage.tempdiskstore = cm.storage.diskstore
	//initialise templog and commit index
	cm.templog = make([]*pb.LogEntry, 0)
	cm.tempcommitindex = 0

	log.Printf("Local Raft succesfully initialised")
	cm.localraft = true

	//if there is one more server connected to it
	//if this is the smaller server id . Then behave as a virtual leader
	go func() {
		// Immediately send AEs to peers.
		cm.SendLocalAEs()
		for {
			doSend := false
			ok := <-cm.triggerLocalAEChan
			if ok == 1 {
				doSend = true
			} else {
				return
			}
			if doSend {
				log.Printf("Sending local AE's")
				cm.SendLocalAEs()
			}
		}
	}()

}

// in local raft mode, when 1 more peer is connected
// this func sends the templog from the lowest server id peer to another.
// this ensures consistency between both the peers.
func (cm *ConsensusModule) SendLocalAEs() {
	cm.mu.Lock()
	tempcommmitindex := cm.tempcommitindex
	cm.mu.Unlock()
	activepeers := cm.server.GetActivePeers()
	if len(activepeers) > 0 && activepeers[0] > cm.id {
		log.Printf("Active peer is connected %v", activepeers[0])
		ni := cm.nexttempindex
		// pick logs which are not present in peer templog.
		entries := cm.templog[ni:]
		log.Printf("Entries to send to peer : %v", entries)
		args := &pb.LocalAERequest{
			Entries:     entries,
			Commitindex: tempcommmitindex,
			Nextindex:   ni,
		}
		var reply pb.AppendEntriesResponse
		err := cm.server.peerClients[activepeers[0]].Invoke(context.Background(), "/chat.RaftService/LocalAEs", args, &reply)
		if err == nil {
			log.Printf("sent the localAE to follower. received success")
			cm.nexttempindex = ni + int64(len(entries))
		}
	}
	//commit the latest entries
	entries := cm.templog[tempcommmitindex:]
	log.Printf("Entries that need to be commited  %v", entries)
	for i, entry := range entries {
		cm.tempcommitchan <- CommitEntry{
			Command: entry.Command,
			Index:   tempcommmitindex + int64(i) + 1,
			Term:    cm.currentTerm,
		}
	}
	//update the temp log commit index
	cm.tempcommitindex = int64(int(cm.tempcommitindex) + len(entries))
	log.Printf("New Entries commited to temp storage")
}

// function to commit entries in temp storage
func (cm *ConsensusModule) tempcommitEntries() {
	for {
		log.Printf("Temp CommitEntries function waits for temp commits.")
		entry := <-cm.tempcommitchan
		cm.server.persistTempData(entry)
	}
}

// close the local raft instance
func (cm *ConsensusModule) closeLocalRaft() {
	log.Printf("Closing the local raft as leader is Elected. Switching back to raft mode")
	cm.localraft = false
	cm.commitIndex = 0
	cm.nexttempindex = 0
}

// server the client with the group details based on the mode of cm
func (cm *ConsensusModule) GetGroup(groupname string) *pb.Group {
	//if local raft mode is on, serve from the temp storage
	if cm.localraft {
		log.Printf("Fetching the group details from the temp storage..")
		group, _ := cm.tempstorage.GetGroup(groupname)
		return group
	} else {
		log.Printf("Fetching the group details from the raft storage..")
		group, _ := cm.storage.GetGroup(groupname)
		return group
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
	//in local raft mode, persist to temp storage
	if cm.localraft {
		err := cm.tempstorage.SetState(cm.currentTerm, cm.votedFor, cm.log)
		if err == nil {
			log.Printf("State successFully temp persisted in the term %v", cm.currentTerm)
			return

		}
		//in raft mode, persist to raft storage.
	} else {
		err := cm.storage.SetState(cm.currentTerm, cm.votedFor, cm.log)
		if err == nil {
			return
		}
	}
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

// helper minimum function
func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
