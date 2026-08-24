package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"minidocker/internal/api/dto"
	"minidocker/internal/container"
)

func (d *Deps) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("body inválido: %w", err))
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

	c, err := d.Mgr.CreateContainer(req.Image, cmd, opts...)
	if err != nil {
		// Nombre duplicado, etc. -> error del cliente, no del servidor.
		WriteError(w, http.StatusConflict, err)
		return
	}

	WriteJSON(w, http.StatusCreated, dto.CreateContainerResponse{ID: c.Config.ID, Name: c.Config.Name})
}
