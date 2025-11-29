package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdInit(args []string) error {
	// Check if config already exists
	cfgPath := configPath()
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("Config file already exists at %s\n", cfgPath)
		if !promptYesNo("Overwrite?", false) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Println("pipeboard init - Configuration Wizard")
	fmt.Println("======================================")
	fmt.Println()

	config := &Config{}

	// Step 1: Sync backend
	fmt.Println("Sync Backend")
	fmt.Println("------------")
	fmt.Println("pipeboard can sync clipboard slots across machines using:")
	fmt.Println("  local - Store slots locally (no sync, good for getting started)")
	fmt.Println("  s3    - Sync via AWS S3 bucket (requires AWS credentials)")
	fmt.Println("  none  - Disable slot features")
	fmt.Println()

	backend := promptChoice("Choose backend", []string{"local", "s3", "none"}, "local")
	config.Sync = &SyncConfig{Backend: backend}

	switch backend {
	case "s3":
		fmt.Println()
		fmt.Println("S3 Configuration")
		fmt.Println("-----------------")
		config.Sync.S3 = &S3Config{}
		config.Sync.S3.Bucket = promptString("S3 bucket name", "")
		config.Sync.S3.Region = promptString("AWS region", "us-east-1")
		config.Sync.S3.Prefix = promptString("Key prefix (optional)", "pipeboard/")

		fmt.Println()
		fmt.Println("Encryption")
		fmt.Println("----------")
		if promptYesNo("Enable end-to-end encryption?", true) {
			config.Sync.Encryption = "aes256"
			fmt.Println("Set PIPEBOARD_PASSPHRASE environment variable with your encryption key.")
		}

		fmt.Println()
		fmt.Println("TTL (Time to Live)")
		fmt.Println("------------------")
		ttl := promptString("Slot expiry in days (0 = never)", "30")
		if ttl != "0" && ttl != "" {
			var ttlDays int
			if n, err := fmt.Sscanf(ttl, "%d", &ttlDays); err != nil || n != 1 {
				fmt.Printf("Invalid TTL value %q, using default (30 days)\n", ttl)
				config.Sync.TTLDays = 30
			} else {
				config.Sync.TTLDays = ttlDays
			}
		}
	case "local":
		config.Sync.Local = &LocalConfig{}
		fmt.Println()
		fmt.Println("Local slots will be stored in ~/.config/pipeboard/slots/")
	}

	// Step 2: Peers (SSH sync)
	fmt.Println()
	fmt.Println("Peer Configuration")
	fmt.Println("------------------")
	fmt.Println("Peers allow direct clipboard sharing via SSH.")

	if promptYesNo("Configure a peer now?", false) {
		config.Peers = make(map[string]PeerConfig)
		for {
			fmt.Println()
			name := promptString("Peer name (e.g., 'work', 'laptop')", "")
			if name == "" {
				break
			}
			sshHost := promptString("SSH host (e.g., 'user@hostname')", "")
			if sshHost == "" {
				break
			}
			config.Peers[name] = PeerConfig{
				SSH:       sshHost,
				RemoteCmd: "pipeboard",
			}
			fmt.Printf("Added peer '%s' -> %s\n", name, sshHost)

			if !promptYesNo("Add another peer?", false) {
				break
			}
		}

		if len(config.Peers) > 0 {
			// Ask for default peer
			var peerNames []string
			for name := range config.Peers {
				peerNames = append(peerNames, name)
			}
			if len(peerNames) == 1 {
				config.Defaults = &DefaultsConfig{Peer: peerNames[0]}
			} else if promptYesNo("Set a default peer?", true) {
				defaultPeer := promptChoice("Default peer", peerNames, peerNames[0])
				config.Defaults = &DefaultsConfig{Peer: defaultPeer}
			}
		}
	}

	// Step 3: Transforms (fx)
	fmt.Println()
	fmt.Println("Transforms")
	fmt.Println("----------")
	fmt.Println("Transforms let you process clipboard contents (e.g., format JSON, strip ANSI).")

	if promptYesNo("Add example transforms?", true) {
		config.Fx = map[string]FxConfig{
			"pretty-json": {
				Cmd:         []string{"jq", "."},
				Description: "Format JSON with jq",
			},
			"minify-json": {
				Cmd:         []string{"jq", "-c", "."},
				Description: "Minify JSON to single line",
			},
			"sort-lines": {
				Shell:       "sort",
				Description: "Sort lines alphabetically",
			},
			"unique": {
				Shell:       "sort -u",
				Description: "Remove duplicate lines",
			},
			"trim": {
				Shell:       "sed 's/^[[:space:]]*//;s/[[:space:]]*$//'",
				Description: "Trim whitespace from lines",
			},
			"upper": {
				Shell:       "tr '[:lower:]' '[:upper:]'",
				Description: "Convert to uppercase",
			},
			"lower": {
				Shell:       "tr '[:upper:]' '[:lower:]'",
				Description: "Convert to lowercase",
			},
		}
	}

	// Write config
	fmt.Println()
	fmt.Println("Writing configuration...")

	// Ensure directory exists
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Generate YAML manually (to avoid yaml dependency issues)
	yaml := generateConfigYAML(config)

	if err := os.WriteFile(cfgPath, []byte(yaml), 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("\n✓ Configuration saved to %s\n", cfgPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  pipeboard doctor    - Verify your clipboard setup")
	fmt.Println("  pipeboard copy      - Copy text to clipboard")
	fmt.Println("  pipeboard paste     - Paste from clipboard")
	if backend != "none" {
		fmt.Println("  pipeboard push <name> - Save clipboard to a slot")
		fmt.Println("  pipeboard slots     - List your slots")
	}
	if len(config.Fx) > 0 {
		fmt.Println("  pipeboard fx --list - See available transforms")
	}

	return nil
}

// promptString asks for a string input with a default value
func promptString(prompt, defaultVal string) string {
	reader := bufio.NewReader(os.Stdin)
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

// promptYesNo asks a yes/no question
func promptYesNo(prompt string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	hint := "y/N"
	if defaultYes {
		hint = "Y/n"
	}
	fmt.Printf("%s [%s]: ", prompt, hint)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

// promptChoice asks user to choose from options
func promptChoice(prompt string, options []string, defaultVal string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (%s) [%s]: ", prompt, strings.Join(options, "/"), defaultVal)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultVal
	}
	// Validate
	for _, opt := range options {
		if input == opt {
			return input
		}
	}
	// Invalid, return default
	fmt.Printf("Invalid choice, using '%s'\n", defaultVal)
	return defaultVal
}

// generateConfigYAML creates YAML from config struct
func generateConfigYAML(cfg *Config) string {
	var sb strings.Builder

	sb.WriteString("# pipeboard configuration\n")
	sb.WriteString("# Generated by 'pipeboard init'\n\n")

	// Sync section
	if cfg.Sync != nil {
		sb.WriteString("sync:\n")
		sb.WriteString(fmt.Sprintf("  backend: %s\n", cfg.Sync.Backend))

		if cfg.Sync.S3 != nil {
			sb.WriteString("  s3:\n")
			sb.WriteString(fmt.Sprintf("    bucket: %s\n", cfg.Sync.S3.Bucket))
			sb.WriteString(fmt.Sprintf("    region: %s\n", cfg.Sync.S3.Region))
			if cfg.Sync.S3.Prefix != "" {
				sb.WriteString(fmt.Sprintf("    prefix: %s\n", cfg.Sync.S3.Prefix))
			}
		}

		if cfg.Sync.Local != nil && cfg.Sync.Local.Path != "" {
			sb.WriteString("  local:\n")
			sb.WriteString(fmt.Sprintf("    path: %s\n", cfg.Sync.Local.Path))
		}

		if cfg.Sync.Encryption != "" {
			sb.WriteString(fmt.Sprintf("  encryption: %s\n", cfg.Sync.Encryption))
		}

		if cfg.Sync.TTLDays > 0 {
			sb.WriteString(fmt.Sprintf("  ttl_days: %d\n", cfg.Sync.TTLDays))
		}
	}

	// Peers section
	if len(cfg.Peers) > 0 {
		sb.WriteString("\npeers:\n")
		for name, peer := range cfg.Peers {
			sb.WriteString(fmt.Sprintf("  %s:\n", name))
			sb.WriteString(fmt.Sprintf("    ssh: %s\n", peer.SSH))
			if peer.RemoteCmd != "" && peer.RemoteCmd != "pipeboard" {
				sb.WriteString(fmt.Sprintf("    remote_cmd: %s\n", peer.RemoteCmd))
			}
		}
	}

	// Defaults section
	if cfg.Defaults != nil && cfg.Defaults.Peer != "" {
		sb.WriteString("\ndefaults:\n")
		sb.WriteString(fmt.Sprintf("  peer: %s\n", cfg.Defaults.Peer))
	}

	// Fx section
	if len(cfg.Fx) > 0 {
		sb.WriteString("\nfx:\n")
		for name, fx := range cfg.Fx {
			sb.WriteString(fmt.Sprintf("  %s:\n", name))
			if len(fx.Cmd) > 0 {
				sb.WriteString("    cmd: [")
				for i, c := range fx.Cmd {
					if i > 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(fmt.Sprintf("%q", c))
				}
				sb.WriteString("]\n")
			} else if fx.Shell != "" {
				sb.WriteString(fmt.Sprintf("    shell: %q\n", fx.Shell))
			}
			if fx.Description != "" {
				sb.WriteString(fmt.Sprintf("    description: %q\n", fx.Description))
			}
		}
	}

	return sb.String()
}
