package isolation

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ContainerStats representa las métricas en un punto en el tiempo
type ContainerStats struct {
	MemoryUsageBytes int64
	MemoryLimitBytes int64
	CPUUsageUsec     int64
	PidsCurrent      int64
	PidsMax          int64
}

// ReadStats lee las métricas directamente de los archivos de cgroups v2
func (c *CgroupManager) ReadStats() (ContainerStats, error) {
	var stats ContainerStats

	// 1. Memoria Actual (memory.current)
	if data, err := os.ReadFile(filepath.Join(c.Path, "memory.current")); err == nil {
		stats.MemoryUsageBytes, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}

	// 2. Límite de Memoria (memory.max)
	if data, err := os.ReadFile(filepath.Join(c.Path, "memory.max")); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "max" {
			stats.MemoryLimitBytes, _ = strconv.ParseInt(val, 10, 64)
		}
	}

	// 3. CPU Stats (cpu.stat -> usage_usec)
	if file, err := os.Open(filepath.Join(c.Path, "cpu.stat")); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) == 2 && fields[0] == "usage_usec" {
				stats.CPUUsageUsec, _ = strconv.ParseInt(fields[1], 10, 64)
				break
			}
		}
		_ = scanner.Err() // Chequeo del linter para scanner.Err()
	}

	// 4. Procesos Actuales (pids.current) y Límite (pids.max)
	if data, err := os.ReadFile(filepath.Join(c.Path, "pids.current")); err == nil {
		stats.PidsCurrent, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}
	if data, err := os.ReadFile(filepath.Join(c.Path, "pids.max")); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "max" {
			stats.PidsMax, _ = strconv.ParseInt(val, 10, 64)
		}
	}

	return stats, nil
}
