package main

import (
	"encoding/json"
	"fmt"
	"os"
)

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

	printInfo("pushed %s to slot %q\n", formatSize(int64(len(data))), slot)
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
		printInfo("pulled %s from slot %q (source: %s)\n", formatSize(int64(len(data))), slot, host)
	} else {
		printInfo("pulled %s from slot %q\n", formatSize(int64(len(data))), slot)
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
	var jsonOutput bool
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown flag: %s\nusage: pipeboard slots [--json]", arg)
		}
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
		if jsonOutput {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("No slots found.")
		return nil
	}

	if jsonOutput {
		type jsonSlot struct {
			Name      string `json:"name"`
			Size      int64  `json:"size"`
			SizeHuman string `json:"size_human"`
			CreatedAt string `json:"created_at"`
			Age       string `json:"age"`
			ExpiresAt string `json:"expires_at,omitempty"`
			ExpiresIn string `json:"expires_in,omitempty"`
		}
		jsonSlots := make([]jsonSlot, len(slots))
		for i, s := range slots {
			js := jsonSlot{
				Name:      s.Name,
				Size:      s.Size,
				SizeHuman: formatSize(s.Size),
				CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Age:       formatAge(s.CreatedAt),
			}
			if !s.ExpiresAt.IsZero() {
				js.ExpiresAt = s.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
				js.ExpiresIn = formatTimeUntil(s.ExpiresAt)
			}
			jsonSlots[i] = js
		}
		out, err := json.MarshalIndent(jsonSlots, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	// Check if any slots have expiry
	hasExpiry := false
	for _, s := range slots {
		if !s.ExpiresAt.IsZero() {
			hasExpiry = true
			break
		}
	}

	// Print header
	if hasExpiry {
		fmt.Printf("%-20s  %-10s  %-12s  %-12s\n", "NAME", "SIZE", "AGE", "EXPIRES")
	} else {
		fmt.Printf("%-20s  %-10s  %-12s\n", "NAME", "SIZE", "AGE")
	}

	for _, s := range slots {
		if hasExpiry {
			expires := "-"
			if !s.ExpiresAt.IsZero() {
				expires = formatTimeUntil(s.ExpiresAt)
			}
			fmt.Printf("%-20s  %-10s  %-12s  %-12s\n",
				s.Name,
				formatSize(s.Size),
				formatAge(s.CreatedAt),
				expires,
			)
		} else {
			fmt.Printf("%-20s  %-10s  %-12s\n",
				s.Name,
				formatSize(s.Size),
				formatAge(s.CreatedAt),
			)
		}
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

	printInfo("deleted slot %q\n", slot)
	return nil
}
