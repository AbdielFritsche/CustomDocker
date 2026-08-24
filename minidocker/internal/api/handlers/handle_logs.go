package handlers

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"minidocker/internal/container"
)

func (d *Deps) HandleLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	follow := r.URL.Query().Get("follow") == "true"
	mgr := d.ManagerFor(r.URL.Query().Get("data_root"))

	c, err := mgr.GetContainer(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, err)
		return
	}

	logPath := filepath.Join(c.Config.BasePath, c.Config.ID, "container.log")
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			WriteError(w, http.StatusNotFound, fmt.Errorf("no hay registros para [%s]", id))
			return
		}
		WriteError(w, http.StatusInternalServerError, err)
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
