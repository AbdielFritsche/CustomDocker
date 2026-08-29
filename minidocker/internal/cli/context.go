package cli

import (
	"fmt"

	"minidocker/internal/api"
)

// GlobalDataRoot almacena la ruta que vendrá de la bandera persistente en root.go
var GlobalSocketPath string

func GetAPIClient() *api.Client {
	return api.NewClient(GlobalSocketPath)
}

// EnsureValidArgs valida que el usuario haya pasado exactamente la cantidad de argumentos requeridos
func EnsureValidArgs(args []string, expected int, usage string) error {
	if len(args) < expected {
		return fmt.Errorf("argumentos insuficientes. Uso: %s", usage)
	}
	return nil
}
