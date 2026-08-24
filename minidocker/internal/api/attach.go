package api

import (
	"encoding/json"
	"net/http"

	"github.com/hashicorp/yamux"
)

func (s *Server) handleAttach(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "servidor no soporta hijacking de conexión")
		return
	}

	w.Header().Set("Connection", "Upgrade")
	w.Header().Set("Upgrade", "yamux")
	w.WriteHeader(http.StatusSwitchingProtocols)

	rawConn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer rawConn.Close()

	session, err := yamux.Server(rawConn, yamux.DefaultConfig())
	if err != nil {
		return
	}
	defer session.Close()

	stdoutStream, err := session.OpenStream()
	if err != nil {
		return
	}
	defer stdoutStream.Close()

	stderrStream, err := session.OpenStream()
	if err != nil {
		return
	}
	defer stderrStream.Close()

	controlStream, err := session.OpenStream()
	if err != nil {
		return
	}
	defer controlStream.Close()

	stdinStream, err := session.AcceptStream()
	if err != nil {
		return
	}
	defer stdinStream.Close()

	go func() {
		dec := json.NewDecoder(controlStream)
		for {
			var msg ControlMessage
			if err := dec.Decode(&msg); err != nil {
				return
			}
		}
	}()

	exitCode := 0
	if err := s.mgr.StartAttached(id, stdinStream, stdoutStream, stderrStream); err != nil {
		exitCode = 1
	}

	_ = json.NewEncoder(controlStream).Encode(ControlMessage{
		Type: "exit",
		Code: exitCode,
	})
}
