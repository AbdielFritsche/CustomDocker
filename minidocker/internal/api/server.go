package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"minidocker/internal/container"
	"minidocker/internal/isolation"
)

const DefaultSocketPath = "/var/run/minidocker.sock"

type Server struct {
	mgr        *container.Manager
	socketPath string
	httpSrv    *http.Server
}

func NewServer(mgr *container.Manager, socketPath string) *Server {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	s := &Server{mgr: mgr, socketPath: socketPath}

	mux := http.NewServeMux()
	s.registerRoutes(mux)
	s.httpSrv = &http.Server{Handler: mux}

	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {

	mux.HandleFunc("POST /containers", s.handleCreate)
	mux.HandleFunc("GET /containers", s.handlePs)
	mux.HandleFunc("POST /containers/{id}/start", s.handleStart)
	mux.HandleFunc("POST /containers/{id}/stop", s.handleStop)
	mux.HandleFunc("DELETE /containers/{id}", s.handleDelete)
	mux.HandleFunc("GET /containers/{id}/stats", s.handleStats)
}

func (s *Server) ListenAndServe() error {
	if _, err := os.Stat(s.socketPath); err == nil {
		if err := os.Remove(s.socketPath); err != nil {
			return fmt.Errorf("no se pudo limpiar socket previo %s: %w", s.socketPath, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0755); err != nil {
		return fmt.Errorf("error creando directorio del socket: %w", err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("error abriendo socket %s: %w", s.socketPath, err)
	}

	if err := os.Chmod(s.socketPath, 0666); err != nil {
		log.Printf("[minidockerd] advertencia: no se pudo ajustar permisos del socket: %v", err)
	}

	log.Printf("[minidockerd] escuchando en %s", s.socketPath)
	return s.httpSrv.Serve(listener)
}

func (s *Server) Close() error {
	_ = os.Remove(s.socketPath)
	return s.httpSrv.Close()
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, ErrorResponse{Error: err.Error()})
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("body inválido: %w", err))
		return
	}

	cmd := req.Command
	if len(cmd) == 0 {
		cmd = []string{"/bin/sh"}
	}

	opts := []container.Option{
		container.WithMemoryLimit(req.MemoryMB * 1024 * 1024),
		container.WithPidsMax(req.PidsMax),
	}
	if req.Name != "" {
		opts = append(opts, container.WithName(req.Name))
	}
	if req.Port != "" {
		opts = append(opts, container.WithPortMapping(req.Port))
	}
	if len(req.Env) > 0 {
		opts = append(opts, container.WithEnv(req.Env))
	}

	c, err := s.mgr.CreateContainer(req.Image, cmd, opts...)
	if err != nil {
		// Nombre duplicado, etc. -> error del cliente, no del servidor.
		writeError(w, http.StatusConflict, err)
		return
	}

	writeJSON(w, http.StatusCreated, CreateResponse{ID: c.Config.ID, Name: c.Config.Name})
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req StartRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // el body es opcional

	go func() {
		if err := s.mgr.StartContainer(id, req.CommandOverride); err != nil {
			log.Printf("[minidockerd] error arrancando [%s]: %v", id, err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "starting", "id": id})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.mgr.StopContainer(id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "id": id})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.mgr.DeleteContainer(id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func (s *Server) handlePs(w http.ResponseWriter, r *http.Request) {
	dirs, err := s.mgr.ListContainerDirs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := PsResponse{Containers: []ContainerDTO{}}
	for _, dir := range dirs {
		id := filepath.Base(dir)
		c, err := s.mgr.GetContainer(id)
		if err != nil {
			continue
		}
		resp.Containers = append(resp.Containers, toDTO(c))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := s.mgr.GetContainer(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if c.State != container.StateRunning {
		writeError(w, http.StatusConflict, fmt.Errorf("el contenedor [%s] no está en ejecución", id))
		return
	}

	cg := isolation.NewCgroupManager(c.Config.ID)
	stats, err := cg.ReadStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, StatsDTO{
		MemUsageBytes: stats.MemoryUsageBytes,
		MemLimitBytes: stats.MemoryLimitBytes,
		CPUUsageUsec:  stats.CPUUsageUsec,
		PidsCurrent:   stats.PidsCurrent,
		PidsMax:       stats.PidsMax,
	})
}

func toDTO(c *container.Container) ContainerDTO {
	dto := ContainerDTO{
		ID:        c.Config.ID,
		Name:      c.Config.Name,
		Image:     c.Config.Image,
		Command:   c.Config.Command,
		State:     string(c.State),
		PID:       c.PID,
		IP:        c.Config.IP,
		CreatedAt: c.Config.CreatedAt,
	}
	if c.Config.PortMapping != nil {
		dto.HostPort = c.Config.PortMapping.HostPort
		dto.ContPort = c.Config.PortMapping.ContainerPort
	}
	return dto
}
