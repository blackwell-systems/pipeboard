package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func cmdCopy(args []string) error {
	// Check for --image flag
	imageMode := false
	var filteredArgs []string
	for _, arg := range args {
		if arg == "--image" || arg == "-i" {
			imageMode = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	b, err := getBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return missingToolsError(b)
	}

	if imageMode {
		if len(b.ImageCopyCmd) == 0 {
			return fmt.Errorf("image copy not supported on backend %s", b.Kind)
		}
		// For image mode, read from stdin only (no text args)
		if len(filteredArgs) > 0 {
			return errors.New("--image mode reads PNG data from stdin, does not accept text arguments")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		return runWithInput(b.ImageCopyCmd, data)
	}

	data, err := readInputOrArgs(filteredArgs)
	if err != nil {
		return err
	}
	return runWithInput(b.CopyCmd, data)
}

func cmdPaste(args []string) error {
	// Check for --image flag
	imageMode := false
	for _, arg := range args {
		if arg == "--image" || arg == "-i" {
			imageMode = true
		} else {
			return fmt.Errorf("unknown argument: %s", arg)
		}
	}

	b, err := getBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return missingToolsError(b)
	}

	if imageMode {
		if len(b.ImagePasteCmd) == 0 {
			return fmt.Errorf("image paste not supported on backend %s", b.Kind)
		}
		return runAndPipeStdout(b.ImagePasteCmd)
	}

	return runAndPipeStdout(b.PasteCmd)
}

func cmdClear(args []string) error {
	if len(args) > 0 {
		return errors.New("clear does not take arguments")
	}
	b, err := getBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return missingToolsError(b)
	}

	if len(b.ClearCmd) > 0 {
		return runCommand(b.ClearCmd)
	}

	// Fallback: copy empty string
	return runWithInput(b.CopyCmd, []byte{})
}

func cmdBackend(args []string) error {
	if len(args) > 0 {
		return errors.New("backend does not take arguments")
	}
	b, err := getBackend()
	if err != nil {
		return err
	}

	fmt.Printf("Backend:   %s\n", b.Kind)
	fmt.Printf("OS:        %s\n", runtime.GOOS)
	if b.EnvSource != "" {
		fmt.Printf("Env:       %s\n", b.EnvSource)
	}
	fmt.Printf("Copy cmd:  %s\n", strings.Join(b.CopyCmd, " "))
	fmt.Printf("Paste cmd: %s\n", strings.Join(b.PasteCmd, " "))
	if len(b.ClearCmd) > 0 {
		fmt.Printf("Clear cmd: %s\n", strings.Join(b.ClearCmd, " "))
	}
	if len(b.Missing) > 0 {
		fmt.Printf("Missing:   %s\n", strings.Join(b.Missing, ", "))
	}
	if b.Notes != "" {
		fmt.Printf("Notes:     %s\n", b.Notes)
	}
	return nil
}

func cmdDoctor(args []string) error {
	var jsonOutput bool
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown flag: %s\nusage: pipeboard doctor [--json]", arg)
		}
	}

	b, err := getBackend()
	if err != nil {
		return err
	}

	if jsonOutput {
		status := "ok"
		if len(b.Missing) > 0 || b.Kind == BackendUnknown {
			status = "warning"
		}
		result := struct {
			OS        string   `json:"os"`
			Backend   string   `json:"backend"`
			EnvSource string   `json:"env_source,omitempty"`
			Status    string   `json:"status"`
			CopyCmd   []string `json:"copy_cmd"`
			PasteCmd  []string `json:"paste_cmd"`
			ClearCmd  []string `json:"clear_cmd,omitempty"`
			Missing   []string `json:"missing,omitempty"`
			Notes     string   `json:"notes,omitempty"`
		}{
			OS:        runtime.GOOS,
			Backend:   string(b.Kind),
			EnvSource: b.EnvSource,
			Status:    status,
			CopyCmd:   b.CopyCmd,
			PasteCmd:  b.PasteCmd,
			ClearCmd:  b.ClearCmd,
			Missing:   b.Missing,
			Notes:     b.Notes,
		}
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Println("pipeboard doctor")
	fmt.Println("-----------------")
	fmt.Printf("OS:       %s\n", runtime.GOOS)
	fmt.Printf("Backend:  %s\n", b.Kind)
	if b.EnvSource != "" {
		fmt.Printf("Env:      %s\n", b.EnvSource)
	}

	if len(b.Missing) == 0 && b.Kind != BackendUnknown {
		fmt.Println("\nStatus:   OK ✅")
		fmt.Println("Details:  All required commands for this backend are available.")
	} else {
		fmt.Println("\nStatus:   WARNING ⚠️")
		if b.Kind == BackendUnknown {
			fmt.Println("Details:  Could not detect a suitable clipboard backend for this environment.")
		}
		if len(b.Missing) > 0 {
			fmt.Printf("Missing:  %s\n", strings.Join(b.Missing, ", "))
		}
	}

	fmt.Println("\nTips:")
	fmt.Println("  - On macOS:   pbcopy / pbpaste should be available by default.")
	fmt.Println("  - On Wayland: install `wl-clipboard` (wl-copy, wl-paste).")
	fmt.Println("  - On X11:     install `xclip` or `xsel`.")
	fmt.Println("  - On WSL:     ensure `clip.exe` and `powershell.exe` are in PATH.")

	return nil
}

// readClipboard reads the current local clipboard contents
func readClipboard() ([]byte, error) {
	b, err := getBackend()
	if err != nil {
		return nil, err
	}
	if len(b.Missing) > 0 {
		return nil, missingToolsError(b)
	}
	if len(b.PasteCmd) == 0 {
		return nil, errors.New("no paste command configured")
	}

	cmd := exec.Command(b.PasteCmd[0], b.PasteCmd[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("reading clipboard: %w", err)
	}
	return out.Bytes(), nil
}

// writeClipboard writes data to the local clipboard
func writeClipboard(data []byte) error {
	b, err := getBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return missingToolsError(b)
	}
	return runWithInput(b.CopyCmd, data)
}
