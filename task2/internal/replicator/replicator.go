package replicator

import "raftdb/internal/raft"

type Replicator struct {
	raftServer *raft.RaftServer
}

func NewReplicator(raftServer *raft.RaftServer) *Replicator {
	return &Replicator{
		raftServer: raftServer,
	}
}

func (r *Replicator) ApplyAndReplicate(command string, key string, value, oldValue *string) (bool, error) {
	return r.raftServer.ReplicateLogEntry(command, key, value, oldValue)
}
