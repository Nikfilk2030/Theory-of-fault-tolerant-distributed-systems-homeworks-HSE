package raft

import (
	"log/slog"

	"raftdb/internal/proto/pb"
)

func (s *RaftServer) beginElection() {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("Begin election...", "node", s.id)

	s.currentTerm++
	s.state = CANDIDATE
	s.lastVotedFor = s.id
	votesReceived := 1

	s.electionTimer = s.tick(s.electionTimer, s.electionTimeout, s.beginElection)

	for _, peer := range s.peers {
		go s.requestVoteFromPeer(peer, &votesReceived)
	}
}

func (s *RaftServer) requestVoteFromPeer(peer string, votesReceived *int) {
	request := &pb.VoteRequest{
		Term:         s.currentTerm,
		CandidateID:  s.id,
		LastLogIndex: int64(len(s.log) - 1),
		LastLogTerm:  int64(s.log[len(s.log)-1].Term),
	}

	response, err := sendRequestVote(peer, request)
	if err != nil {
		slog.Error("Failed to request vote", "peer", peer, "error", err)
		return
	}

	if response.VoteGranted {
		s.mu.Lock()
		defer s.mu.Unlock()
		(*votesReceived)++
		slog.Info("Vote granted", "node", s.id, "from", peer, "term", s.currentTerm)
		if *votesReceived > len(s.peers)/2 && s.state == CANDIDATE {
			s.becomeLeader()
		}
	}
}

func (s *RaftServer) becomeLeader() {
	slog.Info("Becoming leader", "node", s.id)

	s.state = LEADER
	s.leaderID = s.id

	s.heartbeatTimer = s.tick(s.heartbeatTimer, s.heartbeatTimeout, s.sendHeartbeats)

	if s.electionTimer != nil {
		s.electionTimer.Stop()
	}
}
