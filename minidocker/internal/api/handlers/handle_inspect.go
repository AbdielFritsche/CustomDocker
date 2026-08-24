package handlers

import "net/http"

func (d *Deps) HandleInspect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	mgr := d.ManagerFor(r.URL.Query().Get("data_root"))
	c, err := mgr.GetContainer(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, err)
		return
	}

	WriteJSON(w, http.StatusOK, ToDTO(c))
}
