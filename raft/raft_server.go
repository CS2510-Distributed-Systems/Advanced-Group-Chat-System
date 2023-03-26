//raft server for the raft consensus ,,odul. Exposes the Raft to the network and
//enables RPCs between Raft Peers

package raft

import "sync"


type server struct{
	mu sync.Mutex

	serverId int
	peerIds []int

	cm *ConsensusModule
	
}