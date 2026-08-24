package handlers

import "net/http"

func (d *Deps) HandleStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := d.Mgr.StopContainer(id); err != nil {
		WriteError(w, http.StatusInternalServerError, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "stopped", "id": id})
}
