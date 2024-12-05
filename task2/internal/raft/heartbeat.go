package raft

import (
	"log/slog"
	"raftdb/internal/proto/pb"
)

func (s *RaftServer) sendHeartbeats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != LEADER {
		return
	}

	for _, peer := range s.peers {
		go s.sendHeartbeatToPeer(peer)
	}

	s.heartbeatTimer = s.Tick(s.heartbeatTimer, s.heartbeatTimeout, s.sendHeartbeats)
}

func (s *RaftServer) sendHeartbeatToPeer(peer string) {
	for {
		s.mu.Lock()
		nextIndex := s.ensureNextIndex(peer)
		entriesProto := s.prepareLogEntriesProto(nextIndex)

		prevLogTerm := int64(0)
		if nextIndex > 0 {
			prevLogTerm = s.log[nextIndex-1].Term
		}

		req := &pb.AppendEntriesRequest{
			Term:         s.currentTerm,
			LeaderID:     s.id,
			LeaderCommit: s.commitIndex,
			PrevLogIndex: nextIndex - 1,
			PrevLogTerm:  prevLogTerm,
			Entries:      entriesProto,
		}
		s.mu.Unlock()

		resp, err := sendAppendEntries(peer, req)
		if err != nil {
			slog.Error("Heartbeat error from leader", "error", err, "leader", s.id, "node", peer)
			return
		}

		if nextIndex == 0 {
			break
		}

		s.processAppendEntriesResponse(peer, resp, nextIndex)
	}
}

func (s *RaftServer) ensureNextIndex(peer string) int64 {
	nextIndex, exists := s.nextIndex[peer]
	if !exists {
		s.nextIndex[peer] = 0
		nextIndex = 0
	}
	return nextIndex
}

func (s *RaftServer) prepareLogEntriesProto(nextIndex int64) []*pb.LogEntry {
	entries := s.log[nextIndex:]
	entriesProto := make([]*pb.LogEntry, len(entries))
	for i, entry := range entries {
		entriesProto[i] = &pb.LogEntry{
			Term:     entry.Term,
			Command:  entry.Command,
			Key:      entry.Key,
			Value:    entry.Value,
			OldValue: entry.OldValue,
		}
	}
	return entriesProto
}

func (s *RaftServer) processAppendEntriesResponse(peer string, resp *pb.AppendEntriesResponse, nextIndex int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if resp.Success {
		s.nextIndex[peer] = nextIndex + int64(len(s.log[nextIndex:]))
	} else {
		s.nextIndex[peer] = nextIndex - 1
		slog.Info("Replica not in sync! Decrementing next index and retrying", "leader", s.id, "node", peer)
	}
}
