package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// Log entry.
type LogEntry struct {
	Term    int
	Command interface{}
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	term                   int
	state                  State
	voteFor                int
	votes                  int
	lastRequestedTime      time.Time
	rpcCallTimeout         time.Duration
	timeBetweenElections   time.Duration
	timeUntilStartElection time.Duration
	timeBetweenHeartbeats  time.Duration
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (3A).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	return rf.term, rf.state == Leader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (3B).

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) recordRequestedTime() {
	rf.lastRequestedTime = time.Now()
}

func (rf *Raft) resetTimeUntilStartElection() {
	rf.timeUntilStartElection = time.Duration(200+rand.Int63()%200) * time.Millisecond
}

func (rf *Raft) resetTimeBetweenElections() {
	rf.timeBetweenElections = time.Duration(250+rand.Int63()%500) * time.Millisecond
}

func (rf *Raft) startElection() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != Follower {
		return
	}

	if time.Since(rf.lastRequestedTime) < rf.timeUntilStartElection {
		return
	}

	/*
	 * Start election.
	 */
	// Update current state.
	rf.state = Candidate
	rf.term++
	DPrintf("Candidate %d New term %d: Starts election.", rf.me, rf.term)

	// Vote for itself.
	rf.votes = 1
	rf.voteFor = rf.me

	// Request other's vote.
	args := RequestVoteArgs{
		Term:        rf.term,
		CandidateId: rf.me,
	}

	rf.mu.Unlock()
	var wait sync.WaitGroup
	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		wait.Add(1)
		go func(peerIndex int) {
			defer wait.Done()
			// Check if the state has changed to follower before sending the request.
			rf.mu.Lock()
			if rf.state != Candidate {
				rf.mu.Unlock()
				return
			}
			rf.mu.Unlock()

			// Send request and recieve reply.
			reply := RequestVoteReply{}
			resultCh := make(chan bool, 1)
			go func() {
				ok := rf.peers[peerIndex].Call("Raft.RequestVote", &args, &reply)
				resultCh <- ok
			}()
			select {
			case ok := <-resultCh:
				if !ok {
					return
				}
			case <-time.After(rf.rpcCallTimeout):
				return
			}

			// Analyze reply.
			rf.mu.Lock()
			defer rf.mu.Unlock()

			if rf.state == Leader {
				return
			}

			if reply.Term > rf.term {
				DPrintf("Candidate %d: Recieve more legistimate term from peer %d. Current term: %d, new term: %d", rf.me, peerIndex, rf.term, reply.Term)
				rf.term = reply.Term
				rf.state = Follower
				rf.recordRequestedTime()
				return
			}

			if reply.VoteGranted {
				DPrintf("Candidate %d Term %d: Recieve vote from Follower %d.", rf.me, rf.term, peerIndex)
				rf.votes++
			} else {
				DPrintf("Candidate %d Term %d: Recieve reject vote from peer %d.", rf.me, rf.term, peerIndex)
			}
		}(i)
	}
	wait.Wait()

	// Check if it has won the election.
	rf.mu.Lock()
	if rf.votes > len(rf.peers)/2 {
		DPrintf("(Candidate -> Leader) %d Term %d: Wins election with votes %d/%d.", rf.me, rf.term, rf.votes, len(rf.peers))
		rf.state = Leader
	} else {
		DPrintf("(Candidate -> Follower) %d Term %d: Loses election with votes %d/%d.", rf.me, rf.term, rf.votes, len(rf.peers))

		rf.voteFor = -1
		rf.votes = 0
		rf.state = Follower

		rf.resetTimeUntilStartElection()
	}
}

func (rf *Raft) startElectionRoutine() {
	for !rf.killed() {
		rf.startElection()
		time.Sleep(rf.timeBetweenElections)
		rf.resetTimeBetweenElections()
	}
}

func (rf *Raft) sendHeartbeatsRoutine() {
	for !rf.killed() {
		rf.sendHeartbeats()
		time.Sleep(rf.timeBetweenHeartbeats)
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.term = 0
	rf.state = Follower
	rf.votes = 0
	rf.voteFor = -1
	rf.timeBetweenHeartbeats = 100 * time.Millisecond
	rf.rpcCallTimeout = 500 * time.Millisecond
	rf.recordRequestedTime()
	rf.resetTimeBetweenElections()
	rf.resetTimeUntilStartElection()

	// Your initialization code here (3A, 3B, 3C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.startElectionRoutine()
	go rf.sendHeartbeatsRoutine()

	return rf
}
