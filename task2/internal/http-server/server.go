package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"raftdb/internal/db"
	"raftdb/internal/raft"
	"raftdb/internal/replicator"

	"log/slog"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	replicator *replicator.Replicator
}

func NewServer(raftServer *raft.RaftServer) *Server {
	return &Server{
		replicator: replicator.NewReplicator(raftServer),
	}
}

func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		handleError(w, "Key is required", http.StatusBadRequest)
		return
	}

	value, exists := db.Get(key)
	if !exists {
		handleError(w, "Key not found", http.StatusNotFound)
		return
	}

	writeResponse(w, fmt.Sprintf("%v", value))
}

func (s *Server) CreateHandler(w http.ResponseWriter, r *http.Request) {
	req, err := decodeKeyValueRequest(r)
	if err != nil {
		handleError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	success, err := s.replicator.ApplyAndReplicate("CREATE", req.Key, &req.Value, nil)
	if err != nil {
		handleError(w, fmt.Sprintf("Failed to apply command: %v", err), http.StatusInternalServerError)
		return
	}

	writeResponse(w, fmt.Sprintf("%v", success))
}

func (s *Server) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	req, err := decodeKeyRequest(r)
	if err != nil {
		handleError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	success, err := s.replicator.ApplyAndReplicate("DELETE", req.Key, nil, nil)
	if err != nil {
		handleError(w, fmt.Sprintf("Failed to apply command: %v", err), http.StatusInternalServerError)
		return
	}

	writeResponse(w, fmt.Sprintf("%v", success))
}

func (s *Server) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	req, err := decodeKeyValueRequest(r)
	if err != nil {
		handleError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	success, err := s.replicator.ApplyAndReplicate("UPDATE", req.Key, &req.Value, nil)
	if err != nil {
		handleError(w, fmt.Sprintf("Failed to apply command: %v", err), http.StatusInternalServerError)
		return
	}

	writeResponse(w, fmt.Sprintf("%v", success))
}

func (s *Server) CompareAndSwapHandler(w http.ResponseWriter, r *http.Request) {
	req, err := decodeCASRequest(r)
	if err != nil {
		handleError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	success, err := s.replicator.ApplyAndReplicate("CAS", req.Key, &req.NewValue, &req.OldValue)
	if err != nil {
		handleError(w, fmt.Sprintf("Failed to apply command: %v", err), http.StatusInternalServerError)
		return
	}

	writeResponse(w, fmt.Sprintf("%v", success))
}

type TKeyValueRequestType struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func decodeKeyValueRequest(r *http.Request) (TKeyValueRequestType, error) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

type TKeyRequestType struct {
	Key string `json:"key"`
}

func decodeKeyRequest(r *http.Request) (TKeyRequestType, error) {
	var req struct {
		Key string `json:"key"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

type TCASRequestType struct {
	Key      string `json:"key"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

func decodeCASRequest(r *http.Request) (TCASRequestType, error) {
	var req struct {
		Key      string `json:"key"`
		OldValue string `json:"old_value"`
		NewValue string `json:"new_value"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func handleError(w http.ResponseWriter, message string, statusCode int) {
	slog.Error(message)
	http.Error(w, message, statusCode)
}

func writeResponse(w http.ResponseWriter, response string) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}
