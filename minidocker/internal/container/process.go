package container

// ProcessManager gestionara señales y descriptores de terminal interactiva (PTY)
type ProcessManager struct {
	StdinChannel chan []byte
}