package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

func runCommand(cmdParts []string) error {
	if len(cmdParts) == 0 {
		return errors.New("no command configured")
	}
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runWithInput(cmdParts []string, data []byte) error {
	if len(cmdParts) == 0 {
		return errors.New("no command configured")
	}
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = os.Stdout // some tools might print warnings
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runAndPipeStdout(cmdParts []string) error {
	if len(cmdParts) == 0 {
		return errors.New("no command configured")
	}
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func readInputOrArgs(args []string) ([]byte, error) {
	if len(args) > 0 {
		// Treat arguments as the text to copy
		return []byte(strings.Join(args, " ")), nil
	}
	// Read from stdin until EOF
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, os.Stdin); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
