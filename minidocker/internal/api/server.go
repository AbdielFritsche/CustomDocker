package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"minidocker/internal/container"
)

const DefaultSocketPath = "/var/run/minidocker.sock"

type Server struct {
	socketPath string
	mgr        *container.Manager
	listener   net.Listener
	startedAt  time.Time
}

type StartRequest struct {
	Command []string `json:"command,omitempty"`
}

func NewServer(socketPath string, mgr *container.Manager) *Server {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	return &Server{
		socketPath: socketPath,
		mgr:        mgr,
		startedAt:  time.Now(),
	}
}

func (s *Server) Start() error {
	_ = syscall.Unlink(s.socketPath)

	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	s.listener = l
	_ = os.Chmod(s.socketPath, 0660)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/containers/create", s.handleCreate)
	mux.HandleFunc("/containers/start", s.handleStart)
	mux.HandleFunc("/containers/stop", s.handleStop)
	mux.HandleFunc("/containers/list", s.handleList)

	server := &http.Server{Handler: mux}
	return server.Serve(l)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"daemon_pid": os.Getpid(),
		"uptime_sec": int64(time.Since(s.startedAt).Seconds()),
		"status":     "healthy",
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Image    string   `json:"image"`
		Name     string   `json:"name"`
		Command  []string `json:"command"`
		MemoryMB int64    `json:"memory_mb"`
		PidsMax  int64    `json:"pids_max"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	opts := []container.Option{
		container.WithName(req.Name),
		container.WithMemoryLimit(req.MemoryMB * 1024 * 1024),
		container.WithPidsMax(req.PidsMax),
	}

	c, err := s.mgr.CreateContainer(req.Image, req.Command, opts...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(c)
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing container id", http.StatusBadRequest)
		return
	}

	var req StartRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	go func() {
		_ = s.mgr.StartContainer(id, req.Command)
	}()

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"starting"}`))
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := s.mgr.StopContainer(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	containers, err := s.mgr.ListContainers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var respList []ContainerResponse
	for _, c := range containers {
		portsStr := "-"
		if c.Config.PortMapping != nil && c.Config.PortMapping.HostPort > 0 {
			portsStr = fmt.Sprintf("0.0.0.0:%d->%d/tcp", c.Config.PortMapping.HostPort, c.Config.PortMapping.ContainerPort)
		}

		ipStr := c.Config.IP
		if ipStr == "" {
			ipStr = c.Config.StaticIP
		}
		if ipStr == "" {
			ipStr = "-"
		}

		respList = append(respList, ContainerResponse{
			ID:        c.Config.ID,
			Name:      c.Config.Name,
			State:     string(c.State),
			IP:        ipStr,
			PID:       c.PID,
			CreatedAt: c.Config.CreatedAt,
			Ports:     portsStr,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(respList)
}
