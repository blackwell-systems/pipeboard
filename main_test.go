package main

import (
	"bytes"
	"os"
	"runtime"
	"testing"
)

func TestBackendKindConstants(t *testing.T) {
	// Verify backend kind constants are distinct
	kinds := []BackendKind{
		BackendDarwin,
		BackendWayland,
		BackendX11,
		BackendWSL,
		BackendUnknown,
	}

	seen := make(map[BackendKind]bool)
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("duplicate backend kind: %s", k)
		}
		seen[k] = true
	}
}

func TestDetectBackend(t *testing.T) {
	b, err := detectBackend()

	switch runtime.GOOS {
	case "darwin":
		if err != nil {
			t.Fatalf("detectBackend() error on darwin: %v", err)
		}
		if b.Kind != BackendDarwin {
			t.Errorf("expected BackendDarwin on darwin, got %s", b.Kind)
		}
	case "linux":
		if err != nil {
			t.Fatalf("detectBackend() error on linux: %v", err)
		}
		// On linux, could be Wayland, X11, WSL, or Unknown depending on env
		validKinds := map[BackendKind]bool{
			BackendWayland: true,
			BackendX11:     true,
			BackendWSL:     true,
			BackendUnknown: true,
		}
		if !validKinds[b.Kind] {
			t.Errorf("unexpected backend kind on linux: %s", b.Kind)
		}
	case "windows":
		if err != nil {
			t.Fatalf("detectBackend() error on windows: %v", err)
		}
		// Windows uses WSL-style backend
		if b.Kind != BackendWSL {
			t.Errorf("expected BackendWSL on windows, got %s", b.Kind)
		}
	default:
		// Unsupported OS should return error
		if err == nil {
			t.Errorf("expected error for unsupported OS %s", runtime.GOOS)
		}
	}
}

func TestDetectDarwin(t *testing.T) {
	b, err := detectDarwin()
	if err != nil {
		t.Fatalf("detectDarwin() error: %v", err)
	}
	if b.Kind != BackendDarwin {
		t.Errorf("expected BackendDarwin, got %s", b.Kind)
	}
	if len(b.CopyCmd) == 0 {
		t.Error("CopyCmd should not be empty")
	}
	if len(b.PasteCmd) == 0 {
		t.Error("PasteCmd should not be empty")
	}
}

func TestDetectWayland(t *testing.T) {
	// Test with WAYLAND_DISPLAY unset
	orig := os.Getenv("WAYLAND_DISPLAY")
	_ = os.Unsetenv("WAYLAND_DISPLAY")
	defer func() {
		if orig != "" {
			_ = os.Setenv("WAYLAND_DISPLAY", orig)
		}
	}()

	b := detectWayland()
	if b != nil {
		t.Error("detectWayland() should return nil when WAYLAND_DISPLAY is unset")
	}

	// Test with WAYLAND_DISPLAY set
	_ = os.Setenv("WAYLAND_DISPLAY", "wayland-0")
	b = detectWayland()
	if b == nil {
		t.Fatal("detectWayland() should return backend when WAYLAND_DISPLAY is set")
	}
	if b.Kind != BackendWayland {
		t.Errorf("expected BackendWayland, got %s", b.Kind)
	}
	if b.EnvSource != "WAYLAND_DISPLAY" {
		t.Errorf("expected EnvSource WAYLAND_DISPLAY, got %s", b.EnvSource)
	}
}

func TestDetectX11(t *testing.T) {
	// Test with DISPLAY unset
	orig := os.Getenv("DISPLAY")
	_ = os.Unsetenv("DISPLAY")
	defer func() {
		if orig != "" {
			_ = os.Setenv("DISPLAY", orig)
		}
	}()

	b := detectX11()
	if b != nil {
		t.Error("detectX11() should return nil when DISPLAY is unset")
	}

	// Test with DISPLAY set
	_ = os.Setenv("DISPLAY", ":0")
	b = detectX11()
	if b == nil {
		t.Fatal("detectX11() should return backend when DISPLAY is set")
	}
	if b.Kind != BackendX11 {
		t.Errorf("expected BackendX11, got %s", b.Kind)
	}
	if b.EnvSource != "DISPLAY" {
		t.Errorf("expected EnvSource DISPLAY, got %s", b.EnvSource)
	}
}

func TestHasCmd(t *testing.T) {
	// "go" should exist since we're running go test
	if !hasCmd("go") {
		t.Error("hasCmd(\"go\") should return true")
	}

	// Random non-existent command
	if hasCmd("this-command-definitely-does-not-exist-12345") {
		t.Error("hasCmd should return false for non-existent command")
	}
}

func TestReadInputOrArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "single arg",
			args:     []string{"hello"},
			expected: "hello",
		},
		{
			name:     "multiple args",
			args:     []string{"hello", "world"},
			expected: "hello world",
		},
		{
			name:     "empty args reads stdin",
			args:     []string{},
			expected: "", // stdin is empty in test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.args) > 0 {
				data, err := readInputOrArgs(tt.args)
				if err != nil {
					t.Fatalf("readInputOrArgs() error: %v", err)
				}
				if string(data) != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, string(data))
				}
			}
		})
	}
}

func TestReadInputOrArgsFromStdin(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe and write test data
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	testData := "test input from stdin"
	go func() {
		_, _ = w.WriteString(testData)
		_ = w.Close()
	}()

	os.Stdin = r
	data, err := readInputOrArgs([]string{})
	if err != nil {
		t.Fatalf("readInputOrArgs() error: %v", err)
	}
	if string(data) != testData {
		t.Errorf("expected %q, got %q", testData, string(data))
	}
}

func TestBackendStruct(t *testing.T) {
	b := Backend{
		Kind:      BackendDarwin,
		CopyCmd:   []string{"pbcopy"},
		PasteCmd:  []string{"pbpaste"},
		ClearCmd:  []string{},
		Notes:     "test notes",
		Missing:   []string{},
		EnvSource: "",
	}

	if b.Kind != BackendDarwin {
		t.Errorf("expected Kind BackendDarwin, got %s", b.Kind)
	}
	if len(b.CopyCmd) != 1 || b.CopyCmd[0] != "pbcopy" {
		t.Errorf("unexpected CopyCmd: %v", b.CopyCmd)
	}
}

func TestRunWithInputEmptyCommand(t *testing.T) {
	err := runWithInput([]string{}, []byte("test"))
	if err == nil {
		t.Error("runWithInput with empty command should return error")
	}
}

func TestRunCommandEmptyCommand(t *testing.T) {
	err := runCommand([]string{})
	if err == nil {
		t.Error("runCommand with empty command should return error")
	}
}

func TestRunAndPipeStdoutEmptyCommand(t *testing.T) {
	err := runAndPipeStdout([]string{})
	if err == nil {
		t.Error("runAndPipeStdout with empty command should return error")
	}
}

func TestCmdPasteWithArgs(t *testing.T) {
	err := cmdPaste([]string{"unexpected", "args"})
	if err == nil {
		t.Error("cmdPaste with args should return error")
	}
}

func TestCmdClearWithArgs(t *testing.T) {
	err := cmdClear([]string{"unexpected"})
	if err == nil {
		t.Error("cmdClear with args should return error")
	}
}

func TestCmdBackendWithArgs(t *testing.T) {
	err := cmdBackend([]string{"unexpected"})
	if err == nil {
		t.Error("cmdBackend with args should return error")
	}
}

func TestCmdDoctorWithArgs(t *testing.T) {
	err := cmdDoctor([]string{"unexpected"})
	if err == nil {
		t.Error("cmdDoctor with args should return error")
	}
}

// Integration test helper to capture stdout
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestPrintHelp(t *testing.T) {
	output := captureOutput(printHelp)

	if output == "" {
		t.Error("printHelp should produce output")
	}

	// Check for key elements including v3 commands
	expectedStrings := []string{
		"pipeboard",
		"copy",
		"paste",
		"clear",
		"backend",
		"doctor",
		"push",
		"pull",
		"show",
		"slots",
		"rm",
		"send",
		"recv",
		"peek",
	}

	for _, s := range expectedStrings {
		if !bytes.Contains([]byte(output), []byte(s)) {
			t.Errorf("help output should contain %q", s)
		}
	}
}

func TestCmdPushNoArgs(t *testing.T) {
	err := cmdPush([]string{})
	if err == nil {
		t.Error("cmdPush with no args should return error")
	}
}

func TestCmdPushTooManyArgs(t *testing.T) {
	err := cmdPush([]string{"slot1", "slot2"})
	if err == nil {
		t.Error("cmdPush with too many args should return error")
	}
}

func TestCmdPullNoArgs(t *testing.T) {
	err := cmdPull([]string{})
	if err == nil {
		t.Error("cmdPull with no args should return error")
	}
}

func TestCmdPullTooManyArgs(t *testing.T) {
	err := cmdPull([]string{"slot1", "slot2"})
	if err == nil {
		t.Error("cmdPull with too many args should return error")
	}
}

func TestCmdShowNoArgs(t *testing.T) {
	err := cmdShow([]string{})
	if err == nil {
		t.Error("cmdShow with no args should return error")
	}
}

func TestCmdShowTooManyArgs(t *testing.T) {
	err := cmdShow([]string{"slot1", "slot2"})
	if err == nil {
		t.Error("cmdShow with too many args should return error")
	}
}

func TestCmdSlotsWithArgs(t *testing.T) {
	err := cmdSlots([]string{"unexpected"})
	if err == nil {
		t.Error("cmdSlots with args should return error")
	}
}

func TestCmdRmNoArgs(t *testing.T) {
	err := cmdRm([]string{})
	if err == nil {
		t.Error("cmdRm with no args should return error")
	}
}

func TestCmdRmTooManyArgs(t *testing.T) {
	err := cmdRm([]string{"slot1", "slot2"})
	if err == nil {
		t.Error("cmdRm with too many args should return error")
	}
}

func TestCmdSendNoArgs(t *testing.T) {
	err := cmdSend([]string{})
	if err == nil {
		t.Error("cmdSend with no args should return error")
	}
}

func TestCmdSendTooManyArgs(t *testing.T) {
	err := cmdSend([]string{"peer1", "peer2"})
	if err == nil {
		t.Error("cmdSend with too many args should return error")
	}
}

func TestCmdRecvNoArgs(t *testing.T) {
	err := cmdRecv([]string{})
	if err == nil {
		t.Error("cmdRecv with no args should return error")
	}
}

func TestCmdRecvTooManyArgs(t *testing.T) {
	err := cmdRecv([]string{"peer1", "peer2"})
	if err == nil {
		t.Error("cmdRecv with too many args should return error")
	}
}

func TestCmdPeekNoArgs(t *testing.T) {
	err := cmdPeek([]string{})
	if err == nil {
		t.Error("cmdPeek with no args should return error")
	}
}

func TestCmdPeekTooManyArgs(t *testing.T) {
	err := cmdPeek([]string{"peer1", "peer2"})
	if err == nil {
		t.Error("cmdPeek with too many args should return error")
	}
}
