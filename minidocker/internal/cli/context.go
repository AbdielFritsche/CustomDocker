package cli

import (
	"fmt"
	"minidocker/internal/container"
	"syscall"
)

// GlobalFlag almacena la raíz común
var globalDataRoot string

// GetManager devuelve una instancia lista de Manager con la ruta global configurada
func GetManager() *container.Manager {
	return container.NewManager(globalDataRoot)
}

// GetRunningContainer busca un contenedor y valida que su proceso esté vivo en el host
func GetRunningContainer(idOrName string) (*container.Container, error) {
	mgr := GetManager()
	c, err := mgr.GetContainer(idOrName)
	if err != nil {
		return nil, fmt.Errorf("contenedor [%s] no encontrado: %w", idOrName, err)
	}

	if c.State != container.StateRunning || c.PID <= 0 {
		return nil, fmt.Errorf("el contenedor [%s] no está en ejecución (estado: %s)", idOrName, c.State)
	}

	if err := syscall.Kill(c.PID, 0); err != nil {
		return nil, fmt.Errorf("el proceso del contenedor (PID %d) ya no existe", c.PID)
	}

	return c, nil
}
