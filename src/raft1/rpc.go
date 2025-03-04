package raft

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term        int
	CandidateId int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int
	VoteGranted bool
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.currentTerm {
		switch rf.state {
		case Candidate:
			DPrintf("Candidate->Follower[%d]-term[%d->%d]: Discover a peer with higher term, becomes follower.", rf.me, rf.currentTerm, args.Term)
		case Leader:
			DPrintf("Leader->Follower[%d]-term[%d->%d]: Discover a peer with higher term, becomes follower.", rf.me, rf.currentTerm, args.Term)
		}
		rf.discoverHigherTerm(args.Term)
	}

	reply.Term = rf.currentTerm
	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		rf.updateRecieveHeartbeatTimestamp()
		rf.votedFor = args.CandidateId
		DPrintf("Follower[%d]-term[%d] voted for candidate[%d]-term[%d].", rf.me, rf.currentTerm, args.CandidateId, args.Term)
		reply.VoteGranted = true
	} else {
		DPrintf("(Leader, Candidate, Follower)[%d]-term[%d] did not vote for candidate[%d]-term[%d].", rf.me, rf.currentTerm, args.CandidateId, args.Term)
		reply.VoteGranted = false
	}
}

type AppendEntriesArgs struct {
	Term     int
	LeaderId int
	Entries  []LogEntry
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	if args.Term > rf.currentTerm {
		rf.discoverHigherTerm(args.Term)
	}

	if len(args.Entries) == 0 {
		// This is a heartbeat message.
		DPrintf("Follower[%d]-term[%d] received heartbeat from leader[%d]-term[%d].", rf.me, rf.currentTerm, args.LeaderId, args.Term)
		rf.updateRecieveHeartbeatTimestamp()
		reply.Term = rf.currentTerm
		reply.Success = true
		return
	}
}
