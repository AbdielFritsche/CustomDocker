package container

import (
	"time"
	"minidocker/internal/isolation"
)

// State representa el estado actual del contenedor en su ciclo de vida
type State string
const (
	StateCreated State = "created"
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateFailed State = "failed"
)

// Config contiene la especificación base para crear un contenedor
type Config struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Image     string                 `json:"image"`
	Command   []string               `json:"command"`
	RootFS    string                 `json:"rootfs"`
	Limits    isolation.CgroupLimits `json:"limits"`
	CreatedAt time.Time              `json:"created_at"`
}

// Container representa la entidad viva y persistida
type Container struct {
	Config    Config    `json:"config"`
	State     State     `json:"state"`
	PID       int       `json:"pid"`
	ExitCode  int       `json:"exit_code"`
	StartedAt time.Time `json:"started_at"`
	StoppedAt time.Time `json:"stopped_at"`
}

type Option func(*Config)

// WithMemoryLimit define el limite de memoria RAM
func WithMemoryLimit(bytes int64) Option {
	return func(c *Config) {
		c.Limits.MemoryLimitBytes = bytes
	}
}

// WithPidsMax define el limite maximo de procesos simultaneos
func WithPidsMax(pids int64) Option {
	return func (c *Config) {
		c.Limits.PidsMax = pids
	}
}


// WithName asigna un nombre personalizado al contenedor 
func WithName(name string ) Option {
	return func(c *Config) {
		c.Name = name
	}
}
