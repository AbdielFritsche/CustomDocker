package handlers

import "net/http"

func (d *Deps) HandlePing(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
