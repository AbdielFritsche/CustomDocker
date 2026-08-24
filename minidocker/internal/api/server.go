package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"minidocker/internal/container"
	"minidocker/internal/isolation"
)

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
	s.httpSrv = &http.Server{Handler: s.routes()}
	return s
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	prefix := "/" + APIVersion

	mux.HandleFunc("POST "+prefix+"/containers", s.handleCreate)
	mux.HandleFunc("GET "+prefix+"/containers", s.handleList)
	mux.HandleFunc("GET "+prefix+"/containers/{id}", s.handleInspect)
	mux.HandleFunc("POST "+prefix+"/containers/{id}/start", s.handleStart)
	mux.HandleFunc("POST "+prefix+"/containers/{id}/attach", s.handleAttach)
	mux.HandleFunc("POST "+prefix+"/containers/{id}/stop", s.handleStop)
	mux.HandleFunc("DELETE "+prefix+"/containers/{id}", s.handleDelete)
	mux.HandleFunc("GET "+prefix+"/containers/{id}/logs", s.handleLogs)
	mux.HandleFunc("GET "+prefix+"/containers/{id}/stats", s.handleStats)
	mux.HandleFunc("GET "+prefix+"/ping", s.handlePing)

	return logMiddleware(mux)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[minidockerd] %s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
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

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateContainerRequest
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

	writeJSON(w, http.StatusCreated, CreateContainerResponse{ID: c.Config.ID, Name: c.Config.Name})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.mgr.DeleteContainer(id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func (s *Server) handleInspect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	mgr := s.managerFor(r.URL.Query().Get("data_root"))
	c, err := mgr.GetContainer(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	writeJSON(w, http.StatusOK, toDTO(c))
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	mgr := s.managerFor(r.URL.Query().Get("data_root"))
	dirs, err := mgr.ListContainerDirs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := PsResponse{Containers: []ContainerView{}}
	for _, dir := range dirs {
		id := filepath.Base(dir)
		c, err := mgr.GetContainer(id)
		if err != nil {
			continue
		}
		resp.Containers = append(resp.Containers, toDTO(c))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	follow := r.URL.Query().Get("follow") == "true"
	mgr := s.managerFor(r.URL.Query().Get("data_root"))

	c, err := mgr.GetContainer(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	logPath := filepath.Join(c.Config.BasePath, c.Config.ID, "container.log")
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, fmt.Errorf("no hay registros para [%s]", id))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	flusher, canFlush := w.(http.Flusher)

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			_, _ = io.WriteString(w, line)
			if canFlush {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			if !follow {
				return
			}
			curr, gerr := mgr.GetContainer(id)
			if gerr != nil || curr.State != container.StateRunning {
				return
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err != nil {
			return
		}
	}
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePs(w http.ResponseWriter, r *http.Request) {
	dirs, err := s.mgr.ListContainerDirs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := PsResponse{Containers: []ContainerView{}}
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

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req StartContainerRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // el body es opcional

	go func() {
		if err := s.mgr.StartContainer(id, req.Command); err != nil {
			log.Printf("[minidockerd] error arrancando [%s]: %v", id, err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "starting", "id": id})
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

	writeJSON(w, http.StatusOK, StatsResponse{
		MemUsageBytes: stats.MemoryUsageBytes,
		MemLimitBytes: stats.MemoryLimitBytes,
		CPUUsageUsec:  stats.CPUUsageUsec,
		PidsCurrent:   stats.PidsCurrent,
		PidsMax:       stats.PidsMax,
	})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.mgr.StopContainer(id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "id": id})
}

func (s *Server) managerFor(dataRoot string) *container.Manager {
	if dataRoot == "" {
		return s.mgr
	}
	return container.NewManager(dataRoot)
}

func toDTO(c *container.Container) ContainerView {
	dto := ContainerView{
		ID:        c.Config.ID,
		Name:      c.Config.Name,
		Image:     c.Config.Image,
		Command:   c.Config.Command,
		State:     c.State,
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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, ErrorResponse{Error: err.Error()})
}

func (s *Server) writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
