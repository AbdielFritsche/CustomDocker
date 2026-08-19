package isolation

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const cgroupBasePath = "/sys/fs/cgroup/minidocker"

// CgroupLimits define los límites de recursos a aplicar
type CgroupLimits struct {
	MemoryLimitBytes int64 // Memoria en bytes
	PidsMax          int64 // Cantidad máxima de procesos
}

// CgroupManager administra el ciclo de vida del cgroup de un contenedor
type CgroupManager struct {
	ContainerID string
	Path        string
}

// NewCgroupManager inicializa la referencia al cgroup
func NewCgroupManager(containerID string) *CgroupManager {
	return &CgroupManager{
		ContainerID: containerID,
		Path:        filepath.Join(cgroupBasePath, containerID),
	}
}

// enableControllers activa los controladores en un nivel del cgroup
func enableControllers(path string) {
	subtreeFile := filepath.Join(path, "cgroup.subtree_control")
	// Intentamos activar los controladores soportados (+pids +memory +cpu)
	_ = os.WriteFile(subtreeFile, []byte("+pids +memory +cpu"), 0644)
}

// Apply crea el grupo y escribe los límites configurados
func (c *CgroupManager) Apply(pid int, limits CgroupLimits) error {
	// 1. Habilitar controladores en la raíz /sys/fs/cgroup
	enableControllers("/sys/fs/cgroup")

	// 2. Crear el directorio base /sys/fs/cgroup/minidocker y activar controladores
	if err := os.MkdirAll(cgroupBasePath, 0755); err != nil {
		return fmt.Errorf("error creando cgroup base: %w", err)
	}
	enableControllers(cgroupBasePath)

	// 3. Crear el subdirectorio específico para este contenedor
	if err := os.MkdirAll(c.Path, 0755); err != nil {
		return fmt.Errorf("error creando cgroup en %s: %w", c.Path, err)
	}

	// 4. Inscribir primero el PID en cgroup.procs
	procsFile := filepath.Join(c.Path, "cgroup.procs")
	if err := os.WriteFile(procsFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("error inscribiendo PID %d: %w", pid, err)
	}

	// 5. Limitar número de procesos (pids.max)
	if limits.PidsMax > 0 {
		pidsFile := filepath.Join(c.Path, "pids.max")
		if err := os.WriteFile(pidsFile, []byte(strconv.FormatInt(limits.PidsMax, 10)), 0644); err != nil {
			return fmt.Errorf("error asignando pids.max (%d): %w", limits.PidsMax, err)
		}
	}

	// 6. Limitar consumo de memoria (memory.max)
	if limits.MemoryLimitBytes > 0 {
		memFile := filepath.Join(c.Path, "memory.max")
		if err := os.WriteFile(memFile, []byte(strconv.FormatInt(limits.MemoryLimitBytes, 10)), 0644); err != nil {
			return fmt.Errorf("error asignando memory.max (%d): %w", limits.MemoryLimitBytes, err)
		}
	}

	return nil
}

func (c *CgroupManager) KillAll() error {
	killFile := filepath.Join(c.Path, "cgroup.kill")
	if _, err := os.Stat(killFile); err == nil {
		return os.WriteFile(killFile, []byte("1"), 0644)
	}

	// Fallback para cgroups v2 si cgroup.kill no estuviera disponible:
	// Leer todos los PIDs restantes en cgroup.procs y enviarles SIGKILL
	procsFile := filepath.Join(c.Path, "cgroup.procs")
	data, err := os.ReadFile(procsFile)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if p, err := strconv.Atoi(line); err == nil && p > 0 {
				_ = syscall.Kill(p, syscall.SIGKILL)
			}
		}
	}
	return nil
}

// Cleanup elimina el directorio del cgroup tras finalizar el contenedor
func (c *CgroupManager) Cleanup() error {
	if _, err := os.Stat(c.Path); err == nil {
		_ = os.Remove(c.Path)
	}
	return nil
}
