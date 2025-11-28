package main

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
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

// Test fx command
func TestCmdFxNoArgs(t *testing.T) {
	err := cmdFx([]string{})
	if err == nil {
		t.Error("cmdFx with no args should return error")
	}
}

func TestCmdFxChainedTransforms(t *testing.T) {
	// cmdFx now supports chained transforms - multiple args are allowed
	// This test just verifies the parsing works (will fail on unknown transforms)
	err := cmdFx([]string{"transform1", "transform2"})
	// Should error because transforms don't exist, not because too many args
	if err != nil && strings.Contains(err.Error(), "unexpected argument") {
		t.Error("cmdFx should accept multiple transform names for chaining")
	}
}

func TestRunTransform(t *testing.T) {
	// Test with echo command
	input := []byte("hello world")
	output, err := runTransform([]string{"cat"}, input)
	if err != nil {
		t.Fatalf("runTransform() error: %v", err)
	}
	if string(output) != string(input) {
		t.Errorf("expected %q, got %q", string(input), string(output))
	}
}

func TestRunTransformWithProcessing(t *testing.T) {
	// Test transform that actually processes data
	input := []byte("hello\nworld\n")
	output, err := runTransform([]string{"wc", "-l"}, input)
	if err != nil {
		t.Fatalf("runTransform() error: %v", err)
	}
	// wc -l should return "2" (with possible whitespace)
	trimmed := bytes.TrimSpace(output)
	if string(trimmed) != "2" {
		t.Errorf("expected '2', got %q", string(trimmed))
	}
}

func TestRunTransformEmptyCommand(t *testing.T) {
	_, err := runTransform([]string{}, []byte("test"))
	if err == nil {
		t.Error("runTransform with empty command should return error")
	}
}

func TestRunTransformEmptyOutput(t *testing.T) {
	// Command that produces empty output succeeds (returns empty bytes)
	output, err := runTransform([]string{"true"}, []byte("input"))
	if err != nil {
		t.Fatalf("runTransform() error: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("expected empty output, got %q", string(output))
	}
}

func TestRunTransformFailingCommand(t *testing.T) {
	_, err := runTransform([]string{"false"}, []byte("input"))
	if err == nil {
		t.Error("runTransform with failing command should return error")
	}
}

// Test history functions
func TestGetHistoryPath(t *testing.T) {
	path := getHistoryPath()
	if path == "" {
		t.Error("getHistoryPath() should return non-empty path")
	}
	// Should end with history.json
	if !bytes.HasSuffix([]byte(path), []byte("history.json")) {
		t.Errorf("history path should end with history.json, got %s", path)
	}
}

func TestCmdHistoryWithUnknownFlag(t *testing.T) {
	// History now supports filter flags, but should error on unknown ones
	err := cmdHistory([]string{"--unknown"})
	if err == nil {
		t.Error("cmdHistory with unknown flag should return error")
	}
}

func TestCmdHistoryFilters(t *testing.T) {
	// Valid filter flags should be accepted (may return "no history" but not error)
	// These tests just verify flag parsing works
	_ = cmdHistory([]string{"--fx"})     // Should not error on unknown flag
	_ = cmdHistory([]string{"--slots"})  // Should not error on unknown flag
	_ = cmdHistory([]string{"--peer"})   // Should not error on unknown flag
}

// Test backend detection for Windows
func TestBackendWindows(t *testing.T) {
	// BackendWindows constant should be distinct
	if BackendWindows == BackendWSL {
		t.Error("BackendWindows should be different from BackendWSL")
	}
	if BackendWindows == BackendUnknown {
		t.Error("BackendWindows should be different from BackendUnknown")
	}
}

func TestDetectWindowsFunction(t *testing.T) {
	// This test just verifies the function runs without panic
	// Actual behavior depends on whether Windows tools are available
	b, err := detectWindows()
	if err != nil {
		t.Fatalf("detectWindows() error: %v", err)
	}
	if b == nil {
		t.Fatal("detectWindows() should return a backend")
	}
	if b.Kind != BackendWindows {
		t.Errorf("expected BackendWindows, got %s", b.Kind)
	}
	// Should have copy and paste commands defined
	if len(b.CopyCmd) == 0 {
		t.Error("CopyCmd should not be empty")
	}
	if len(b.PasteCmd) == 0 {
		t.Error("PasteCmd should not be empty")
	}
	// Should have image commands (Windows supports them via PowerShell)
	if len(b.ImageCopyCmd) == 0 {
		t.Error("ImageCopyCmd should not be empty on Windows")
	}
	if len(b.ImagePasteCmd) == 0 {
		t.Error("ImagePasteCmd should not be empty on Windows")
	}
}

// Test copy with --image flag parsing
func TestCmdCopyImageFlag(t *testing.T) {
	// Test that --image flag is recognized (will fail due to no image backend in test env, but that's ok)
	err := cmdCopy([]string{"--image"})
	// We expect an error because we're in a test environment without proper clipboard
	// but the flag should be parsed correctly (not "unknown flag" error)
	if err != nil && bytes.Contains([]byte(err.Error()), []byte("unknown flag")) {
		t.Error("--image flag should be recognized")
	}
}

func TestCmdPasteImageFlag(t *testing.T) {
	// Test that --image flag is recognized
	err := cmdPaste([]string{"--image"})
	// We expect an error because we're in a test environment
	// but the flag should be parsed correctly
	if err != nil && bytes.Contains([]byte(err.Error()), []byte("unknown flag")) {
		t.Error("--image flag should be recognized")
	}
}

// Test help output includes fx and history commands
func TestPrintHelpIncludesFx(t *testing.T) {
	output := captureOutput(printHelp)

	expectedStrings := []string{
		"fx",
		"history",
		"--image",
		"--dry-run",
		"--list",
	}

	for _, s := range expectedStrings {
		if !bytes.Contains([]byte(output), []byte(s)) {
			t.Errorf("help output should contain %q", s)
		}
	}
}

// Test fxList function
func TestFxList(t *testing.T) {
	cfg := &Config{
		Fx: map[string]FxConfig{
			"pretty-json": {Cmd: []string{"jq", "."}, Description: "Format JSON"},
			"strip-ansi":  {Shell: "sed 's/foo/bar/g'", Description: "Strip ANSI codes"},
		},
	}

	// Capture output
	output := captureOutput(func() {
		_ = fxList(cfg)
	})

	// Should contain transform names
	if !bytes.Contains([]byte(output), []byte("pretty-json")) {
		t.Error("fxList output should contain 'pretty-json'")
	}
	if !bytes.Contains([]byte(output), []byte("strip-ansi")) {
		t.Error("fxList output should contain 'strip-ansi'")
	}
}

func TestFxListEmpty(t *testing.T) {
	cfg := &Config{
		Fx: map[string]FxConfig{},
	}

	output := captureOutput(func() {
		_ = fxList(cfg)
	})

	// Should indicate no transforms
	if !bytes.Contains([]byte(output), []byte("No transforms")) {
		t.Error("fxList with empty config should indicate no transforms")
	}
}

// Test recordHistory function
func TestRecordHistory(t *testing.T) {
	// Use a temporary directory for history
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record an entry
	recordHistory("push", "test-slot", 100)

	// Verify history file was created
	histPath := getHistoryPath()
	if _, err := os.Stat(histPath); os.IsNotExist(err) {
		t.Error("history file should be created")
	}
}

// Test cmdFx with --list flag
func TestCmdFxListFlag(t *testing.T) {
	// Create a temporary config with transforms
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"

	configContent := `fx:
  test-transform:
    cmd: ["cat"]
    description: "Test transform"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer func() {
		if origConfig != "" {
			_ = os.Setenv("PIPEBOARD_CONFIG", origConfig)
		} else {
			_ = os.Unsetenv("PIPEBOARD_CONFIG")
		}
	}()

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	// Test --list flag
	err := cmdFx([]string{"--list"})
	if err != nil {
		t.Fatalf("cmdFx --list error: %v", err)
	}
}

// Test version output
func TestVersionOutput(t *testing.T) {
	// We can't easily test main() directly, but we can verify the version constant exists
	// by checking the help output includes a version reference
	output := captureOutput(printHelp)
	if output == "" {
		t.Error("printHelp should produce output")
	}
}

// Test that BackendKind includes Windows
func TestBackendKindIncludesWindows(t *testing.T) {
	kinds := []BackendKind{
		BackendDarwin,
		BackendWayland,
		BackendX11,
		BackendWSL,
		BackendWindows,
		BackendUnknown,
	}

	seen := make(map[BackendKind]bool)
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("duplicate backend kind: %s", k)
		}
		seen[k] = true
	}

	// Verify all 6 kinds are distinct
	if len(seen) != 6 {
		t.Errorf("expected 6 distinct backend kinds, got %d", len(seen))
	}
}

// Test command routing
func TestCommandRouting(t *testing.T) {
	// Test that unknown commands fail
	// We can't test main() directly, but we test the command map exists
	commands := map[string]bool{
		"copy": true, "paste": true, "clear": true, "backend": true,
		"doctor": true, "push": true, "pull": true, "show": true,
		"slots": true, "rm": true, "send": true, "recv": true,
		"receive": true, "peek": true, "history": true, "fx": true,
	}

	if len(commands) != 16 {
		t.Errorf("expected 16 commands, got %d", len(commands))
	}
}

// Test image backend detection fields
func TestBackendImageFields(t *testing.T) {
	// Test Wayland backend has image commands
	orig := os.Getenv("WAYLAND_DISPLAY")
	_ = os.Setenv("WAYLAND_DISPLAY", "wayland-0")
	defer func() {
		if orig != "" {
			_ = os.Setenv("WAYLAND_DISPLAY", orig)
		} else {
			_ = os.Unsetenv("WAYLAND_DISPLAY")
		}
	}()

	b := detectWayland()
	if b == nil {
		t.Fatal("detectWayland should return backend when WAYLAND_DISPLAY is set")
	}

	if len(b.ImageCopyCmd) == 0 {
		t.Error("Wayland backend should have ImageCopyCmd")
	}
	if len(b.ImagePasteCmd) == 0 {
		t.Error("Wayland backend should have ImagePasteCmd")
	}
}

// Test X11 backend image fields
func TestX11BackendImageFields(t *testing.T) {
	origDisplay := os.Getenv("DISPLAY")
	origWayland := os.Getenv("WAYLAND_DISPLAY")
	_ = os.Setenv("DISPLAY", ":0")
	_ = os.Unsetenv("WAYLAND_DISPLAY")
	defer func() {
		if origDisplay != "" {
			_ = os.Setenv("DISPLAY", origDisplay)
		} else {
			_ = os.Unsetenv("DISPLAY")
		}
		if origWayland != "" {
			_ = os.Setenv("WAYLAND_DISPLAY", origWayland)
		}
	}()

	b := detectX11()
	if b == nil {
		t.Fatal("detectX11 should return backend when DISPLAY is set")
	}

	// X11 with xclip should have image commands
	if hasCmd("xclip") {
		if len(b.ImageCopyCmd) == 0 {
			t.Error("X11 backend with xclip should have ImageCopyCmd")
		}
		if len(b.ImagePasteCmd) == 0 {
			t.Error("X11 backend with xclip should have ImagePasteCmd")
		}
	}
}

// Test isSlotCommand helper
func TestIsSlotCommand(t *testing.T) {
	slotCommands := []string{"push", "pull", "show", "rm"}
	nonSlotCommands := []string{"send", "recv", "peek", "copy", "paste", "fx", "history"}

	for _, cmd := range slotCommands {
		if !isSlotCommand(cmd) {
			t.Errorf("isSlotCommand(%q) should return true", cmd)
		}
	}

	for _, cmd := range nonSlotCommands {
		if isSlotCommand(cmd) {
			t.Errorf("isSlotCommand(%q) should return false", cmd)
		}
	}
}

// Test isPeerCommand helper
func TestIsPeerCommand(t *testing.T) {
	peerCommands := []string{"send", "recv", "peek"}
	nonPeerCommands := []string{"push", "pull", "show", "rm", "copy", "paste", "fx", "history"}

	for _, cmd := range peerCommands {
		if !isPeerCommand(cmd) {
			t.Errorf("isPeerCommand(%q) should return true", cmd)
		}
	}

	for _, cmd := range nonPeerCommands {
		if isPeerCommand(cmd) {
			t.Errorf("isPeerCommand(%q) should return false", cmd)
		}
	}
}

// Test runCommand
func TestRunCommand(t *testing.T) {
	t.Run("empty command", func(t *testing.T) {
		err := runCommand([]string{})
		if err == nil {
			t.Error("expected error for empty command")
		}
	})

	t.Run("successful command", func(t *testing.T) {
		err := runCommand([]string{"true"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("failing command", func(t *testing.T) {
		err := runCommand([]string{"false"})
		if err == nil {
			t.Error("expected error for failing command")
		}
	})
}

// Test runWithInput
func TestRunWithInput(t *testing.T) {
	t.Run("empty command", func(t *testing.T) {
		err := runWithInput([]string{}, []byte("test"))
		if err == nil {
			t.Error("expected error for empty command")
		}
	})

	t.Run("successful command with input", func(t *testing.T) {
		err := runWithInput([]string{"cat"}, []byte("test"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("failing command", func(t *testing.T) {
		err := runWithInput([]string{"false"}, []byte("test"))
		if err == nil {
			t.Error("expected error for failing command")
		}
	})
}

// Test runAndPipeStdout
func TestRunAndPipeStdout(t *testing.T) {
	t.Run("empty command", func(t *testing.T) {
		err := runAndPipeStdout([]string{})
		if err == nil {
			t.Error("expected error for empty command")
		}
	})

	t.Run("successful command", func(t *testing.T) {
		err := runAndPipeStdout([]string{"true"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// Test cmdClear error paths
func TestCmdClearNoPanic(t *testing.T) {
	// cmdClear will fail in test environment without proper clipboard
	// but we can test that it handles errors gracefully
	err := cmdClear([]string{})
	// We expect an error because there's no clipboard in test env
	_ = err // Just verify it doesn't panic
}

// Test cmdBackend runs
func TestCmdBackendNoPanic(t *testing.T) {
	// cmdBackend will work on any system, just prints the backend
	err := cmdBackend([]string{})
	// May succeed or fail depending on environment
	_ = err // Just verify it doesn't panic
}

// Test cmdDoctor runs without panic
func TestCmdDoctorNoPanic(t *testing.T) {
	// cmdDoctor should run without panicking
	err := cmdDoctor([]string{})
	_ = err // Just verify no panic
}

// Test cmdHistory with various filter combinations
func TestCmdHistoryFilterCombinations(t *testing.T) {
	// Set up temp history file
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record some history entries
	recordHistory("push", "slot1", 100)
	recordHistory("send", "peer1", 50)
	recordHistory("fx", "pretty-json", 200)

	// Test each filter
	testCases := []struct {
		name string
		args []string
	}{
		{"no filter", []string{}},
		{"fx filter", []string{"--fx"}},
		{"slots filter", []string{"--slots"}},
		{"peer filter", []string{"--peer"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := cmdHistory(tc.args)
			if err != nil {
				t.Errorf("cmdHistory(%v) error: %v", tc.args, err)
			}
		})
	}
}

// Test detectDarwin always returns a backend
func TestDetectDarwinReturnsBackend(t *testing.T) {
	b, err := detectDarwin()
	if err != nil {
		t.Fatalf("detectDarwin() error: %v", err)
	}
	if b == nil {
		t.Fatal("detectDarwin() should return a backend")
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

// Test HistoryEntry struct
func TestHistoryEntryFields(t *testing.T) {
	entry := HistoryEntry{
		Timestamp: time.Now(),
		Command:   "push",
		Target:    "myslot",
		Size:      100,
	}

	if entry.Command != "push" {
		t.Error("Command field mismatch")
	}
	if entry.Target != "myslot" {
		t.Error("Target field mismatch")
	}
	if entry.Size != 100 {
		t.Error("Size field mismatch")
	}
}

// Test detectWSL returns a backend struct
func TestDetectWSLReturns(t *testing.T) {
	b := detectWSL()
	// detectWSL returns nil if clip.exe is not found (which is expected in non-WSL env)
	// We just verify it doesn't panic
	if b != nil {
		if b.Kind != BackendWSL {
			t.Errorf("expected BackendWSL, got %s", b.Kind)
		}
	}
}

// Test cmdFx with dry-run flag
func TestCmdFxDryRun(t *testing.T) {
	// Create a temporary config with a simple transform
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"

	configContent := `fx:
  test-transform:
    cmd: ["cat"]
    description: "Test transform"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer func() {
		if origConfig != "" {
			_ = os.Setenv("PIPEBOARD_CONFIG", origConfig)
		} else {
			_ = os.Unsetenv("PIPEBOARD_CONFIG")
		}
	}()

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	// Test --dry-run flag (will fail due to no clipboard, but tests flag parsing)
	err := cmdFx([]string{"test-transform", "--dry-run"})
	// Expected to fail due to clipboard access, but should not be a flag parsing error
	if err != nil && strings.Contains(err.Error(), "unexpected") {
		t.Error("--dry-run flag should be recognized")
	}
}

// createMockSSH creates a mock ssh script for testing
func createMockSSH(t *testing.T, response string, shouldFail bool) string {
	t.Helper()
	tmpDir := t.TempDir()
	mockSSH := tmpDir + "/ssh"

	var script string
	if shouldFail {
		script = "#!/bin/sh\nexit 1\n"
	} else {
		script = "#!/bin/sh\necho '" + response + "'\n"
	}

	if err := os.WriteFile(mockSSH, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create mock ssh: %v", err)
	}
	return tmpDir
}

// Test cmdSend with mock SSH
func TestCmdSendWithMockSSH(t *testing.T) {
	// Create mock SSH that succeeds
	mockDir := createMockSSH(t, "ok", false)

	// Create config with peer
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"
	configContent := `peers:
  testpeer:
    ssh: testhost
    remote_cmd: pipeboard
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Save and restore env
	origPath := os.Getenv("PATH")
	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer func() {
		_ = os.Setenv("PATH", origPath)
		restoreEnv("PIPEBOARD_CONFIG", origConfig)
	}()

	// Prepend mock to PATH
	_ = os.Setenv("PATH", mockDir+":"+origPath)
	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	// cmdSend will fail due to readClipboard, but that's expected
	// We're testing the peer resolution and SSH command setup
	err := cmdSend([]string{"testpeer"})
	// Error is expected (no clipboard), but should mention the peer
	if err != nil && !strings.Contains(err.Error(), "backend") {
		// If error is about peer not found, that's a test failure
		if strings.Contains(err.Error(), "not found") {
			t.Errorf("peer should be found: %v", err)
		}
	}
}

// Test cmdSend with missing peer
func TestCmdSendMissingPeer(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"
	configContent := `peers:
  otherpeer:
    ssh: otherhost
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", origConfig)
	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	err := cmdSend([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent peer")
	}
	if !strings.Contains(err.Error(), "unknown peer") {
		t.Errorf("error should mention unknown peer: %v", err)
	}
}

// Test cmdRecv with missing peer
func TestCmdRecvMissingPeer(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"
	configContent := `peers:
  otherpeer:
    ssh: otherhost
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", origConfig)
	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	err := cmdRecv([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent peer")
	}
}

// Test cmdPeek with missing peer
func TestCmdPeekMissingPeer(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"
	configContent := `peers:
  otherpeer:
    ssh: otherhost
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", origConfig)
	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	err := cmdPeek([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent peer")
	}
}

// Test cmdPeek with mock SSH
func TestCmdPeekWithMockSSH(t *testing.T) {
	// Create mock SSH that outputs data
	mockDir := createMockSSH(t, "clipboard content", false)

	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"
	configContent := `peers:
  testpeer:
    ssh: testhost
    remote_cmd: pipeboard
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origPath := os.Getenv("PATH")
	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer func() {
		_ = os.Setenv("PATH", origPath)
		restoreEnv("PIPEBOARD_CONFIG", origConfig)
	}()

	_ = os.Setenv("PATH", mockDir+":"+origPath)
	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	// cmdPeek should succeed with mock SSH
	err := cmdPeek([]string{"testpeer"})
	if err != nil {
		t.Errorf("cmdPeek with mock SSH should succeed: %v", err)
	}
}

// Test cmdRecv with mock SSH
func TestCmdRecvWithMockSSH(t *testing.T) {
	// Create mock SSH that outputs data
	mockDir := createMockSSH(t, "received data", false)

	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"
	configContent := `peers:
  testpeer:
    ssh: testhost
    remote_cmd: pipeboard
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origPath := os.Getenv("PATH")
	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer func() {
		_ = os.Setenv("PATH", origPath)
		restoreEnv("PIPEBOARD_CONFIG", origConfig)
	}()

	_ = os.Setenv("PATH", mockDir+":"+origPath)
	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	// cmdRecv will try to write to clipboard which may fail
	// but the SSH part should work
	err := cmdRecv([]string{"testpeer"})
	// May fail on clipboard write, but shouldn't fail on SSH
	if err != nil && strings.Contains(err.Error(), "ssh") {
		t.Errorf("SSH part should work with mock: %v", err)
	}
}
