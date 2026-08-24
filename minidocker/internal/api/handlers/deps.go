package handlers

import (
	"encoding/json"
	"net/http"

	"minidocker/internal/api/dto"
	"minidocker/internal/container"
)

type Deps struct {
	Mgr *container.Manager
}

func (d *Deps) ManagerFor(dataRoot string) *container.Manager {
	if dataRoot == "" {
		return d.Mgr
	}
	return container.NewManager(dataRoot)
}

func ToDTO(c *container.Container) dto.ContainerView {
	view := dto.ContainerView{
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
		view.HostPort = c.Config.PortMapping.HostPort
		view.ContPort = c.Config.PortMapping.ContainerPort
	}
	return view
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, status int, err error) {
	WriteJSON(w, status, dto.ErrorResponse{Error: err.Error()})
}

func writeErrorMsg(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, dto.ErrorResponse{Error: msg})
}
