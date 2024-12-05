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
		slog.Error("WTF, replicate in replica", "leader", s.leaderID, "node", s.id)
		return false, fmt.Errorf("cannot handle write on replica. LeaderID: %v", s.leaderID)
	}

	var prevLogIndex int64
	var prevLogTerm int64

	prevLogIndex = int64(len(s.log) - 1)
	if prevLogIndex >= 0 {
		prevLogTerm = s.log[prevLogIndex].Term
	}
	entry := LogEntry{
		Term:     s.currentTerm,
		Command:  command,
		Key:      key,
		Value:    value,
		OldValue: oldValue,
	}

	s.log = append(s.log, entry)
	slog.Info("master append entry", "leader", s.id, "entry", entry)

	ackCh := make(chan bool, len(s.peers))
	for _, peer := range s.peers {
		go func(peer string) {
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

			resp, err := sendAppendEntries(peer, req)
			if err == nil && resp.Success {
				ackCh <- true
				s.mu.Lock()
				s.nextIndex[peer]++
				s.mu.Unlock()
			} else {
				if err != nil {
					slog.Error("Append new entry error", "leader", s.id, "error", err)
				}

				ackCh <- false
			}
		}(peer)
	}

	ackCount := 1 // self ack
	for i := 0; i < len(s.peers); i++ {
		if <-ackCh {
			ackCount++
		}
		if ackCount > len(s.peers)/2 {
			break
		}
	}

	if ackCount > len(s.peers)/2 {
		slog.Info("Successfully replicated entry, applying...", "leader", s.id, "entry", entry)
		success, err := db.ProcessWrite(entry.Command, entry.Key, entry.Value, entry.OldValue)
		s.commitIndex = int64(len(s.log) - 1)
		return success, err
	}

	return false, fmt.Errorf("cannot replicate entry %+v", entry)
}
