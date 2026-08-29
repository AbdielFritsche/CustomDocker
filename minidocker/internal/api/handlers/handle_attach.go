package handlers

import (
	"encoding/json"
	"net/http"

	"minidocker/internal/api/dto"

	"github.com/hashicorp/yamux"
)

func (d *Deps) HandleAttach(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	cmdOverride := r.URL.Query()["cmd"]

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		writeErrorMsg(w, http.StatusInternalServerError, "servidor no soporta hijacking de conexión")
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
			var msg dto.ControlMessage
			if err := dec.Decode(&msg); err != nil {
				return
			}
		}
	}()

	exitCode := 0
	if err := d.Mgr.StartAttached(id, cmdOverride, stdinStream, stdoutStream, stderrStream); err != nil {
		exitCode = 1
	}

	_ = json.NewEncoder(controlStream).Encode(dto.ControlMessage{
		Type: "exit",
		Code: exitCode,
	})
}
