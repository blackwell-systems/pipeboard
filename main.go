package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		// Check if stdin has data (piped input) - default to copy
		if stdinHasData() {
			if err := cmdCopy([]string{}); err != nil {
				fatal(err)
			}
			return
		}
		printHelp()
		return
	}

	cmd := args[0]
	rest := args[1:]

	commands := map[string]func([]string) error{
		"copy":    cmdCopy,
		"paste":   cmdPaste,
		"clear":   cmdClear,
		"backend": cmdBackend,
		"doctor":  cmdDoctor,
		"push":    cmdPush,
		"pull":    cmdPull,
		"show":    cmdShow,
		"slots":   cmdSlots,
		"rm":      cmdRm,
		"send":    cmdSend,
		"recv":    cmdRecv,
		"receive": cmdRecv,
		"peek":    cmdPeek,
		"history": cmdHistory,
		"fx":      cmdFx,
	}

	if fn, ok := commands[cmd]; ok {
		// Check for --help flag on any command
		if hasHelpFlag(rest) {
			printCommandHelp(cmd)
			return
		}
		if err := fn(rest); err != nil {
			fatal(err)
		}
		return
	}

	switch cmd {
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		fmt.Println("pipeboard v0.5.0")
	default:
		if useColor() {
			fmt.Fprintf(os.Stderr, "%sUnknown command: %s%s\n\n", colorRed, cmd, colorReset)
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		}
		printHelp()
		os.Exit(1)
	}
}
