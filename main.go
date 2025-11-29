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

// ANSI color codes for terminal output
const (
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// useColor returns true if color output should be used
func useColor() bool {
	// Disable color if NO_COLOR is set (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Disable color if not a terminal
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	// Check if stdout is a terminal (basic heuristic)
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// commandHelp provides per-command help text
var commandHelp = map[string]string{
	"copy": `Usage: pipeboard copy [text] [--image]

Copy text or image to clipboard.

Options:
  --image, -i    Copy PNG image from stdin instead of text

Examples:
  echo "hello" | pipeboard copy     Copy text from stdin
  pipeboard copy "hello world"      Copy provided text
  cat image.png | pipeboard copy --image`,

	"paste": `Usage: pipeboard paste [--image]

Paste clipboard contents to stdout.

Options:
  --image, -i    Paste clipboard image as PNG

Examples:
  pipeboard paste                   Print clipboard text
  pipeboard paste | jq .            Pipe to other commands
  pipeboard paste --image > out.png`,

	"clear": `Usage: pipeboard clear

Clear the clipboard contents (best-effort, may not work on all platforms).`,

	"backend": `Usage: pipeboard backend

Show the detected clipboard backend for your platform.
Useful for debugging clipboard issues.`,

	"doctor": `Usage: pipeboard doctor

Run environment checks to verify clipboard tools are available.
Shows detected backend, available commands, and any issues.`,

	"push": `Usage: pipeboard push <name>

Push current clipboard contents to a remote slot.

Arguments:
  name    Slot name (e.g., "work", "snippet", "tmp")

Examples:
  pipeboard push work               Push to "work" slot
  pipeboard push kube && ssh server "pipeboard pull kube"`,

	"pull": `Usage: pipeboard pull <name>

Pull a remote slot into the local clipboard.

Arguments:
  name    Slot name to pull

Examples:
  pipeboard pull work               Pull "work" slot to clipboard`,

	"show": `Usage: pipeboard show <name>

Print remote slot contents to stdout without modifying local clipboard.

Arguments:
  name    Slot name to show

Examples:
  pipeboard show work               Print slot contents
  pipeboard show work | jq .        Pipe to other commands`,

	"slots": `Usage: pipeboard slots

List all remote slots with size and age.`,

	"rm": `Usage: pipeboard rm <name>

Delete a remote slot.

Arguments:
  name    Slot name to delete`,

	"send": `Usage: pipeboard send [peer]

Send local clipboard directly to a peer's clipboard via SSH.

Arguments:
  peer    Peer name from config (optional, uses defaults.peer if omitted)

Examples:
  pipeboard send                    Send to default peer
  pipeboard send devbox             Send to "devbox" peer`,

	"recv": `Usage: pipeboard recv [peer]

Receive peer's clipboard into local clipboard via SSH.

Arguments:
  peer    Peer name from config (optional, uses defaults.peer if omitted)`,

	"peek": `Usage: pipeboard peek [peer]

Print peer's clipboard to stdout without modifying local clipboard.

Arguments:
  peer    Peer name from config (optional, uses defaults.peer if omitted)`,

	"history": `Usage: pipeboard history [--fx] [--slots] [--peer]

Show recent clipboard operations.

Options:
  --fx       Filter to fx transforms only
  --slots    Filter to push/pull/show/rm only
  --peer     Filter to send/recv/peek only

Examples:
  pipeboard history                 Show all history
  pipeboard history --fx            Show only transforms`,

	"fx": `Usage: pipeboard fx <name> [name2...] [--dry-run] [--list]

Run transforms on clipboard contents.

Options:
  --dry-run    Preview output without modifying clipboard
  --list       List available transforms from config

Examples:
  pipeboard fx pretty-json              Format JSON in clipboard
  pipeboard fx strip-ansi pretty-json   Chain multiple transforms
  pipeboard fx uppercase --dry-run      Preview without changing clipboard
  pipeboard fx --list                   Show available transforms`,
}

// stdinHasData returns true if stdin is a pipe (not a terminal)
func stdinHasData() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is a pipe or has data
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// hasHelpFlag checks if args contain -h or --help
func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

// printCommandHelp prints help for a specific command
func printCommandHelp(cmd string) {
	if help, ok := commandHelp[cmd]; ok {
		fmt.Println(help)
	} else {
		printHelp()
	}
}

type BackendKind string

const (
	BackendDarwin  BackendKind = "darwin-pasteboard"
	BackendWayland BackendKind = "wayland-wl-copy"
	BackendX11     BackendKind = "x11-xclip"
	BackendWSL     BackendKind = "wsl-clip"
	BackendWindows BackendKind = "windows-clip"
	BackendUnknown BackendKind = "unknown"
)

type Backend struct {
	Kind          BackendKind
	CopyCmd       []string
	PasteCmd      []string
	ClearCmd      []string // if empty, use CopyCmd with empty stdin
	ImageCopyCmd  []string // for copying images (PNG)
	ImagePasteCmd []string // for pasting images (PNG)
	Notes         string
	Missing       []string
	EnvSource     string
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		// Check if stdin has data (piped input) - default to copy
		if stdinHasData() {
			if err := cmdCopy([]string{}); err != nil {
				fatal(err)
			}
			return
		}
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
		"fx":      cmdFx,
	}

	if fn, ok := commands[cmd]; ok {
		// Check for --help flag on any command
		if hasHelpFlag(rest) {
			printCommandHelp(cmd)
			return
		}
		if err := fn(rest); err != nil {
			fatal(err)
		}
		return
	}

	switch cmd {
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		fmt.Println("pipeboard v0.4.0")
	default:
		if useColor() {
			fmt.Fprintf(os.Stderr, "%sUnknown command: %s%s\n\n", colorRed, cmd, colorReset)
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		}
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`pipeboard - the programmable clipboard router for terminals

Usage:
  pipeboard <command> [args...]
  <stdin> | pipeboard              Piped input defaults to copy

Local clipboard:
  copy [text]          Copy stdin or provided text to clipboard
  copy --image         Copy PNG image from stdin to clipboard
  paste                Paste clipboard contents to stdout
  paste --image        Paste clipboard image as PNG to stdout
  clear                Clear clipboard (best-effort)
  backend              Show detected clipboard backend
  doctor               Run environment checks

Transforms (programmable clipboard pipelines):
  fx <name> [name2...] Run transform(s) on clipboard (chained, in-place)
  fx <name> --dry-run  Preview output without modifying clipboard
  fx --list            List available transforms

  Chaining: pipeboard fx strip-ansi pretty-json
  Safety: clipboard unchanged if any transform fails

Direct peer-to-peer (SSH):
  send [peer]          Send local clipboard to peer's clipboard
  recv [peer]          Receive peer's clipboard into local clipboard
  peek [peer]          Print peer's clipboard to stdout (no local change)
                       (peer defaults to 'defaults.peer' in config)

Remote slots (S3 or local backend):
  push <name>          Push clipboard to remote slot
  pull <name>          Pull remote slot into clipboard
  show <name>          Print remote slot to stdout
  slots                List remote slots
  rm <name>            Delete remote slot

History:
  history              Show recent operations (most recent first)
  history --fx         Filter to fx transforms only
  history --slots      Filter to push/pull/show/rm only
  history --peer       Filter to send/recv/peek only

Other:
  <command> --help     Show help for a specific command
  help                 Show this help
  version              Show version

Config: ~/.config/pipeboard/config.yaml

  defaults:
    peer: dev              # default peer for send/recv/peek

  peers:
    dev:
      ssh: devbox

  fx:                      # clipboard transforms
    pretty-json:
      cmd: ["jq", "."]
      description: "Format JSON"
    strip-ansi:
      shell: "sed 's/\\x1b\\[[0-9;]*m//g'"

  sync:
    backend: local         # or "s3" for cloud sync
    encryption: aes256     # client-side encryption (optional)
    passphrase: secret     # encryption passphrase
    ttl_days: 30           # auto-expire slots (optional)
    # For S3 backend:
    # s3:
    #   bucket: my-bucket
    #   region: us-west-2

Examples:
  echo "hello" | pipeboard             # implicit copy
  pipeboard paste | jq .
  pipeboard fx pretty-json           # format JSON in clipboard
  pipeboard fx strip-ansi --dry-run  # preview transform
  pipeboard send                     # uses default peer
  pipeboard send dev
  pipeboard push kube && ssh server "pipeboard pull kube"
  cat screenshot.png | pipeboard copy --image
  pipeboard paste --image > clipboard.png`)
}

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

	b, err := detectBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return fmt.Errorf("backend %s is missing required tools: %s", b.Kind, strings.Join(b.Missing, ", "))
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

	b, err := detectBackend()
	if err != nil {
		return err
	}
	if len(b.Missing) > 0 {
		return fmt.Errorf("backend %s is missing required tools: %s", b.Kind, strings.Join(b.Missing, ", "))
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

	// Image commands use osascript for reading/writing PNG from clipboard
	// Copy: read PNG from stdin and set as clipboard image
	// Paste: get clipboard image as PNG to stdout
	imageCopyCmd := []string{}
	imagePasteCmd := []string{}

	if hasCmd("osascript") {
		// For paste: use osascript to write clipboard image to temp file, then cat it
		// This is complex, so we use a helper approach
		imagePasteCmd = []string{"osascript", "-e", `
			use framework "AppKit"
			use scripting additions
			set pb to current application's NSPasteboard's generalPasteboard()
			set imgData to pb's dataForType:(current application's NSPasteboardTypePNG)
			if imgData is missing value then
				error "No image on clipboard"
			end if
			set rawData to (current application's NSString's alloc()'s initWithData:imgData encoding:(current application's NSUTF8StringEncoding))
			return (imgData's base64EncodedStringWithOptions:0) as text
		`}
		// Note: Darwin image paste outputs base64, needs wrapper
	}

	// Check for pngpaste (brew install pngpaste) - simpler alternative
	if hasCmd("pngpaste") {
		imagePasteCmd = []string{"pngpaste", "-"}
	}

	// Check for impbcopy (for copying images)
	if hasCmd("impbcopy") {
		imageCopyCmd = []string{"impbcopy", "-"}
	}

	return &Backend{
		Kind:          BackendDarwin,
		CopyCmd:       []string{"pbcopy"},
		PasteCmd:      []string{"pbpaste"},
		ImageCopyCmd:  imageCopyCmd,
		ImagePasteCmd: imagePasteCmd,
		Missing:       missing,
		Notes:         "For image support, install pngpaste and impbcopy (brew install pngpaste impbcopy)",
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
		Kind:          BackendWayland,
		CopyCmd:       []string{"wl-copy"},
		PasteCmd:      []string{"wl-paste"},
		ClearCmd:      []string{"wl-copy", "--clear"},
		ImageCopyCmd:  []string{"wl-copy", "--type", "image/png"},
		ImagePasteCmd: []string{"wl-paste", "--type", "image/png"},
		Missing:       missing,
		EnvSource:     "WAYLAND_DISPLAY",
	}
}

func detectX11() *Backend {
	if os.Getenv("DISPLAY") == "" {
		return nil
	}
	missing := []string{}
	copyCmd := []string{"xclip", "-selection", "clipboard"}
	pasteCmd := []string{"xclip", "-selection", "clipboard", "-o"}
	imageCopyCmd := []string{"xclip", "-selection", "clipboard", "-t", "image/png"}
	imagePasteCmd := []string{"xclip", "-selection", "clipboard", "-t", "image/png", "-o"}

	if !hasCmd("xclip") {
		if hasCmd("xsel") {
			copyCmd = []string{"xsel", "--clipboard", "--input"}
			pasteCmd = []string{"xsel", "--clipboard", "--output"}
			// xsel doesn't support images well, clear image commands
			imageCopyCmd = nil
			imagePasteCmd = nil
		} else {
			missing = append(missing, "xclip/xsel")
		}
	}

	return &Backend{
		Kind:          BackendX11,
		CopyCmd:       copyCmd,
		PasteCmd:      pasteCmd,
		ImageCopyCmd:  imageCopyCmd,
		ImagePasteCmd: imagePasteCmd,
		Missing:       missing,
		EnvSource:     "DISPLAY",
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

	// Image support via PowerShell (limited - paste outputs base64)
	imagePasteCmd := []string{
		"powershell.exe", "-NoProfile", "-Command",
		"$img = Get-Clipboard -Format Image; if ($img) { $ms = New-Object System.IO.MemoryStream; $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png); [Convert]::ToBase64String($ms.ToArray()) } else { throw 'No image on clipboard' }",
	}

	return &Backend{
		Kind:          BackendWSL,
		CopyCmd:       []string{"clip.exe"},
		PasteCmd:      []string{"powershell.exe", "-NoProfile", "-Command", "Get-Clipboard"},
		ImagePasteCmd: imagePasteCmd,
		Missing:       missing,
		Notes:         "WSL detection based on clip.exe in PATH. Image copy not supported.",
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

	// Determine PowerShell executable
	psCmd := "powershell.exe"
	if !hasCmd("powershell.exe") && hasCmd("powershell") {
		psCmd = "powershell"
	}

	// Copy: use clip.exe for text
	copyCmd := []string{"clip.exe"}
	if !hasCmd("clip.exe") && hasCmd("clip") {
		copyCmd = []string{"clip"}
	}

	// Paste: use PowerShell Get-Clipboard
	pasteCmd := []string{psCmd, "-NoProfile", "-Command", "Get-Clipboard"}

	// Image copy via PowerShell - read PNG from stdin and set to clipboard
	imageCopyCmd := []string{
		psCmd, "-NoProfile", "-Command",
		"Add-Type -AssemblyName System.Windows.Forms; $input | Set-Content -Path $env:TEMP\\pb_img.png -Encoding Byte; [System.Windows.Forms.Clipboard]::SetImage([System.Drawing.Image]::FromFile(\"$env:TEMP\\pb_img.png\")); Remove-Item $env:TEMP\\pb_img.png",
	}

	// Image paste via PowerShell - get clipboard image as PNG bytes to stdout
	imagePasteCmd := []string{
		psCmd, "-NoProfile", "-Command",
		"Add-Type -AssemblyName System.Windows.Forms; $img = [System.Windows.Forms.Clipboard]::GetImage(); if ($img) { $ms = New-Object System.IO.MemoryStream; $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png); [Console]::OpenStandardOutput().Write($ms.ToArray(), 0, $ms.Length) } else { throw 'No image on clipboard' }",
	}

	return &Backend{
		Kind:          BackendWindows,
		CopyCmd:       copyCmd,
		PasteCmd:      pasteCmd,
		ImageCopyCmd:  imageCopyCmd,
		ImagePasteCmd: imagePasteCmd,
		Missing:       missing,
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
	if useColor() {
		fmt.Fprintf(os.Stderr, "%spipeboard: %v%s\n", colorRed, err, colorReset)
	} else {
		fmt.Fprintf(os.Stderr, "pipeboard: %v\n", err)
	}
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
	// Parse filter flags
	var filterFx, filterSlots, filterPeer bool
	for _, arg := range args {
		switch arg {
		case "--fx":
			filterFx = true
		case "--slots":
			filterSlots = true
		case "--peer":
			filterPeer = true
		default:
			return fmt.Errorf("unknown flag: %s\nusage: pipeboard history [--fx] [--slots] [--peer]", arg)
		}
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

	// Filter history if requested
	var filtered []HistoryEntry
	for _, h := range history {
		if filterFx && !strings.HasPrefix(h.Command, "fx:") {
			continue
		}
		if filterSlots && !isSlotCommand(h.Command) {
			continue
		}
		if filterPeer && !isPeerCommand(h.Command) {
			continue
		}
		filtered = append(filtered, h)
	}

	if len(filtered) == 0 {
		fmt.Println("No matching history entries.")
		return nil
	}

	// Show most recent first (reverse order)
	fmt.Printf("%-20s  %-12s  %-15s  %s\n", "TIME", "COMMAND", "TARGET", "SIZE")
	for i := len(filtered) - 1; i >= 0; i-- {
		h := filtered[i]
		sizeStr := ""
		if h.Size > 0 {
			sizeStr = formatSize(h.Size)
		}
		fmt.Printf("%-20s  %-12s  %-15s  %s\n",
			h.Timestamp.Format("2006-01-02 15:04:05"),
			h.Command,
			h.Target,
			sizeStr,
		)
	}
	return nil
}

func isSlotCommand(cmd string) bool {
	return cmd == "push" || cmd == "pull" || cmd == "show" || cmd == "rm"
}

func isPeerCommand(cmd string) bool {
	return cmd == "send" || cmd == "recv" || cmd == "peek"
}

// cmdFx runs a user-defined clipboard transform (supports chaining)
func cmdFx(args []string) error {
	// Parse flags and collect transform names
	var dryRun bool
	var listMode bool
	var fxNames []string

	for _, arg := range args {
		switch arg {
		case "--list", "-l":
			listMode = true
		case "--dry-run", "-n":
			dryRun = true
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown flag: %s", arg)
			}
			fxNames = append(fxNames, arg)
		}
	}

	cfg, err := loadConfigForFx()
	if err != nil {
		return err
	}

	// List mode
	if listMode {
		return fxList(cfg)
	}

	// Require at least one transform name
	if len(fxNames) == 0 {
		return fmt.Errorf("usage: pipeboard fx <name> [name2...] [--dry-run]\n       pipeboard fx --list")
	}

	// Validate all transforms exist before reading clipboard
	var transforms []FxConfig
	for _, name := range fxNames {
		fx, err := cfg.getFx(name)
		if err != nil {
			return err
		}
		transforms = append(transforms, fx)
	}

	// Read clipboard
	data, err := readClipboard()
	if err != nil {
		return fmt.Errorf("reading clipboard: %w", err)
	}
	originalSize := len(data)

	// Run transforms in order, feeding output → input
	// If any step fails, abort without modifying clipboard
	result := data
	for i, fx := range transforms {
		cmdArgs := fx.getCommand()
		result, err = runTransform(cmdArgs, result)
		if err != nil {
			return fmt.Errorf("transform %q (step %d) failed: %w; clipboard unchanged", fxNames[i], i+1, err)
		}
		// Check for empty output
		if len(result) == 0 {
			return fmt.Errorf("transform %q (step %d) produced empty output; clipboard unchanged", fxNames[i], i+1)
		}
	}

	// Dry run mode - print result to stdout, never touch clipboard
	if dryRun {
		_, err = os.Stdout.Write(result)
		return err
	}

	// Write result back to clipboard
	if err := writeClipboard(result); err != nil {
		return fmt.Errorf("writing clipboard: %w", err)
	}

	// Report what happened
	chainDesc := strings.Join(fxNames, " → ")
	fmt.Printf("fx %s: %s → %s\n", chainDesc, formatSize(int64(originalSize)), formatSize(int64(len(result))))
	recordHistory("fx:"+chainDesc, "", int64(len(result)))
	return nil
}

// fxList prints available transforms
func fxList(cfg *Config) error {
	if len(cfg.Fx) == 0 {
		fmt.Println("No transforms defined.")
		fmt.Println("\nAdd transforms to your config:")
		fmt.Println("  fx:")
		fmt.Println("    pretty-json:")
		fmt.Println("      cmd: [\"jq\", \".\"]")
		fmt.Println("      description: \"Format JSON\"")
		return nil
	}

	fmt.Printf("%-20s  %s\n", "NAME", "DESCRIPTION")
	for name, fx := range cfg.Fx {
		desc := fx.Description
		if desc == "" {
			if fx.Shell != "" {
				desc = fmt.Sprintf("sh -c %q", fx.Shell)
			} else if len(fx.Cmd) > 0 {
				desc = strings.Join(fx.Cmd, " ")
			}
			// Truncate long descriptions
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
		}
		fmt.Printf("%-20s  %s\n", name, desc)
	}
	return nil
}

// runTransform executes a transform command with input data
func runTransform(cmdArgs []string, input []byte) ([]byte, error) {
	if len(cmdArgs) == 0 {
		return nil, errors.New("no command specified")
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = bytes.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Include stderr in error message for debugging
		errMsg := stderr.String()
		if errMsg != "" {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(errMsg))
		}
		return nil, err
	}

	return stdout.Bytes(), nil
}
