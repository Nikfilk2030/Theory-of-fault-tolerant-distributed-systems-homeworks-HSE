package raft

import (
	"context"
	"log"
	"net"
	"raftdb/internal/db"
	"raftdb/internal/proto/pb"
	"sync"
	"time"

	"log/slog"

	"google.golang.org/grpc"
)

type LogEntry struct {
	Term     int64
	Command  string
	Key      string
	Value    *string
	OldValue *string
}

type RaftServer struct {
	pb.UnimplementedRaftServer

	id           int64
	currentTerm  int64
	lastVotedFor int64
	log          []LogEntry

	state       int // leader, follower, candidate
	leaderID    int64
	commitIndex int64
	nextIndex   map[string]int64

	electionTimeout time.Duration
	electionTimer   *time.Timer

	heartbeatTimeout time.Duration
	heartbeatTimer   *time.Timer

	peers []string
	mu    sync.Mutex
}

func NewRaftServer(id int64, peers []string) *RaftServer {
	server := &RaftServer{
		id:           id,
		currentTerm:  0,
		lastVotedFor: -1,
		log: []LogEntry{
			{
				Term:    0,
				Command: "init",
			},
		},

		state:       FOLLOWER,
		leaderID:    -1,
		commitIndex: 0,
		nextIndex:   make(map[string]int64),

		electionTimeout:  time.Second * time.Duration(8+id*5),
		heartbeatTimeout: time.Second * 5,

		peers: peers,
	}

	server.electionTimer = server.tick(nil, server.electionTimeout, server.beginElection)

	return server
}

func (s *RaftServer) tick(timer *time.Timer, timeout time.Duration, callback func()) *time.Timer {
	if timer != nil {
		timer.Stop()
	}
	return time.AfterFunc(timeout, callback)
}

func (s *RaftServer) RequestVote(ctx context.Context, req *pb.VoteRequest) (*pb.VoteResponse, error) {
	slog.Info("RequestVote received", "node", s.id, "candidate", req.CandidateID, "request_term", req.Term, "current_term", s.currentTerm)

	if req.Term < s.currentTerm {
		return &pb.VoteResponse{Term: s.currentTerm, VoteGranted: false}, nil
	}

	if s.canGrantVote(req) {
		s.currentTerm = req.Term
		s.lastVotedFor = req.CandidateID
		return &pb.VoteResponse{Term: s.currentTerm, VoteGranted: true}, nil
	}

	return &pb.VoteResponse{Term: s.currentTerm, VoteGranted: false}, nil
}

func (s *RaftServer) canGrantVote(req *pb.VoteRequest) bool {
	return s.lastVotedFor == -1 || req.Term > s.currentTerm || s.lastVotedFor == int64(req.CandidateID)
}

func (s *RaftServer) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	slog.Info("AppendEntries received", "node", s.id, "leader", req.LeaderID)

	if req.Term < s.currentTerm {
		return &pb.AppendEntriesResponse{Term: s.currentTerm, Success: false}, nil
	}

	s.updateTermAndState(req)

	if !s.isLogUpToDate(req) {
		return &pb.AppendEntriesResponse{Term: s.currentTerm, Success: false}, nil
	}

	s.appendEntries(req)

	if req.LeaderCommit > s.commitIndex {
		s.applyEntries(req.LeaderCommit)
	}

	return &pb.AppendEntriesResponse{Term: s.currentTerm, Success: true}, nil
}

func (s *RaftServer) updateTermAndState(req *pb.AppendEntriesRequest) {
	s.currentTerm = req.Term
	s.lastVotedFor = -1
	s.state = FOLLOWER
	s.leaderID = req.LeaderID
	s.electionTimer = s.tick(s.electionTimer, s.electionTimeout, s.beginElection)
}

func (s *RaftServer) isLogUpToDate(req *pb.AppendEntriesRequest) bool {
	return req.PrevLogIndex < 0 || (req.PrevLogIndex < int64(len(s.log)) && s.log[req.PrevLogIndex].Term == req.PrevLogTerm)
}

func (s *RaftServer) appendEntries(req *pb.AppendEntriesRequest) {
	for i, entry := range req.Entries {
		logIndex := req.PrevLogIndex + int64(i) + 1
		if logIndex < int64(len(s.log)) && s.log[logIndex].Term != req.Term {
			s.log = s.log[:logIndex] // remove conflict
		}
		s.log = append(s.log, LogEntry{
			Term:     entry.Term,
			Command:  entry.Command,
			Key:      entry.Key,
			Value:    entry.Value,
			OldValue: entry.OldValue,
		})
	}
}

func (s *RaftServer) applyEntries(leaderCommit int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := s.commitIndex; i <= leaderCommit; i++ {
		entry := s.log[i]
		if entry.Command != "init" {
			slog.Info("applying entry", "node", s.id, "entry", entry)
			db.ProcessWrite(entry.Command, entry.Key, entry.Value, entry.OldValue)
		}
	}
	s.commitIndex = leaderCommit
}

func (s *RaftServer) StartRaftServer(port string) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterRaftServer(grpcServer, s)

	slog.Info("Raft server starts to listen", "node_id", s.id, "port", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *RaftServer) GetLeaderID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.leaderID
}

func (s *RaftServer) ResetTimeouts() {
	if s.electionTimer != nil {
		s.electionTimer.Stop()
	}
	if s.heartbeatTimer != nil {
		s.heartbeatTimer.Stop()
	}
}

func (s *RaftServer) StartTimeouts() {
	s.electionTimer = s.tick(s.electionTimer, s.electionTimeout, s.beginElection)
	s.heartbeatTimer = s.tick(s.heartbeatTimer, s.heartbeatTimeout, s.sendHeartbeats)
}

func (s *RaftServer) LogLength() int {
	return len(s.log)
}
