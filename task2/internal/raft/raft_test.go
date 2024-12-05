package raft_test

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	httpserver "raftdb/internal/http-server"
	"raftdb/internal/proto/pb"
	"raftdb/internal/raft"

	"github.com/lmittmann/tint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

const (
	BeginTestRaftPort = 5050
	BeginTestHttpPort = 8080
	ClusterSize       = 6
)

type TestRaftServer struct {
	raftServer *raft.RaftServer
	grpcServer *grpc.Server
	raftPort   string
	httpServer *httpserver.Server
	httpPort   string
}

func NewTestServer(id int64, peers []string, raftPort, httpPort int64) *TestRaftServer {
	raftServer := raft.NewRaftServer(id, peers)
	httpServer := httpserver.NewServer(raftServer)

	return &TestRaftServer{
		raftServer: raftServer,
		raftPort:   fmt.Sprintf(":%d", raftPort),
		httpServer: httpServer,
		httpPort:   fmt.Sprintf(":%d", httpPort),
	}
}

func StartTestServer(t *TestRaftServer) {
	grpcServer := grpc.NewServer()
	pb.RegisterRaftServer(grpcServer, t.raftServer)
	t.grpcServer = grpcServer

	t.raftServer.StartTimeouts()

	lis, err := net.Listen("tcp", t.raftPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		slog.Info("Raft-server starting", "port", t.raftPort)
		if err := t.grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
}

func StopTestServer(t *TestRaftServer) {
	t.grpcServer.GracefulStop()
	t.raftServer.ResetTimeouts()
}

func NewTestCluster(count int) []*TestRaftServer {
	result := make([]*TestRaftServer, 0, count)

	peers := make([]string, count)
	for id := 0; id < count; id++ {
		peers[id] = fmt.Sprintf("localhost:%d", BeginTestRaftPort+id)
	}

	for id := int64(0); id < int64(count); id++ {
		filteredPeers := Filter(peers, func(s string) bool {
			return s != fmt.Sprintf("localhost:%d", BeginTestRaftPort+id)
		})
		newServer := NewTestServer(id, filteredPeers, BeginTestRaftPort+id, BeginTestHttpPort+id)
		StartTestServer(newServer)
		result = append(result, newServer)
	}

	return result
}

func Filter(ss []string, test func(string) bool) (ret []string) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

func setupLogger() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelInfo,
		}),
	))
}

// TestLeaderElection performs a test to ensure that leader election in the cluster
// functions correctly. It verifies that initially, the leader is unknown, and after
// a timeout, the first server becomes the leader. It then stops the leader server
// and checks if a new leader is elected, ensuring the leader ID is within the valid
// range of server IDs in the cluster.
func TestLeaderElection(t *testing.T) {
	setupLogger()
	cluster := NewTestCluster(ClusterSize)
	defer func() {
		for _, server := range cluster {
			StopTestServer(server)
		}
	}()

	// Initial leader should be unknown (-1)
	assert.Equal(t, int64(-1), cluster[0].raftServer.GetLeaderID(), "Initial leader should be -1 (unknown)")

	time.Sleep(10 * time.Second)
	assert.Equal(t, int64(0), cluster[0].raftServer.GetLeaderID(), "Server 0 should be the leader after timeout")

	StopTestServer(cluster[0])
	time.Sleep(15 * time.Second)

	leaderID := cluster[1].raftServer.GetLeaderID()
	assert.Greater(t, leaderID, int64(0), "Leader ID should be greater than 0 after leader stops")
	assert.Less(t, leaderID, int64(ClusterSize), "Leader ID should be less than cluster size")
}

// TestLogReplication tests that log entries are correctly replicated to other servers
// in the cluster. It waits for a leader to be elected, then creates a log entry
// on the leader and verifies that the log entry is replicated to all other servers.
func TestLogReplication(t *testing.T) {
	setupLogger()
	cluster := NewTestCluster(ClusterSize)
	defer func() {
		for _, server := range cluster {
			StopTestServer(server)
		}
	}()

	time.Sleep(10 * time.Second)

	value := "2"
	success, err := cluster[0].raftServer.ReplicateLogEntry("CREATE", "1", &value, nil)
	assert.NoError(t, err, "Should not return error when replicating log entry")
	assert.True(t, success, "Log replication should be successful")

	time.Sleep(5 * time.Second)

	for id := 1; id < ClusterSize; id++ {
		assert.Equal(t, 2, cluster[id].raftServer.LogLength(), "Log length should be 2 ['init', 'create'] on server %d", id)
	}
}

// TestLogSync ensures that when a server rejoins the cluster, its log is
// brought up-to-date with the rest of the cluster. It initially stops one
// server, then replicates entries on the leader. After restarting the stopped
// server, it verifies that the server synchronizes its log with the leader.
func TestLogSync(t *testing.T) {
	setupLogger()
	cluster := NewTestCluster(ClusterSize)
	defer func() {
		for _, server := range cluster {
			StopTestServer(server)
		}
	}()

	time.Sleep(10 * time.Second)
	StopTestServer(cluster[1])

	value := "2"
	for i := 1; i <= 3; i++ {
		success, err := cluster[0].raftServer.ReplicateLogEntry("CREATE", fmt.Sprintf("%d", i), &value, nil)
		assert.NoError(t, err, "Should not return error when replicating log entry %d", i)
		assert.True(t, success, "Log replication should be successful for entry %d", i)
	}

	time.Sleep(5 * time.Second)

	for id := 2; id < ClusterSize; id++ {
		assert.Equal(t, 4, cluster[id].raftServer.LogLength(), "Log length should be 4 on server %d after replication", id)
	}

	StartTestServer(cluster[1])
	time.Sleep(5 * time.Second)
	assert.Equal(t, 4, cluster[1].raftServer.LogLength(), "Log length should be 4 on restarted server 1")
}
