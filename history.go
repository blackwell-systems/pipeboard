package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Command   string    `json:"command"`
	Target    string    `json:"target"`
	Size      int64     `json:"size,omitempty"`
}

// ClipboardHistoryEntry stores clipboard content snapshots
type ClipboardHistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Hash      string    `json:"hash"`      // SHA256 hash for deduplication
	Preview   string    `json:"preview"`   // First 100 chars
	Size      int64     `json:"size"`
	Content   []byte    `json:"content"`   // Full content (stored separately)
}

const maxHistoryEntries = 50
const maxClipboardHistory = 20
const previewLength = 100

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

func getClipboardHistoryPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "pipeboard", "clipboard_history.json")
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

// recordClipboardHistory saves clipboard content to local history
func recordClipboardHistory(content []byte) {
	path := getClipboardHistoryPath()
	if path == "" {
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// Load existing history
	var history []ClipboardHistoryEntry
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &history)
	}

	// Compute hash
	hashBytes := sha256.Sum256(content)
	hash := hex.EncodeToString(hashBytes[:])

	// Check for duplicate (don't add if same as most recent)
	if len(history) > 0 && history[len(history)-1].Hash == hash {
		return
	}

	// Generate preview
	preview := string(content)
	if len(preview) > previewLength {
		preview = preview[:previewLength] + "..."
	}
	// Clean up preview (remove newlines for display)
	preview = strings.ReplaceAll(preview, "\n", "\\n")
	preview = strings.ReplaceAll(preview, "\r", "")

	// Add new entry
	history = append(history, ClipboardHistoryEntry{
		Timestamp: time.Now(),
		Hash:      hash,
		Preview:   preview,
		Size:      int64(len(content)),
		Content:   content,
	})

	// Trim to max entries
	if len(history) > maxClipboardHistory {
		history = history[len(history)-maxClipboardHistory:]
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
	var filterFx, filterSlots, filterPeer, filterLocal, jsonOutput bool
	for _, arg := range args {
		switch arg {
		case "--fx":
			filterFx = true
		case "--slots":
			filterSlots = true
		case "--peer":
			filterPeer = true
		case "--local":
			filterLocal = true
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown flag: %s\nusage: pipeboard history [--fx] [--slots] [--peer] [--local] [--json]", arg)
		}
	}

	// Local clipboard history mode
	if filterLocal {
		return showClipboardHistory(jsonOutput)
	}

	path := getHistoryPath()
	if path == "" {
		return errors.New("could not determine history path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if jsonOutput {
				fmt.Println("[]")
				return nil
			}
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
		if jsonOutput {
			fmt.Println("[]")
			return nil
		}
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
		if jsonOutput {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("No matching history entries.")
		return nil
	}

	// Reverse for most recent first
	reversed := make([]HistoryEntry, len(filtered))
	for i := 0; i < len(filtered); i++ {
		reversed[i] = filtered[len(filtered)-1-i]
	}

	if jsonOutput {
		out, err := json.MarshalIndent(reversed, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	// Show most recent first (reverse order)
	fmt.Printf("%-20s  %-12s  %-15s  %s\n", "TIME", "COMMAND", "TARGET", "SIZE")
	for _, h := range reversed {
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

func showClipboardHistory(jsonOutput bool) error {
	path := getClipboardHistoryPath()
	if path == "" {
		return errors.New("could not determine clipboard history path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if jsonOutput {
				fmt.Println("[]")
				return nil
			}
			fmt.Println("No clipboard history yet. Use 'pipeboard copy' to record history.")
			return nil
		}
		return err
	}

	var history []ClipboardHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return err
	}

	if len(history) == 0 {
		if jsonOutput {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("No clipboard history yet.")
		return nil
	}

	// Reverse for most recent first
	reversed := make([]ClipboardHistoryEntry, len(history))
	for i := 0; i < len(history); i++ {
		reversed[i] = history[len(history)-1-i]
	}

	if jsonOutput {
		// Output without content for JSON (too large)
		type jsonEntry struct {
			Index     int       `json:"index"`
			Timestamp time.Time `json:"timestamp"`
			Preview   string    `json:"preview"`
			Size      int64     `json:"size"`
		}
		entries := make([]jsonEntry, len(reversed))
		for i, h := range reversed {
			entries[i] = jsonEntry{
				Index:     i + 1,
				Timestamp: h.Timestamp,
				Preview:   h.Preview,
				Size:      h.Size,
			}
		}
		out, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("%-5s  %-20s  %-10s  %s\n", "INDEX", "TIME", "SIZE", "PREVIEW")
	for i, h := range reversed {
		fmt.Printf("%-5d  %-20s  %-10s  %s\n",
			i+1,
			h.Timestamp.Format("2006-01-02 15:04:05"),
			formatSize(h.Size),
			truncateString(h.Preview, 50),
		)
	}
	fmt.Println()
	fmt.Println("Use 'pipeboard recall <index>' to restore an entry to clipboard.")
	return nil
}

func cmdRecall(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: pipeboard recall <index>")
	}

	// Parse index
	var index int
	if _, err := fmt.Sscanf(args[0], "%d", &index); err != nil {
		return fmt.Errorf("invalid index: %s", args[0])
	}

	if index < 1 {
		return fmt.Errorf("index must be >= 1")
	}

	path := getClipboardHistoryPath()
	if path == "" {
		return errors.New("could not determine clipboard history path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("no clipboard history yet")
		}
		return err
	}

	var history []ClipboardHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return err
	}

	if len(history) == 0 {
		return errors.New("no clipboard history yet")
	}

	// Index is 1-based, most recent first
	reversedIndex := len(history) - index
	if reversedIndex < 0 || reversedIndex >= len(history) {
		return fmt.Errorf("index %d out of range (1-%d)", index, len(history))
	}

	entry := history[reversedIndex]

	if err := writeClipboard(entry.Content); err != nil {
		return err
	}

	fmt.Printf("restored entry %d (%s) to clipboard\n", index, formatSize(entry.Size))
	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func isSlotCommand(cmd string) bool {
	return cmd == "push" || cmd == "pull" || cmd == "show" || cmd == "rm"
}

func isPeerCommand(cmd string) bool {
	return cmd == "send" || cmd == "recv" || cmd == "peek" ||
		cmd == "watch:send" || cmd == "watch:recv"
}
