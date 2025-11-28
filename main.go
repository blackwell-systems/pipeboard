package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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

	commands := map[string]func([]string) error{
		"copy":    cmdCopy,
		"paste":   cmdPaste,
		"clear":   cmdClear,
		"backend": cmdBackend,
		"doctor":  cmdDoctor,
		"push":    cmdPush,
		"pull":    cmdPull,
		"show":    cmdShow,
		"slots":   cmdSlots,
		"rm":      cmdRm,
		"send":    cmdSend,
		"recv":    cmdRecv,
		"receive": cmdRecv,
		"peek":    cmdPeek,
		"history": cmdHistory,
	}

	if fn, ok := commands[cmd]; ok {
		if err := fn(rest); err != nil {
			fatal(err)
		}
		return
	}

	switch cmd {
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		fmt.Println("pipeboard v0.2.0")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`pipeboard - tiny cross-platform clipboard CLI

Usage:
  pipeboard <command> [args...]

Local clipboard:
  copy [text]          Copy stdin or provided text to clipboard
  paste                Paste clipboard contents to stdout
  clear                Clear clipboard (best-effort)
  backend              Show detected clipboard backend
  doctor               Run environment checks

Direct peer-to-peer (SSH):
  send [peer]          Send local clipboard to peer's clipboard
  recv [peer]          Receive peer's clipboard into local clipboard
  peek [peer]          Print peer's clipboard to stdout (no local change)
                       (peer defaults to 'defaults.peer' in config)

Remote slots (optional sync backend):
  push <name>          Push clipboard to remote slot
  pull <name>          Pull remote slot into clipboard
  show <name>          Print remote slot to stdout
  slots                List remote slots
  rm <name>            Delete remote slot
  history              Show recent operations

Other:
  help                 Show this help
  version              Show version

Config: ~/.config/pipeboard/config.yaml

  defaults:
    peer: dev              # default peer for send/recv/peek

  peers:
    dev:
      ssh: devbox

  sync:
    backend: s3
    encryption: aes256     # client-side encryption (optional)
    passphrase: secret     # encryption passphrase
    ttl_days: 30           # auto-expire slots (optional)
    s3:
      bucket: my-bucket
      region: us-west-2

Examples:
  echo "hello" | pipeboard copy
  pipeboard paste | jq .
  pipeboard send                    # uses default peer
  pipeboard send dev
  pipeboard push kube && ssh server "pipeboard pull kube"`)
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

// readClipboard reads the current local clipboard contents
func readClipboard() ([]byte, error) {
	b, err := detectBackend()
	if err != nil {
		return nil, err
	}
	if len(b.Missing) > 0 {
		return nil, fmt.Errorf("backend %s is missing required tools: %s", b.Kind, strings.Join(b.Missing, ", "))
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
	b, err := detectBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return fmt.Errorf("backend %s is missing required tools: %s", b.Kind, strings.Join(b.Missing, ", "))
	}
	return runWithInput(b.CopyCmd, data)
}

func cmdPush(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: pipeboard push <name>")
	}
	slot := args[0]

	// Read from local clipboard
	data, err := readClipboard()
	if err != nil {
		return err
	}

	// Get remote backend
	backend, err := newRemoteBackendFromConfig()
	if err != nil {
		return err
	}

	host, _ := os.Hostname()
	meta := map[string]string{"hostname": host}

	// Push to remote
	if err := backend.Push(slot, data, meta); err != nil {
		return err
	}

	fmt.Printf("pushed %s to slot %q\n", formatSize(int64(len(data))), slot)
	recordHistory("push", slot, int64(len(data)))
	return nil
}

func cmdPull(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: pipeboard pull <name>")
	}
	slot := args[0]

	backend, err := newRemoteBackendFromConfig()
	if err != nil {
		return err
	}

	data, meta, err := backend.Pull(slot)
	if err != nil {
		return err
	}

	if err := writeClipboard(data); err != nil {
		return err
	}

	host := meta["hostname"]
	if host != "" {
		fmt.Printf("pulled %s from slot %q (source: %s)\n", formatSize(int64(len(data))), slot, host)
	} else {
		fmt.Printf("pulled %s from slot %q\n", formatSize(int64(len(data))), slot)
	}
	recordHistory("pull", slot, int64(len(data)))
	return nil
}

func cmdShow(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: pipeboard show <name>")
	}
	slot := args[0]

	backend, err := newRemoteBackendFromConfig()
	if err != nil {
		return err
	}

	data, _, err := backend.Pull(slot)
	if err != nil {
		return err
	}

	// Write to stdout instead of clipboard
	_, err = os.Stdout.Write(data)
	return err
}

func cmdSlots(args []string) error {
	if len(args) > 0 {
		return errors.New("slots does not take arguments")
	}

	backend, err := newRemoteBackendFromConfig()
	if err != nil {
		return err
	}

	slots, err := backend.List()
	if err != nil {
		return err
	}

	if len(slots) == 0 {
		fmt.Println("No slots found.")
		return nil
	}

	// Print header
	fmt.Printf("%-20s  %-10s  %-12s\n", "NAME", "SIZE", "AGE")

	for _, s := range slots {
		fmt.Printf("%-20s  %-10s  %-12s\n",
			s.Name,
			formatSize(s.Size),
			formatAge(s.CreatedAt),
		)
	}

	return nil
}

func cmdRm(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: pipeboard rm <name>")
	}
	slot := args[0]

	backend, err := newRemoteBackendFromConfig()
	if err != nil {
		return err
	}

	if err := backend.Delete(slot); err != nil {
		return err
	}

	fmt.Printf("deleted slot %q\n", slot)
	return nil
}

func cmdSend(args []string) error {
	cfg, err := loadConfigForPeers()
	if err != nil {
		return err
	}

	var peerName string
	if len(args) == 0 {
		peerName, err = cfg.getDefaultPeer()
		if err != nil {
			return fmt.Errorf("usage: pipeboard send [peer]\n%w", err)
		}
	} else if len(args) == 1 {
		peerName = args[0]
	} else {
		return fmt.Errorf("usage: pipeboard send [peer]")
	}

	peer, err := cfg.getPeer(peerName)
	if err != nil {
		return err
	}

	data, err := readClipboard()
	if err != nil {
		return err
	}

	sshTarget := peer.SSH
	remoteCmd := peer.RemoteCmd

	cmd := exec.Command("ssh", sshTarget, remoteCmd, "copy")
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send to peer %q (%s): %w", peerName, sshTarget, err)
	}

	fmt.Printf("sent %s to peer %q (%s)\n", formatSize(int64(len(data))), peerName, sshTarget)
	recordHistory("send", peerName, int64(len(data)))
	return nil
}

func cmdRecv(args []string) error {
	cfg, err := loadConfigForPeers()
	if err != nil {
		return err
	}

	var peerName string
	if len(args) == 0 {
		peerName, err = cfg.getDefaultPeer()
		if err != nil {
			return fmt.Errorf("usage: pipeboard recv [peer]\n%w", err)
		}
	} else if len(args) == 1 {
		peerName = args[0]
	} else {
		return fmt.Errorf("usage: pipeboard recv [peer]")
	}

	peer, err := cfg.getPeer(peerName)
	if err != nil {
		return err
	}

	sshTarget := peer.SSH
	remoteCmd := peer.RemoteCmd

	var out bytes.Buffer
	cmd := exec.Command("ssh", sshTarget, remoteCmd, "paste")
	cmd.Stdin = nil
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to receive from peer %q (%s): %w", peerName, sshTarget, err)
	}

	if err := writeClipboard(out.Bytes()); err != nil {
		return err
	}

	fmt.Printf("received %s from peer %q (%s)\n", formatSize(int64(out.Len())), peerName, sshTarget)
	recordHistory("recv", peerName, int64(out.Len()))
	return nil
}

func cmdPeek(args []string) error {
	cfg, err := loadConfigForPeers()
	if err != nil {
		return err
	}

	var peerName string
	if len(args) == 0 {
		peerName, err = cfg.getDefaultPeer()
		if err != nil {
			return fmt.Errorf("usage: pipeboard peek [peer]\n%w", err)
		}
	} else if len(args) == 1 {
		peerName = args[0]
	} else {
		return fmt.Errorf("usage: pipeboard peek [peer]")
	}

	peer, err := cfg.getPeer(peerName)
	if err != nil {
		return err
	}

	sshTarget := peer.SSH
	remoteCmd := peer.RemoteCmd

	cmd := exec.Command("ssh", sshTarget, remoteCmd, "paste")
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to peek from peer %q (%s): %w", peerName, sshTarget, err)
	}

	recordHistory("peek", peerName, 0)
	return nil
}

// History types and functions

type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Command   string    `json:"command"`
	Target    string    `json:"target"`
	Size      int64     `json:"size,omitempty"`
}

const maxHistoryEntries = 50

func getHistoryPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "pipeboard", "history.json")
}

func recordHistory(command, target string, size int64) {
	path := getHistoryPath()
	if path == "" {
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// Load existing history
	var history []HistoryEntry
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &history)
	}

	// Add new entry
	history = append(history, HistoryEntry{
		Timestamp: time.Now(),
		Command:   command,
		Target:    target,
		Size:      size,
	})

	// Trim to max entries
	if len(history) > maxHistoryEntries {
		history = history[len(history)-maxHistoryEntries:]
	}

	// Save
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0600)
}

func cmdHistory(args []string) error {
	if len(args) > 0 {
		return errors.New("history does not take arguments")
	}

	path := getHistoryPath()
	if path == "" {
		return errors.New("could not determine history path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No history yet.")
			return nil
		}
		return err
	}

	var history []HistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return err
	}

	if len(history) == 0 {
		fmt.Println("No history yet.")
		return nil
	}

	fmt.Printf("%-20s  %-8s  %-15s  %s\n", "TIME", "COMMAND", "TARGET", "SIZE")
	for _, h := range history {
		sizeStr := ""
		if h.Size > 0 {
			sizeStr = formatSize(h.Size)
		}
		fmt.Printf("%-20s  %-8s  %-15s  %s\n",
			h.Timestamp.Format("2006-01-02 15:04:05"),
			h.Command,
			h.Target,
			sizeStr,
		)
	}
	return nil
}
