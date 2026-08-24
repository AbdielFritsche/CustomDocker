package main

import (
	"fmt"
	"os"

	"minidocker/internal/cli"
	"minidocker/internal/isolation"
	"minidocker/internal/network"
)

func main() {
	// 1. Interceptar punto de entrada para el Daemon DNS independiente
	if len(os.Args) >= 3 && os.Args[1] == "__dnsd__" {
		gatewayIP := os.Args[2]
		if err := network.RunDNSDaemon(gatewayIP); err != nil {
			os.Exit(1)
		}
		return
	}

	// 2. Interceptar el punto de entrada interno para el subproceso hijo aislado
	if len(os.Args) >= 3 && (os.Args[1] == "init" || os.Args[1] == "__init__") {
		rootfs := os.Args[2]
		userCommand := os.Args[3:]

		// os.Environ() ya incluye las variables propagadas por cmd.Env en RunParent
		if err := isolation.RunChild(rootfs, userCommand); err != nil {
			fmt.Fprintf(os.Stderr, "Error en child: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 2. Ejecutar la interfaz CLI de Cobra
	cli.Execute()
}
