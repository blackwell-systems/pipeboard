package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultWatchInterval = 500 * time.Millisecond
	minWatchInterval     = 100 * time.Millisecond
)

func cmdWatch(args []string) error {
	cfg, err := loadConfigForPeers()
	if err != nil {
		return err
	}

	var peerName string
	if len(args) == 0 {
		peerName, err = cfg.getDefaultPeer()
		if err != nil {
			return fmt.Errorf("usage: pipeboard watch [peer]\n%w", err)
		}
	} else if len(args) == 1 {
		peerName = args[0]
	} else {
		return fmt.Errorf("usage: pipeboard watch [peer]")
	}

	peer, err := cfg.getPeer(peerName)
	if err != nil {
		return err
	}

	fmt.Printf("Watching clipboard with peer %q (%s)\n", peerName, peer.SSH)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	return watchLoop(peerName, peer)
}

func watchLoop(peerName string, peer PeerConfig) error {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Track last known clipboard states
	var lastLocalHash [32]byte
	var lastRemoteHash [32]byte

	// Initialize with current states
	localData, err := readClipboard()
	if err == nil {
		lastLocalHash = sha256.Sum256(localData)
	}

	remoteData, err := readRemoteClipboard(peer)
	if err == nil {
		lastRemoteHash = sha256.Sum256(remoteData)
	}

	ticker := time.NewTicker(defaultWatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nStopping watch...")
			return nil
		case <-ticker.C:
			// Check local clipboard
			localData, err := readClipboard()
			if err != nil {
				continue // Skip this iteration on error
			}
			localHash := sha256.Sum256(localData)

			// Check if local clipboard changed
			if localHash != lastLocalHash && localHash != lastRemoteHash {
				// Local changed, send to peer
				if err := sendToRemote(peer, localData); err != nil {
					fmt.Fprintf(os.Stderr, "watch: failed to send: %v\n", err)
				} else {
					fmt.Printf("→ sent %s to %s\n", formatSize(int64(len(localData))), peerName)
					lastLocalHash = localHash
					lastRemoteHash = localHash // Prevent echo
					recordHistory("watch:send", peerName, int64(len(localData)))
				}
				continue
			}

			// Check remote clipboard
			remoteData, err := readRemoteClipboard(peer)
			if err != nil {
				continue // Skip this iteration on error
			}
			remoteHash := sha256.Sum256(remoteData)

			// Check if remote clipboard changed
			if remoteHash != lastRemoteHash && remoteHash != lastLocalHash {
				// Remote changed, copy to local
				if err := writeClipboard(remoteData); err != nil {
					fmt.Fprintf(os.Stderr, "watch: failed to receive: %v\n", err)
				} else {
					fmt.Printf("← received %s from %s\n", formatSize(int64(len(remoteData))), peerName)
					lastRemoteHash = remoteHash
					lastLocalHash = remoteHash // Prevent echo
					recordHistory("watch:recv", peerName, int64(len(remoteData)))
				}
			}

			lastLocalHash = localHash
			lastRemoteHash = remoteHash
		}
	}
}

// readRemoteClipboard reads clipboard contents from a peer via SSH
func readRemoteClipboard(peer PeerConfig) ([]byte, error) {
	var out bytes.Buffer
	cmd := exec.Command("ssh", peer.SSH, peer.RemoteCmd, "paste")
	cmd.Stdin = nil
	cmd.Stdout = &out
	cmd.Stderr = nil // Suppress errors for polling

	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// sendToRemote sends data to a peer's clipboard via SSH
func sendToRemote(peer PeerConfig, data []byte) error {
	cmd := exec.Command("ssh", peer.SSH, peer.RemoteCmd, "copy")
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}
