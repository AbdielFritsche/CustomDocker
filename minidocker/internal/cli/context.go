package cli

import (
	"fmt"
	"minidocker/internal/container"
)

// GlobalDataRoot almacena la ruta que vendrá de la bandera persistente en root.go
var GlobalDataRoot string

func GetManager() *container.Manager {
	return container.NewManager(GlobalDataRoot)
}

// EnsureValidArgs valida que el usuario haya pasado exactamente la cantidad de argumentos requeridos
func EnsureValidArgs(args []string, expected int, usage string) error {
	if len(args) < expected {
		return fmt.Errorf("argumentos insuficientes. Uso: %s", usage)
	}
	return nil
}
