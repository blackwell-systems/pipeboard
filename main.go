package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type BackendKind string

const (
	BackendDarwin  BackendKind = "darwin-pasteboard"
	BackendWayland BackendKind = "wayland-wl-copy"
	BackendX11     BackendKind = "x11-xclip"
	BackendWSL     BackendKind = "wsl-clip"
	BackendUnknown BackendKind = "unknown"
)

type Backend struct {
	Kind      BackendKind
	CopyCmd   []string
	PasteCmd  []string
	ClearCmd  []string // if empty, use CopyCmd with empty stdin
	Notes     string
	Missing   []string
	EnvSource string
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printHelp()
		return
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "copy":
		if err := cmdCopy(rest); err != nil {
			fatal(err)
		}
	case "paste":
		if err := cmdPaste(rest); err != nil {
			fatal(err)
		}
	case "clear":
		if err := cmdClear(rest); err != nil {
			fatal(err)
		}
	case "backend":
		if err := cmdBackend(rest); err != nil {
			fatal(err)
		}
	case "doctor":
		if err := cmdDoctor(rest); err != nil {
			fatal(err)
		}
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		fmt.Println("pipeboard v0.1.0")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`pipeboard - tiny cross-platform clipboard CLI

Usage:
  pipeboard copy    [text]   # copy stdin or provided text to clipboard
  pipeboard paste            # paste clipboard contents to stdout
  pipeboard clear            # clear clipboard (best-effort)
  pipeboard backend          # show detected backend
  pipeboard doctor           # check dependencies and environment
  pipeboard help             # show this help
  pipeboard version          # show version

Examples:
  echo "hello" | pipeboard copy
  pipeboard copy "hello world"
  pipeboard paste | jq .
  pipeboard clear`)
}

func cmdCopy(args []string) error {
	b, err := detectBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return fmt.Errorf("backend %s is missing required tools: %s", b.Kind, strings.Join(b.Missing, ", "))
	}
	data, err := readInputOrArgs(args)
	if err != nil {
		return err
	}
	return runWithInput(b.CopyCmd, data)
}

func cmdPaste(args []string) error {
	if len(args) > 0 {
		return errors.New("paste does not take arguments")
	}
	b, err := detectBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return fmt.Errorf("backend %s is missing required tools: %s", b.Kind, strings.Join(b.Missing, ", "))
	}
	return runAndPipeStdout(b.PasteCmd)
}

func cmdClear(args []string) error {
	if len(args) > 0 {
		return errors.New("clear does not take arguments")
	}
	b, err := detectBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return fmt.Errorf("backend %s is missing required tools: %s", b.Kind, strings.Join(b.Missing, ", "))
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
	b, err := detectBackend()
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
	if len(args) > 0 {
		return errors.New("doctor does not take arguments")
	}
	b, err := detectBackend()
	if err != nil {
		return err
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

func detectBackend() (*Backend, error) {
	goos := runtime.GOOS

	switch goos {
	case "darwin":
		return detectDarwin()
	case "linux":
		// Try Wayland, then X11, then WSL-friendly
		if b := detectWayland(); b != nil {
			return b, nil
		}
		if b := detectX11(); b != nil {
			return b, nil
		}
		if b := detectWSL(); b != nil {
			return b, nil
		}
		return &Backend{
			Kind: BackendUnknown,
			Notes: "No Wayland/X11/WSL clipboard command found. " +
				"Install wl-clipboard or xclip/xsel, or configure clip.exe for WSL.",
		}, nil
	case "windows":
		// Native Windows – try clip + powershell
		return detectWindows()
	default:
		return nil, fmt.Errorf("unsupported OS: %s", goos)
	}
}

func detectDarwin() (*Backend, error) {
	missing := []string{}
	if !hasCmd("pbcopy") {
		missing = append(missing, "pbcopy")
	}
	if !hasCmd("pbpaste") {
		missing = append(missing, "pbpaste")
	}
	return &Backend{
		Kind:     BackendDarwin,
		CopyCmd:  []string{"pbcopy"},
		PasteCmd: []string{"pbpaste"},
		Missing:  missing,
	}, nil
}

func detectWayland() *Backend {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return nil
	}
	missing := []string{}
	if !hasCmd("wl-copy") {
		missing = append(missing, "wl-copy")
	}
	if !hasCmd("wl-paste") {
		missing = append(missing, "wl-paste")
	}
	return &Backend{
		Kind:      BackendWayland,
		CopyCmd:   []string{"wl-copy"},
		PasteCmd:  []string{"wl-paste"},
		ClearCmd:  []string{"wl-copy", "--clear"},
		Missing:   missing,
		EnvSource: "WAYLAND_DISPLAY",
	}
}

func detectX11() *Backend {
	if os.Getenv("DISPLAY") == "" {
		return nil
	}
	missing := []string{}
	copyCmd := []string{"xclip", "-selection", "clipboard"}
	pasteCmd := []string{"xclip", "-selection", "clipboard", "-o"}

	if !hasCmd("xclip") {
		if hasCmd("xsel") {
			copyCmd = []string{"xsel", "--clipboard", "--input"}
			pasteCmd = []string{"xsel", "--clipboard", "--output"}
		} else {
			missing = append(missing, "xclip/xsel")
		}
	}

	return &Backend{
		Kind:      BackendX11,
		CopyCmd:   copyCmd,
		PasteCmd:  pasteCmd,
		Missing:   missing,
		EnvSource: "DISPLAY",
	}
}

func detectWSL() *Backend {
	// Heuristic: if clip.exe exists, assume WSL-style flow
	if !hasCmd("clip.exe") {
		return nil
	}
	missing := []string{}
	if !hasCmd("powershell.exe") {
		missing = append(missing, "powershell.exe")
	}
	return &Backend{
		Kind:     BackendWSL,
		CopyCmd:  []string{"clip.exe"},
		PasteCmd: []string{"powershell.exe", "-NoProfile", "-Command", "Get-Clipboard"},
		Missing:  missing,
		Notes:    "WSL detection based on clip.exe in PATH.",
	}
}

func detectWindows() (*Backend, error) {
	missing := []string{}
	if !hasCmd("clip") && !hasCmd("clip.exe") {
		missing = append(missing, "clip.exe")
	}
	if !hasCmd("powershell.exe") && !hasCmd("powershell") {
		missing = append(missing, "powershell.exe")
	}
	copyCmd := []string{"clip"}
	if hasCmd("clip.exe") {
		copyCmd = []string{"clip.exe"}
	}
	pasteCmd := []string{"powershell.exe", "-NoProfile", "-Command", "Get-Clipboard"}
	if !hasCmd("powershell.exe") && hasCmd("powershell") {
		pasteCmd[0] = "powershell"
	}
	return &Backend{
		Kind:     BackendWSL,
		CopyCmd:  copyCmd,
		PasteCmd: pasteCmd,
		Missing:  missing,
	}, nil
}

func hasCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

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

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "pipeboard: %v\n", err)
	os.Exit(1)
}
