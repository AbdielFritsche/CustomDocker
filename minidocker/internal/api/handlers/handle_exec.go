package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"minidocker/internal/api/dto"
	"minidocker/internal/container"

	"github.com/hashicorp/yamux"
)

func (d *Deps) HandleExec(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	c, err := d.Mgr.GetContainer(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, err)
		return
	}
	if c.State != container.StateRunning || c.PID <= 0 {
		writeErrorMsg(w, http.StatusConflict, fmt.Sprintf("el contenedor [%s] no está en ejecución", id))
		return
	}

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

	dec := json.NewDecoder(controlStream)
	var execReq dto.ExecRequest
	if err := dec.Decode(&execReq); err != nil || len(execReq.Command) == 0 {
		_ = json.NewEncoder(controlStream).Encode(dto.ControlMessage{
			Type: "error",
			Code: 1,
		})
		return
	}

	cmdSerialized := strings.Join(execReq.Command, "\x1f")

	execCmd := exec.Command("/proc/self/exe")
	execCmd.Stdin = stdinStream
	execCmd.Stdout = stdoutStream
	execCmd.Stderr = stderrStream
	execCmd.Env = append(os.Environ(),
		fmt.Sprintf("_VESSEL_EXEC_PID=%s", strconv.Itoa(c.PID)),
		fmt.Sprintf("_VESSEL_EXEC_CMD=%s", cmdSerialized),
	)

	exitCode := 0
	if err := execCmd.Run(); err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	go func() {
		for {
			var msg dto.ExecRequest
			if err := dec.Decode(&msg); err != nil {
				return
			}
		}
	}()

	_ = json.NewEncoder(controlStream).Encode(dto.ControlMessage{
		Type: "exit",
		Code: exitCode,
	})
}
