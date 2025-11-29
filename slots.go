package main

import (
	"errors"
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
