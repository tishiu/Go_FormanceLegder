package main

import (
	"os"
	"os/exec"
)

func main() {
	// Run integration tests
	cmd := exec.Command("go", "test", "./internal/integration", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = "/app"

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
