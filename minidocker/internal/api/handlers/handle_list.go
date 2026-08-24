package handlers

import (
	"net/http"
	"path/filepath"

	"minidocker/internal/api/dto"
)

func (d *Deps) HandleList(w http.ResponseWriter, r *http.Request) {
	mgr := d.ManagerFor(r.URL.Query().Get("data_root"))
	dirs, err := mgr.ListContainerDirs()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err)
		return
	}

	resp := dto.PsResponse{Containers: []dto.ContainerView{}}
	for _, dir := range dirs {
		id := filepath.Base(dir)
		c, err := mgr.GetContainer(id)
		if err != nil {
			continue
		}
		resp.Containers = append(resp.Containers, ToDTO(c))
	}

	WriteJSON(w, http.StatusOK, resp)
}
