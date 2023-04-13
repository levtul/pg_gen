package domain

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func RunFormatter(filename string) (string, error) {
	path := filename
	_, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: os.ReadFile: %w", filename, err)
	}

	cmd := exec.Command("pg_format", "-N", path)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("stderr: %s", stderr.String())
		return "", fmt.Errorf("cannot run pg_format: %w", err)
	}

	return stdout.String(), nil
}
