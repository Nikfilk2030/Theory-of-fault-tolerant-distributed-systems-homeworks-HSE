package raft

import (
	"fmt"
	"log/slog"
	"raftdb/internal/db"
	"raftdb/internal/proto/pb"
)

func (s *RaftServer) ReplicateLogEntry(command, key string, value, oldValue *string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != LEADER {
		slog.Error("Attempt to replicate log entry on a replica", "leader", s.leaderID, "node", s.id)
		return false, fmt.Errorf("cannot handle write on a replica. LeaderID: %v", s.leaderID)
	}

	newEntry, prevLogIndex, prevLogTerm := s.prepareLogEntry(command, key, value, oldValue)
	s.log = append(s.log, newEntry)
	slog.Info("Log entry appended by leader", "leader", s.id, "entry", newEntry)

	ackCount := s.replicateToPeers(newEntry, prevLogIndex, prevLogTerm)

	if ackCount > len(s.peers)/2 {
		slog.Info("Successfully replicated entry, applying...", "leader", s.id, "entry", newEntry)
		success, err := db.ProcessWrite(newEntry.Command, newEntry.Key, newEntry.Value, newEntry.OldValue)
		s.commitIndex = int64(len(s.log) - 1)
		return success, err
	}
	return false, fmt.Errorf("cannot replicate entry %+v", newEntry)
}

func (s *RaftServer) prepareLogEntry(command, key string, value, oldValue *string) (LogEntry, int64, int64) {
	var prevLogIndex, prevLogTerm int64
	if len(s.log) > 0 {
		prevLogIndex = int64(len(s.log) - 1)
		prevLogTerm = s.log[prevLogIndex].Term
	}
	return LogEntry{
		Term:     s.currentTerm,
		Command:  command,
		Key:      key,
		Value:    value,
		OldValue: oldValue,
	}, prevLogIndex, prevLogTerm
}

func (s *RaftServer) replicateToPeers(entry LogEntry, prevLogIndex, prevLogTerm int64) int {
	ackCh := make(chan bool, len(s.peers))
	for _, peerID := range s.peers {
		go func(peerID string) {
			req := &pb.AppendEntriesRequest{
				Term:         s.currentTerm,
				LeaderID:     s.id,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries: []*pb.LogEntry{
					{
						Term:     entry.Term,
						Command:  entry.Command,
						Key:      entry.Key,
						Value:    entry.Value,
						OldValue: entry.OldValue,
					},
				},
				LeaderCommit: s.commitIndex,
			}

			resp, err := sendAppendEntries(peerID, req)
			if err == nil && resp.Success {
				ackCh <- true
				s.mu.Lock()
				s.nextIndex[peerID]++
				s.mu.Unlock()
			} else {
				if err != nil {
					slog.Error("Error appending entry to peer", "leader", s.id, "peer", peerID, "error", err)
				}
				ackCh <- false
			}
		}(peerID)
	}

	ackCount := 1 // само ак
	for i := 0; i < len(s.peers); i++ {
		if <-ackCh {
			ackCount++
		}
	}
	return ackCount
}
