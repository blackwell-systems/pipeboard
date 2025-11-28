package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version int                   `yaml:"version"`
	Sync    *SyncConfig           `yaml:"sync,omitempty"`
	Peers   map[string]PeerConfig `yaml:"peers,omitempty"`

	// Legacy fields for backwards compatibility
	Backend string    `yaml:"backend,omitempty"`
	S3      *S3Config `yaml:"s3,omitempty"`
}

type SyncConfig struct {
	Backend string    `yaml:"backend"` // "none" or "s3"
	S3      *S3Config `yaml:"s3,omitempty"`
}

type S3Config struct {
	Bucket  string `yaml:"bucket"`
	Region  string `yaml:"region"`
	Prefix  string `yaml:"prefix,omitempty"`
	Profile string `yaml:"profile,omitempty"`
	SSE     string `yaml:"sse,omitempty"` // "AES256" or "aws:kms"
}

type PeerConfig struct {
	SSH       string `yaml:"ssh"`                  // SSH host/alias
	RemoteCmd string `yaml:"remote_cmd,omitempty"` // default: "pipeboard"
}

func configPath() string {
	if p := os.Getenv("PIPEBOARD_CONFIG"); p != "" {
		return p
	}

	// XDG_CONFIG_HOME or ~/.config
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = filepath.Join(home, ".config")
	}

	return filepath.Join(configDir, "pipeboard", "config.yaml")
}

// loadConfigForPeers loads config, returning defaults if file doesn't exist.
// Used by send/recv/peek commands.
func loadConfigForPeers() (*Config, error) {
	path := configPath()
	if path == "" {
		return nil, fmt.Errorf("could not determine config path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s\n\nNo peers configured. Create a config file to enable send/recv.\nSee: pipeboard help", path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

// loadConfig loads config for sync operations (push/pull/slots/rm/show).
// Requires sync backend to be configured.
func loadConfig() (*Config, error) {
	path := configPath()
	if path == "" {
		return nil, fmt.Errorf("could not determine config path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s\n\nRemote sync not configured. Create a config file to enable push/pull.\nSee: pipeboard help", path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(&cfg)
	applyDefaults(&cfg)

	if err := validateSyncConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	if cfg.Peers == nil {
		cfg.Peers = make(map[string]PeerConfig)
	}

	// Ensure RemoteCmd defaults to "pipeboard"
	for name, peer := range cfg.Peers {
		if peer.RemoteCmd == "" {
			peer.RemoteCmd = "pipeboard"
		}
		cfg.Peers[name] = peer
	}
}

func applyEnvOverrides(cfg *Config) {
	// Handle legacy config format (backend at root level)
	if cfg.Backend != "" && cfg.Sync == nil {
		cfg.Sync = &SyncConfig{
			Backend: cfg.Backend,
			S3:      cfg.S3,
		}
	}

	if v := os.Getenv("PIPEBOARD_BACKEND"); v != "" {
		if cfg.Sync == nil {
			cfg.Sync = &SyncConfig{}
		}
		cfg.Sync.Backend = v
	}

	// S3 env overrides
	if cfg.Sync != nil && cfg.Sync.S3 == nil {
		cfg.Sync.S3 = &S3Config{}
	}

	if v := os.Getenv("PIPEBOARD_S3_BUCKET"); v != "" {
		if cfg.Sync == nil {
			cfg.Sync = &SyncConfig{S3: &S3Config{}}
		}
		if cfg.Sync.S3 == nil {
			cfg.Sync.S3 = &S3Config{}
		}
		cfg.Sync.S3.Bucket = v
		if cfg.Sync.Backend == "" {
			cfg.Sync.Backend = "s3"
		}
	}
	if cfg.Sync != nil && cfg.Sync.S3 != nil {
		if v := os.Getenv("PIPEBOARD_S3_REGION"); v != "" {
			cfg.Sync.S3.Region = v
		}
		if v := os.Getenv("PIPEBOARD_S3_PREFIX"); v != "" {
			cfg.Sync.S3.Prefix = v
		}
		if v := os.Getenv("PIPEBOARD_S3_PROFILE"); v != "" {
			cfg.Sync.S3.Profile = v
		}
		if v := os.Getenv("PIPEBOARD_S3_SSE"); v != "" {
			cfg.Sync.S3.SSE = v
		}
	}
}

func validateSyncConfig(cfg *Config) error {
	if cfg.Sync == nil {
		return fmt.Errorf("sync backend not configured")
	}

	if cfg.Sync.Backend == "" || cfg.Sync.Backend == "none" {
		return fmt.Errorf("sync backend not configured (backend: none)")
	}

	switch cfg.Sync.Backend {
	case "s3":
		if cfg.Sync.S3 == nil {
			return fmt.Errorf("s3 backend selected but s3 config missing")
		}
		if cfg.Sync.S3.Bucket == "" {
			return fmt.Errorf("s3.bucket is required")
		}
		if cfg.Sync.S3.Region == "" {
			return fmt.Errorf("s3.region is required")
		}
	default:
		return fmt.Errorf("unsupported backend: %s", cfg.Sync.Backend)
	}

	return nil
}

// getPeer looks up a peer by name and returns a fully-defaulted PeerConfig.
func (cfg *Config) getPeer(name string) (PeerConfig, error) {
	if len(cfg.Peers) == 0 {
		return PeerConfig{}, fmt.Errorf("no peers configured; add peer %q under 'peers' in config", name)
	}
	peer, ok := cfg.Peers[name]
	if !ok {
		return PeerConfig{}, fmt.Errorf("unknown peer %q; define it under 'peers' in config", name)
	}
	if peer.SSH == "" {
		return PeerConfig{}, fmt.Errorf("peer %q is missing 'ssh' field", name)
	}
	if peer.RemoteCmd == "" {
		peer.RemoteCmd = "pipeboard"
	}
	return peer, nil
}
