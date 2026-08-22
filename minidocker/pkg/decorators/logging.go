package decorators

import (
	"fmt"
	"time"
)

// OpFunc es la firma base de cualquier operación dentro del runtime
type OpFunc func() error

// WithLogging es un decorador que mide el tiempo de ejecución y captura errores
func WithLogging(operationName string, fn OpFunc) error {
	start := time.Now()
	err := fn()
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("[\033[31mFAIL\033[0m] %s en %v: %v\n", operationName, elapsed, err)
		return err
	}
	fmt.Printf("[\033[32mOK\033[0m] %s completado en %v\n", operationName, elapsed)
	return nil
}
