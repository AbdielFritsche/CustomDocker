package decorators

import (
	"fmt"
	"time"
)

// OpFunc es la firma de la operación que vamos a envolver
type OpFunc func() error

// WithCLIOutput es un decorador diseñado específicamente para la experiencia de usuario en la terminal
func WithCLIOutput(actionName string, fn OpFunc) error {
	start := time.Now()

	err := fn()
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("[\033[31mFAIL\033[0m] %s (Tomó: %v)\n", actionName, elapsed)
		fmt.Printf("         Error: %v\n", err)
		return err
	}

	fmt.Printf("[\033[32m OK \033[0m] %s (Tomó: %v)\n", actionName, elapsed)
	return nil
}
