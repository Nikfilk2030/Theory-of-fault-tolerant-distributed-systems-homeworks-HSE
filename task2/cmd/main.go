package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	httpserver "raftdb/internal/http-server"
	"raftdb/internal/raft"

	"github.com/lmittmann/tint"
)

type Config struct {
	ID       int64
	RaftPort int
	HTTPPort int
	Peers    []string
}

func main() {
	setupLogging()
	config := parseFlags()

	if err := ensurePortAvailable(config.HTTPPort); err != nil {
		slog.Error("Port check failed", "error", err)
		os.Exit(1)
	}

	raftServer := raft.NewRaftServer(config.ID, config.Peers)
	httpServer := httpserver.NewServer(raftServer)

	registerHTTPHandlers(httpServer)

	go startRaftServer(raftServer, config.RaftPort)

	if err := startHTTPServer(config.HTTPPort); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func setupLogging() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelInfo,
		}),
	))
}

func parseFlags() Config {
	var (
		id       int64
		raftPort int
		httpPort int
	)
	flag.Int64Var(&id, "id", 0, "ID for replicaset")
	flag.IntVar(&raftPort, "raft_port", 0, "Raft port for replica")
	flag.IntVar(&httpPort, "http_port", 0, "HTTP port for user requests")
	flag.Parse()
	peers := flag.Args()

	slog.Info("Received peer replicas", "peers", peers)

	return Config{
		ID:       id,
		RaftPort: raftPort,
		HTTPPort: httpPort,
		Peers:    peers,
	}
}

func registerHTTPHandlers(server *httpserver.Server) {
	http.HandleFunc("/get/{key}", server.GetHandler)
	http.HandleFunc("/create", server.CreateHandler)
	http.HandleFunc("/delete", server.DeleteHandler)
	http.HandleFunc("/update", server.UpdateHandler)
	http.HandleFunc("/cas", server.CompareAndSwapHandler)
}

func startRaftServer(server *raft.RaftServer, port int) {
	address := fmt.Sprintf(":%d", port)
	slog.Info("Raft server starting", "port", port)
	server.StartRaftServer(address)
}

func startHTTPServer(port int) error {
	slog.Info("HTTP server starting", "port", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func ensurePortAvailable(port int) error {
	address := fmt.Sprintf(":%d", port)

	listener, err := net.Listen("tcp", address)
	if err == nil {
		_ = listener.Close()
		return nil
	}

	pid, err := getProcessUsingPort(port)
	if err != nil {
		return fmt.Errorf("port %d is already in use and the process using it could not be identified: %v", port, err)
	}

	slog.Warn("Port is in use", "port", port, "pid", pid)

	if confirmKillProcess(pid, port) {
		if err := killProcess(pid); err != nil {
			return fmt.Errorf("failed to kill process %d: %v", pid, err)
		}
		slog.Info("Process terminated", "pid", pid)
	} else {
		return fmt.Errorf("port %d is in use by process %d", port, pid)
	}

	return nil
}

func getProcessUsingPort(port int) (int, error) {
	cmd := exec.Command("lsof", "-t", fmt.Sprintf("-i:%d", port))
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, fmt.Errorf("no process found using port %d", port)
	}

	var pid int
	if _, err := fmt.Sscanf(pidStr, "%d", &pid); err != nil {
		return 0, err
	}
	return pid, nil
}

func confirmKillProcess(pid int, port int) bool {
	fmt.Printf("Port %d is already in use by process %d.\n", port, pid)
	fmt.Print("Do you want to terminate this process? (y/N): ")
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}

func killProcess(pid int) error {
	cmd := exec.Command("kill", fmt.Sprintf("%d", pid))
	return cmd.Run()
}
