package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"minidocker/internal/api/dto"
)

func (d *Deps) HandleStart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req dto.StartContainerRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // el body es opcional

	go func() {
		if err := d.Mgr.StartContainer(id, req.Command); err != nil {
			log.Printf("[minidockerd] error arrancando [%s]: %v", id, err)
		}
	}()

	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "starting", "id": id})
}
