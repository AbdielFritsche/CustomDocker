package handlers

import (
	"minidocker/internal/network"
	"net/http"
)

func (d *Deps) HandleNetworkDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeErrorMsg(w, http.StatusBadRequest, "nombre de red requerido")
		return
	}

	bridgeName := name
	if len(name) < 3 || name[:3] != "br_" {
		bridgeName = "br_" + name
	}

	subnetCIDR := r.URL.Query().Get("subnet")

	if err := network.DeleteBridge(bridgeName, subnetCIDR); err != nil {
		writeErrorMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted", "bridge": bridgeName})
}
