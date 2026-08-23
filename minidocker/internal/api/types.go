package api

import "time"

type ErrorResponse struct {
	Error string `json:"error"`
}

type CreateRequest struct {
	Image    string   `json:"image"`
	Command  []string `json:"command,omitempty"`
	Name     string   `json:"name,omitempty"`
	MemoryMB int64    `json:"memory_mb,omitempty"`
	PidsMax  int64    `json:"pids_max,omitempty"`
	Port     string   `json:"port,omitempty"`
	Env      []string `json:"env,omitempty"`
}

type CreateResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type StartRequest struct {
	CommandOverride []string `json:"command_override,omitempty"`
}

type ContainerDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Image     string    `json:"image"`
	Command   []string  `json:"command"`
	State     string    `json:"state"`
	PID       int       `json:"pid,omitempty"`
	IP        string    `json:"ip,omitempty"`
	HostPort  int       `json:"host_port,omitempty"`
	ContPort  int       `json:"container_port,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PsResponse struct {
	Containers []ContainerDTO `json:"containers"`
}

type StatsDTO struct {
	MemUsageBytes int64   `json:"mem_usage_bytes"`
	MemLimitBytes int64   `json:"mem_limit_bytes"`
	CPUUsageUsec  int64   `json:"cpu_usage_usec"`
	CPUPercent    float64 `json:"cpu_percent,omitempty"`
	PidsCurrent   int64   `json:"pids_current"`
	PidsMax       int64   `json:"pids_max"`
}
