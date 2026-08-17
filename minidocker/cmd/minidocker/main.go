package main

import (
	"fmt"
	"os"

	"minidocker/internal/cli"
	"minidocker/internal/isolation"
)

func main() {
	// 1. Interceptar el punto de entrada interno para el subproceso hijo aislado
	if len(os.Args) >= 2 && os.Args[1] == "__init__" {
		rootfs := os.Args[2]
		userCommand := os.Args[3:]

		if err := isolation.RunChild(rootfs, userCommand); err != nil {
			fmt.Fprintf(os.Stderr, "Error en child: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 2. Ejecutar la interfaz CLI de Cobra
	cli.Execute()
}