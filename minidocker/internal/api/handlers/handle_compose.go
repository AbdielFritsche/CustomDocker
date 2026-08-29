package handlers

import (
	"encoding/json"
	"net/http"

	"minidocker/internal/api/dto"
	"minidocker/internal/compose"
)

func (d *Deps) HandleComposeUp(w http.ResponseWriter, r *http.Request) {
	var req dto.ComposeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, err)
		return
	}

	cf, err := compose.ParseComposeBytes([]byte(req.ComposeYAML))
	if err != nil {
		WriteError(w, http.StatusBadRequest, err)
		return
	}

	engine := compose.NewEngine(d.Mgr)
	if err := engine.Up(cf); err != nil {
		WriteError(w, http.StatusInternalServerError, err)
		return
	}

	WriteJSON(w, http.StatusOK, dto.ComposeResponse{Status: "up"})
}

func (d *Deps) HandleComposeDown(w http.ResponseWriter, r *http.Request) {
	var req dto.ComposeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, err)
		return
	}

	cf, err := compose.ParseComposeBytes([]byte(req.ComposeYAML))
	if err != nil {
		WriteError(w, http.StatusBadRequest, err)
		return
	}

	engine := compose.NewEngine(d.Mgr)
	if err := engine.Down(cf); err != nil {
		WriteError(w, http.StatusInternalServerError, err)
		return
	}

	WriteJSON(w, http.StatusOK, dto.ComposeResponse{Status: "down"})
}
