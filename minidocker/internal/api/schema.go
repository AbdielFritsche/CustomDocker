package api

import "time"

type CreateRequest struct {
	Image    string   `json:"image"`
	Name     string   `json:"name"`
	Command  []string `json:"command"`
	Env      []string `json:"env,omitempty"`
	MemoryMB int64    `json:"memory_mb,omitempty"`
	PidsMax  int64    `json:"pids_max,omitempty"`
	Ports    []string `json:"ports,omitempty"`
	Volumes  []string `json:"volumes,omitempty"`
	Network  string   `json:"network,omitempty"`
}

type ContainerResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	State     string    `json:"state"`
	IP        string    `json:"ip"`
	PID       int       `json:"pid"`
	CreatedAt time.Time `json:"created_at"`
	Ports     string    `json:"ports"`
}

type StatusResponse struct {
	DaemonPID  int    `json:"daemon_pid"`
	Containers int    `json:"active_containers"`
	UptimeSec  int64  `json:"uptime_sec"`
	Status     string `json:"status"`
}
