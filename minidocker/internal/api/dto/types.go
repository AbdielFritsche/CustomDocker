package dto

import (
	"time"

	"minidocker/internal/container"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type ExecRequest struct {
	Command []string `json:"command"`
}

type ComposeRequest struct {
	ComposeYAML string `json:"compose_yaml"`
}

type ComposeResponse struct {
	Status string `json:"status"`
}

type CreateContainerRequest struct {
	Image     string   `json:"image"`
	Command   []string `json:"command"`
	Name      string   `json:"name,omitempty"`
	MemoryMB  int64    `json:"memory_mb,omitempty"`
	PidsMax   int64    `json:"pids_max,omitempty"`
	Port      string   `json:"port,omitempty"` // "host:container"
	Env       []string `json:"env,omitempty"`
	DataRoot  string   `json:"data_root,omitempty"`
	AutoStart bool     `json:"auto_start,omitempty"`
}

type CreateContainerResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type StartContainerRequest struct {
	Command []string `json:"command,omitempty"`
	Attach  bool     `json:"attach,omitempty"`
}

type PsResponse struct {
	Containers []ContainerView `json:"containers"`
}

type StartContainerResponse struct {
	ID  string `json:"id"`
	PID int    `json:"pid"`
}

type StopContainerResponse struct {
	ID string `json:"id"`
}

type DeleteContainerResponse struct {
	ID string `json:"id"`
}

type ContainerView struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Image       string                 `json:"image"`
	Command     []string               `json:"command"`
	State       container.State        `json:"state"`
	PID         int                    `json:"pid"`
	ExitCode    int                    `json:"exit_code"`
	IP          string                 `json:"ip"`
	PortMapping *container.PortMapping `json:"port_mapping,omitempty"`
	HostPort    int                    `json:"host_port,omitempty"`
	ContPort    int                    `json:"cont_port,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   time.Time              `json:"started_at,omitempty"`
	StoppedAt   time.Time              `json:"stopped_at,omitempty"`
}

func NewContainerView(c *container.Container) ContainerView {
	return ContainerView{
		ID:          c.Config.ID,
		Name:        c.Config.Name,
		Image:       c.Config.Image,
		Command:     c.Config.Command,
		State:       c.State,
		PID:         c.PID,
		ExitCode:    c.ExitCode,
		IP:          c.Config.IP,
		PortMapping: c.Config.PortMapping,
		CreatedAt:   c.Config.CreatedAt,
		StartedAt:   c.StartedAt,
		StoppedAt:   c.StoppedAt,
	}
}

type ListContainersResponse struct {
	Containers []ContainerView `json:"containers"`
}

type StatsResponse struct {
	ID            string  `json:"id,omitempty"`
	Name          string  `json:"name,omitempty"`
	MemUsageBytes int64   `json:"mem_usage_bytes"`
	MemLimitBytes int64   `json:"mem_limit_bytes"`
	CPUUsageUsec  int64   `json:"cpu_usage_usec"`
	CPUPercent    float64 `json:"cpu_percent,omitempty"`
	PidsCurrent   int64   `json:"pids_current"`
	PidsMax       int64   `json:"pids_max"`
}

type ControlMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
	Sign int    `json:"sign,omitempty"`
	Code int    `json:"code,omitempty"`
}
