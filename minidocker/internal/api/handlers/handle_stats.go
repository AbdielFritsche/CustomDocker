package handlers

import (
	"fmt"
	"net/http"

	"minidocker/internal/api/dto"
	"minidocker/internal/container"
	"minidocker/internal/isolation"
)

func (d *Deps) HandleStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := d.Mgr.GetContainer(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, err)
		return
	}
	if c.State != container.StateRunning {
		WriteError(w, http.StatusConflict, fmt.Errorf("el contenedor [%s] no está en ejecución", id))
		return
	}

	cg := isolation.NewCgroupManager(c.Config.ID)
	stats, err := cg.ReadStats()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err)
		return
	}

	WriteJSON(w, http.StatusOK, dto.StatsResponse{
		MemUsageBytes: stats.MemoryUsageBytes,
		MemLimitBytes: stats.MemoryLimitBytes,
		CPUUsageUsec:  stats.CPUUsageUsec,
		PidsCurrent:   stats.PidsCurrent,
		PidsMax:       stats.PidsMax,
	})
}
