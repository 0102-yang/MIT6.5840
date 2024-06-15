package raft

// RequestVote RPC arguments structure.
type RequestVoteArgs struct {
	Term        int
	CandidateId int
}

// RequestVote RPC reply structure.
type RequestVoteReply struct {
	Term        int
	VoteGranted bool
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

// RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.term || (args.Term == rf.term && rf.voteFor != -1) {
		reply.Term = rf.term
		return
	}

	switch rf.state {
	case Follower:
		rf.term = args.Term
		rf.voteFor = args.CandidateId
		rf.recordRequestedTime()
		reply.VoteGranted = true
	case Candidate:
		rf.term = args.Term
		rf.state = Follower
		rf.voteFor = args.CandidateId
		rf.recordRequestedTime()

		reply.VoteGranted = true

		DPrintf("(Candidate -> Follower) %d Term %d: Recieve more legistimate candidate %d with term %d. Change state back to follower.", rf.me, rf.term, args.CandidateId, args.Term)
	case Leader:
		rf.term = args.Term
		rf.state = Follower
		rf.voteFor = args.CandidateId
		rf.recordRequestedTime()

		reply.VoteGranted = true

		DPrintf("(Leader -> Follower) %d Term %d: Recieve more legistimate candidate %d with term %d. Change state back to follower.", rf.me, rf.term, args.CandidateId, args.Term)
	}

	reply.Term = rf.term
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.term {
		reply.Term = rf.term
		reply.Success = false
		return
	}

	if len(args.Entries) == 0 {
		// This is a heartbeat message.
		reply.Term = rf.term
		reply.Success = true

		rf.term = args.Term
		rf.voteFor = -1
		rf.votes = 0
		rf.recordRequestedTime()

		switch rf.state {
		case Follower:
			DPrintf("Follower %d Term %d: Recieve heartbeat package from leader %d with term %d. Reset current election timeout.", rf.me, rf.term, args.LeaderId, args.Term)
		case Candidate:
			DPrintf("(Candidate -> Follower) %d Term %d: Recieve heartbeat package from leader %d with term %d. Change state back to follower.", rf.me, rf.term, args.LeaderId, args.Term)
			rf.state = Follower
		case Leader:
			DPrintf("(Leader -> Follower) %d Term %d: Recieve heartbeat reply from leader %d with term %d.", rf.me, rf.term, args.LeaderId, args.Term)
			rf.state = Follower
		}
	}
}

func (rf *Raft) sendHeartbeats() {
	rf.mu.Lock()

	if rf.state != Leader {
		rf.mu.Unlock()
		return
	}

	args := AppendEntriesArgs{
		Term:     rf.term,
		LeaderId: rf.me,
		Entries:  []LogEntry{},
	}

	rf.mu.Unlock()

	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		go func(peerIndex int) {
			rf.mu.Lock()
			if rf.state != Leader {
				rf.mu.Unlock()
				return
			}
			rf.mu.Unlock()

			reply := AppendEntriesReply{}

			ok := rf.peers[peerIndex].Call("Raft.AppendEntries", &args, &reply)
			if !ok {
				return
			}

			rf.mu.Lock()
			if reply.Term > rf.term {
				DPrintf("(Leader -> Follower) %d Term %d: Recieved more legistimate peer %d with term %d. Change state back to follower.", rf.me, rf.term, peerIndex, reply.Term)
				rf.term = reply.Term
				rf.state = Follower
				rf.recordRequestedTime()
			}
			rf.mu.Unlock()
		}(i)
	}
}
