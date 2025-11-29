package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// Cached backend detection - avoids repeated filesystem/env lookups
var (
	cachedBackend     *Backend
	cachedBackendErr  error
	cachedBackendOnce sync.Once
)

// getBackend returns the cached backend, detecting it on first call.
// This avoids repeated exec.LookPath calls on every clipboard operation.
func getBackend() (*Backend, error) {
	cachedBackendOnce.Do(func() {
		cachedBackend, cachedBackendErr = detectBackend()
	})
	return cachedBackend, cachedBackendErr
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

func detectBackend() (*Backend, error) {
	goos := runtime.GOOS

	switch goos {
	case "darwin":
		return detectDarwin()
	case "linux":
		// Try Wayland, then X11, then WSL-friendly
		// Only use a backend if its required tools are available
		if b := detectWayland(); b != nil && len(b.Missing) == 0 {
			return b, nil
		}
		if b := detectX11(); b != nil && len(b.Missing) == 0 {
			return b, nil
		}
		if b := detectWSL(); b != nil && len(b.Missing) == 0 {
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

// installHint returns a helpful installation hint for the given backend
func installHint(kind BackendKind) string {
	switch kind {
	case BackendWayland:
		return "Install wl-clipboard: sudo apt install wl-clipboard (Debian/Ubuntu) or sudo dnf install wl-clipboard (Fedora)"
	case BackendX11:
		return "Install xclip: sudo apt install xclip (Debian/Ubuntu) or sudo dnf install xclip (Fedora)"
	case BackendDarwin:
		return "pbcopy/pbpaste should be available by default on macOS"
	case BackendWSL, BackendWindows:
		return "Ensure clip.exe and powershell.exe are in your PATH"
	default:
		return "Run 'pipeboard doctor' for more information"
	}
}

// missingToolsError returns a formatted error with installation hints
func missingToolsError(b *Backend) error {
	return fmt.Errorf("backend %s is missing required tools: %s\n       Hint: %s",
		b.Kind, strings.Join(b.Missing, ", "), installHint(b.Kind))
}
