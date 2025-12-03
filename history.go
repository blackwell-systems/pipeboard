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
	Hash      string    `json:"hash"`              // SHA256 hash for deduplication
	Preview   string    `json:"preview"`           // First 100 chars (may be encrypted preview if encryption enabled)
	Size      int64     `json:"size"`
	Content   []byte    `json:"content"`           // Full content (may be encrypted)
	Encrypted bool      `json:"encrypted,omitempty"` // true if content is encrypted
}

const maxHistoryEntries = 50
const defaultClipboardHistoryLimit = 20
const previewLength = 100

// getClipboardHistoryLimit returns the configured history limit or default
func getClipboardHistoryLimit() int {
	cfg, err := loadConfig()
	if err != nil || cfg.History == nil || cfg.History.Limit <= 0 {
		return defaultClipboardHistoryLimit
	}
	return cfg.History.Limit
}

// getHistoryConfig returns the full history configuration
func getHistoryConfig() *HistoryConfig {
	cfg, err := loadConfig()
	if err != nil || cfg.History == nil {
		return &HistoryConfig{Limit: defaultClipboardHistoryLimit}
	}
	return cfg.History
}

// applyHistoryTTL removes entries older than TTL days
func applyHistoryTTL(history []ClipboardHistoryEntry, ttlDays int) []ClipboardHistoryEntry {
	if ttlDays <= 0 {
		return history
	}
	cutoff := time.Now().AddDate(0, 0, -ttlDays)
	var filtered []ClipboardHistoryEntry
	for _, h := range history {
		if h.Timestamp.After(cutoff) {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

// isDuplicateInHistory checks if content hash exists anywhere in history
func isDuplicateInHistory(history []ClipboardHistoryEntry, hash string) bool {
	for _, h := range history {
		if h.Hash == hash {
			return true
		}
	}
	return false
}

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

// getHistoryEncryptionConfig returns encryption settings for clipboard history
func getHistoryEncryptionConfig() (enabled bool, passphrase string) {
	cfg, err := loadConfig()
	if err != nil {
		return false, ""
	}
	// Use the same encryption settings as sync for consistency
	if cfg.Sync.Encryption == "aes256" && cfg.Sync.Passphrase != "" {
		return true, cfg.Sync.Passphrase
	}
	return false, ""
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

	// Get history configuration
	histCfg := getHistoryConfig()

	// Load existing history
	var history []ClipboardHistoryEntry
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &history)
	}

	// Apply TTL cleanup first
	if histCfg.TTLDays > 0 {
		history = applyHistoryTTL(history, histCfg.TTLDays)
	}

	// Compute hash (always on plaintext for deduplication)
	hashBytes := sha256.Sum256(content)
	hash := hex.EncodeToString(hashBytes[:])

	// Check for duplicates
	if histCfg.NoDuplicates {
		// Full duplicate detection: check all entries
		if isDuplicateInHistory(history, hash) {
			return
		}
	} else {
		// Default: only check most recent entry
		if len(history) > 0 && history[len(history)-1].Hash == hash {
			return
		}
	}

	// Generate preview
	preview := string(content)
	if len(preview) > previewLength {
		preview = preview[:previewLength] + "..."
	}
	// Clean up preview (remove newlines for display)
	preview = strings.ReplaceAll(preview, "\n", "\\n")
	preview = strings.ReplaceAll(preview, "\r", "")

	// Check if encryption is enabled
	encEnabled, passphrase := getHistoryEncryptionConfig()
	storeContent := content
	encrypted := false

	if encEnabled && passphrase != "" {
		encData, err := encrypt(content, passphrase)
		if err == nil {
			storeContent = encData
			encrypted = true
			// Also encrypt the preview for privacy
			encPreview, err := encrypt([]byte(preview), passphrase)
			if err == nil {
				preview = hex.EncodeToString(encPreview)
			}
		}
	}

	// Add new entry
	history = append(history, ClipboardHistoryEntry{
		Timestamp: time.Now(),
		Hash:      hash,
		Preview:   preview,
		Size:      int64(len(content)),
		Content:   storeContent,
		Encrypted: encrypted,
	})

	// Trim to max entries
	limit := histCfg.Limit
	if limit <= 0 {
		limit = defaultClipboardHistoryLimit
	}
	if len(history) > limit {
		history = history[len(history)-limit:]
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
	var searchQuery string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--fx":
			filterFx = true
		case arg == "--slots":
			filterSlots = true
		case arg == "--peer":
			filterPeer = true
		case arg == "--local":
			filterLocal = true
		case arg == "--json":
			jsonOutput = true
		case arg == "--search" || arg == "-s":
			if i+1 >= len(args) {
				return fmt.Errorf("--search requires a query argument")
			}
			i++
			searchQuery = args[i]
		case strings.HasPrefix(arg, "--search="):
			searchQuery = strings.TrimPrefix(arg, "--search=")
		case strings.HasPrefix(arg, "-s="):
			searchQuery = strings.TrimPrefix(arg, "-s=")
		default:
			return fmt.Errorf("unknown flag: %s\nusage: pipeboard history [--fx] [--slots] [--peer] [--local] [--search <query>] [--json]", arg)
		}
	}

	// Local clipboard history mode
	if filterLocal {
		return showClipboardHistory(jsonOutput, searchQuery)
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

func showClipboardHistory(jsonOutput bool, searchQuery string) error {
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

	// Apply TTL cleanup on read
	histCfg := getHistoryConfig()
	if histCfg.TTLDays > 0 {
		history = applyHistoryTTL(history, histCfg.TTLDays)
	}

	if len(history) == 0 {
		if jsonOutput {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("No clipboard history yet.")
		return nil
	}

	// Get encryption config for decryption
	_, passphrase := getHistoryEncryptionConfig()

	// Decrypt entries for display/search if needed
	decryptedHistory := make([]ClipboardHistoryEntry, len(history))
	for i, h := range history {
		decryptedHistory[i] = h
		if h.Encrypted && passphrase != "" {
			// Decrypt content
			decContent, err := decrypt(h.Content, passphrase)
			if err == nil {
				decryptedHistory[i].Content = decContent
			}
			// Decrypt preview (stored as hex)
			encPreviewBytes, err := hex.DecodeString(h.Preview)
			if err == nil {
				decPreview, err := decrypt(encPreviewBytes, passphrase)
				if err == nil {
					decryptedHistory[i].Preview = string(decPreview)
				}
			}
		}
	}
	history = decryptedHistory

	// Filter by search query if provided
	if searchQuery != "" {
		searchLower := strings.ToLower(searchQuery)
		var filtered []ClipboardHistoryEntry
		for _, h := range history {
			// Search in both preview and full content
			if strings.Contains(strings.ToLower(h.Preview), searchLower) ||
				strings.Contains(strings.ToLower(string(h.Content)), searchLower) {
				filtered = append(filtered, h)
			}
		}
		history = filtered
		if len(history) == 0 {
			if jsonOutput {
				fmt.Println("[]")
				return nil
			}
			fmt.Printf("No clipboard history entries matching %q.\n", searchQuery)
			return nil
		}
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
	content := entry.Content

	// Decrypt if needed
	if entry.Encrypted {
		_, passphrase := getHistoryEncryptionConfig()
		if passphrase == "" {
			return fmt.Errorf("entry is encrypted but no passphrase configured in sync settings")
		}
		decContent, err := decrypt(content, passphrase)
		if err != nil {
			return fmt.Errorf("decrypting entry: %w", err)
		}
		content = decContent
	}

	if err := writeClipboard(content); err != nil {
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
