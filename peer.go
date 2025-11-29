package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

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
