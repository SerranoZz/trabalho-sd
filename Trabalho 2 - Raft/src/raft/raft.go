package raft

import (
	"labrpc"
	"math/rand"
	"sync"
	"time"
)

type Role int

const (
	Follower Role = iota
	Candidate
	Leader

	//max 10 heartbeats per second
	heartbeatInterval = 100 * time.Millisecond
)

type AppendEntriesArgs struct {
	Term     int
	LeaderId int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

type ApplyMsg struct {
	Index       int
	Command     interface{}
	UseSnapshot bool   // ignore for lab2; only used in lab3
	Snapshot    []byte // ignore for lab2; only used in lab3
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm    int
	votedFor       int
	role           Role // role of server (0: follower, 1: candidate, 2: leader)
	votesReceived  int
	leaderId       int
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer
}

type RequestVoteArgs struct {
	Term        int
	CandidateId int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	var term int
	var isleader bool
	term = rf.currentTerm
	isleader = rf.role == Leader

	return term, isleader
}

func (rf *Raft) persist() {

}

func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
}

func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

func (rf *Raft) Kill() {

}

func randomTimeout() time.Duration {
	//time between 250 and 400 ms
	const minTimeout = 250
	const timeoutInterval = 150 + 1
	return time.Duration(minTimeout+rand.Intn(timeoutInterval)) * time.Millisecond
}

func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.role = Follower
	rf.electionTimer = time.NewTimer(randomTimeout())
	rf.heartbeatTimer = time.NewTimer(heartbeatInterval)
	rf.votesReceived = 0
	rf.leaderId = -1

	// Your initialization code here (2A, 2B, 2C).
	go rf.routine() // start routine for server

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	return rf
}

func (rf *Raft) convertToFollower() {
	rf.role = Follower
	rf.heartbeatTimer.Stop()
	rf.electionTimer.Reset(randomTimeout())
	rf.votedFor = -1
}

func (rf *Raft) convertToLeader() {
	rf.role = Leader
	rf.electionTimer.Stop()
}

func (rf *Raft) convertToCandidate() {
	rf.role = Candidate
	rf.heartbeatTimer.Stop()
}

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.convertToFollower()
	}

	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
	}

	reply.Term = rf.currentTerm
	rf.electionTimer.Reset(randomTimeout())
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.LeaderId != rf.leaderId {
		rf.leaderId = args.LeaderId
	}

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.convertToFollower()
	}

	rf.electionTimer.Reset(randomTimeout())
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) routine() {
	for {
		select {
		case <-rf.electionTimer.C:
			rf.mu.Lock()
			if rf.role == Follower {
				rf.convertToCandidate()
			}
			if rf.role == Candidate {
				rf.startElection()
			}
			rf.mu.Unlock()
		case <-rf.heartbeatTimer.C:
			rf.mu.Lock()
			if rf.role == Leader {
				rf.broadcastHeartbeat()
				rf.heartbeatTimer.Reset(heartbeatInterval)
			}
			rf.mu.Unlock()
		}
	}
}

func (rf *Raft) startElection() {
	rf.currentTerm++
	rf.votedFor = rf.me
	rf.votesReceived = 1
	rf.electionTimer.Reset(randomTimeout())
	go rf.broadcastRequestVote()
}

func (rf *Raft) broadcastRequestVote() {
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}

		rf.mu.Lock()
		requestVoteArgs := &RequestVoteArgs{
			Term:        rf.currentTerm,
			CandidateId: rf.me,
		}
		rf.mu.Unlock()
		requestVoteReply := &RequestVoteReply{}

		go func(i int) {
			ok := rf.sendRequestVote(i, requestVoteArgs, requestVoteReply)
			if !ok {
				return
			}

			rf.mu.Lock()
			defer rf.mu.Unlock()

			if requestVoteReply.VoteGranted {
				rf.votesReceived++
				if rf.votesReceived > len(rf.peers)/2 {
					rf.convertToLeader()
					rf.broadcastHeartbeat()
					rf.heartbeatTimer.Reset(heartbeatInterval)
				}

			} else if requestVoteReply.Term > rf.currentTerm {
				rf.convertToFollower()
			}
		}(i)
	}
}

func (rf *Raft) broadcastHeartbeat() {
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}

		go func(i int) {
			rf.mu.Lock()
			appendEntriesArgs := &AppendEntriesArgs{
				Term:     rf.currentTerm,
				LeaderId: rf.me,
			}
			rf.mu.Unlock()
			appendEntriesReply := &AppendEntriesReply{}

			ok := rf.sendAppendEntries(i, appendEntriesArgs, appendEntriesReply)

			rf.mu.Lock()
			defer rf.mu.Unlock()

			if ok && !appendEntriesReply.Success && appendEntriesReply.Term > rf.currentTerm {
				rf.currentTerm = appendEntriesReply.Term
				rf.convertToFollower()
			}
		}(i)
	}
}

func (rf *Raft) listenHeartBeat() {
	for {
		if rf.role != Follower {
			break
		}

	}
}
