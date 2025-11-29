package main

import (
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
