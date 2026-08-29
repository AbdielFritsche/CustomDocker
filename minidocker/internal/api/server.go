package api

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"minidocker/internal/api/handlers"
	"minidocker/internal/container"
)

type Server struct {
	deps       *handlers.Deps
	socketPath string
	httpSrv    *http.Server
}

func NewServer(mgr *container.Manager, socketPath string) *Server {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	s := &Server{
		deps:       &handlers.Deps{Mgr: mgr},
		socketPath: socketPath,
	}
	s.httpSrv = &http.Server{Handler: s.routes()}
	return s
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	prefix := "/" + APIVersion

	//POST
	mux.HandleFunc("POST "+prefix+"/containers", s.deps.HandleCreate)
	mux.HandleFunc("POST "+prefix+"/containers/{id}/start", s.deps.HandleStart)
	mux.HandleFunc("POST "+prefix+"/containers/{id}/attach", s.deps.HandleAttach)
	mux.HandleFunc("POST "+prefix+"/containers/{id}/exec", s.deps.HandleExec)
	mux.HandleFunc("POST "+prefix+"/containers/{id}/stop", s.deps.HandleStop)
	mux.HandleFunc("POST "+prefix+"/compose/up", s.deps.HandleComposeUp)
	mux.HandleFunc("POST "+prefix+"/compose/down", s.deps.HandleComposeDown)
	//GET
	mux.HandleFunc("GET "+prefix+"/containers/{id}/logs", s.deps.HandleLogs)
	mux.HandleFunc("GET "+prefix+"/containers/{id}/stats", s.deps.HandleStats)
	mux.HandleFunc("GET "+prefix+"/ping", s.deps.HandlePing)
	mux.HandleFunc("GET "+prefix+"/containers", s.deps.HandleList)
	mux.HandleFunc("GET "+prefix+"/containers/{id}", s.deps.HandleInspect)
	//DELETE
	mux.HandleFunc("DELETE "+prefix+"/containers/{id}", s.deps.HandleDelete)
	mux.HandleFunc("DELETE "+prefix+"/networks/{name}", s.deps.HandleNetworkDelete)

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
