package main

import (
	"fmt"
	"os"
)

// version is set at build time via ldflags
var version = "dev"

// commands maps command names to their handler functions
var commands = map[string]func([]string) error{
	"copy":       cmdCopy,
	"paste":      cmdPaste,
	"clear":      cmdClear,
	"backend":    cmdBackend,
	"doctor":     cmdDoctor,
	"push":       cmdPush,
	"pull":       cmdPull,
	"show":       cmdShow,
	"slots":      cmdSlots,
	"rm":         cmdRm,
	"send":       cmdSend,
	"recv":       cmdRecv,
	"receive":    cmdRecv,
	"peek":       cmdPeek,
	"history":    cmdHistory,
	"fx":         cmdFx,
	"init":       cmdInit,
	"completion": cmdCompletion,
	"watch":      cmdWatch,
	"recall":     cmdRecall,
}

// run executes the CLI with the given arguments, returning an exit code
func run(args []string, checkStdin func() bool) int {
	if len(args) == 0 {
		// Check if stdin has data (piped input) - default to copy
		if checkStdin() {
			if err := cmdCopy([]string{}); err != nil {
				printError(err)
				return 1
			}
			return 0
		}
		printHelp()
		return 0
	}

	cmd := args[0]
	rest := args[1:]

	if fn, ok := commands[cmd]; ok {
		// Check for --help flag on any command
		if hasHelpFlag(rest) {
			printCommandHelp(cmd)
			return 0
		}
		if err := fn(rest); err != nil {
			printError(err)
			return 1
		}
		return 0
	}

	switch cmd {
	case "help", "-h", "--help":
		printHelp()
		return 0
	case "version", "-v", "--version":
		fmt.Printf("pipeboard %s\n", version)
		return 0
	default:
		if useColor() {
			fmt.Fprintf(os.Stderr, "%sUnknown command: %s%s\n\n", colorRed, cmd, colorReset)
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		}
		printHelp()
		return 1
	}
}

func main() {
	os.Exit(run(os.Args[1:], stdinHasData))
}
