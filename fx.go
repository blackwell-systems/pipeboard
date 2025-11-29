package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// cmdFx runs a user-defined clipboard transform (supports chaining)
func cmdFx(args []string) error {
	// Parse flags and collect transform names
	var dryRun bool
	var listMode bool
	var fxNames []string

	for _, arg := range args {
		switch arg {
		case "--list", "-l":
			listMode = true
		case "--dry-run", "-n":
			dryRun = true
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown flag: %s", arg)
			}
			fxNames = append(fxNames, arg)
		}
	}

	cfg, err := loadConfigForFx()
	if err != nil {
		return err
	}

	// List mode
	if listMode {
		return fxList(cfg)
	}

	// Require at least one transform name
	if len(fxNames) == 0 {
		return fmt.Errorf("usage: pipeboard fx <name> [name2...] [--dry-run]\n       pipeboard fx --list")
	}

	// Validate all transforms exist before reading clipboard
	var transforms []FxConfig
	for _, name := range fxNames {
		fx, err := cfg.getFx(name)
		if err != nil {
			return err
		}
		transforms = append(transforms, fx)
	}

	// Read clipboard
	data, err := readClipboard()
	if err != nil {
		return fmt.Errorf("reading clipboard: %w", err)
	}
	originalSize := len(data)

	// Run transforms in order, feeding output → input
	// If any step fails, abort without modifying clipboard
	result := data
	for i, fx := range transforms {
		cmdArgs := fx.getCommand()
		result, err = runTransform(cmdArgs, result)
		if err != nil {
			return fmt.Errorf("transform %q (step %d) failed: %w; clipboard unchanged", fxNames[i], i+1, err)
		}
		// Check for empty output
		if len(result) == 0 {
			return fmt.Errorf("transform %q (step %d) produced empty output; clipboard unchanged", fxNames[i], i+1)
		}
	}

	// Dry run mode - print result to stdout, never touch clipboard
	if dryRun {
		_, err = os.Stdout.Write(result)
		return err
	}

	// Write result back to clipboard
	if err := writeClipboard(result); err != nil {
		return fmt.Errorf("writing clipboard: %w", err)
	}

	// Report what happened
	chainDesc := strings.Join(fxNames, " → ")
	fmt.Printf("fx %s: %s → %s\n", chainDesc, formatSize(int64(originalSize)), formatSize(int64(len(result))))
	recordHistory("fx:"+chainDesc, "", int64(len(result)))
	return nil
}

// fxList prints available transforms
func fxList(cfg *Config) error {
	if len(cfg.Fx) == 0 {
		fmt.Println("No transforms defined.")
		fmt.Println("\nAdd transforms to your config:")
		fmt.Println("  fx:")
		fmt.Println("    pretty-json:")
		fmt.Println("      cmd: [\"jq\", \".\"]")
		fmt.Println("      description: \"Format JSON\"")
		return nil
	}

	fmt.Printf("%-20s  %s\n", "NAME", "DESCRIPTION")
	for name, fx := range cfg.Fx {
		desc := fx.Description
		if desc == "" {
			if fx.Shell != "" {
				desc = fmt.Sprintf("sh -c %q", fx.Shell)
			} else if len(fx.Cmd) > 0 {
				desc = strings.Join(fx.Cmd, " ")
			}
			// Truncate long descriptions
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
		}
		fmt.Printf("%-20s  %s\n", name, desc)
	}
	return nil
}

// runTransform executes a transform command with input data
func runTransform(cmdArgs []string, input []byte) ([]byte, error) {
	if len(cmdArgs) == 0 {
		return nil, errors.New("no command specified")
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = bytes.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Include stderr in error message for debugging
		errMsg := stderr.String()
		if errMsg != "" {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(errMsg))
		}
		return nil, err
	}

	return stdout.Bytes(), nil
}
