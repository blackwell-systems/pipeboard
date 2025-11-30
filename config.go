package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  int                   `yaml:"version"`
	Defaults *DefaultsConfig       `yaml:"defaults,omitempty"`
	Sync     *SyncConfig           `yaml:"sync,omitempty"`
	History  *HistoryConfig        `yaml:"history,omitempty"`
	Peers    map[string]PeerConfig `yaml:"peers,omitempty"`
	Fx       map[string]FxConfig   `yaml:"fx,omitempty"` // clipboard transforms

	// Legacy fields for backwards compatibility
	Backend string    `yaml:"backend,omitempty"`
	S3      *S3Config `yaml:"s3,omitempty"`
}

type DefaultsConfig struct {
	Peer string `yaml:"peer,omitempty"` // default peer for send/recv/peek
}

type HistoryConfig struct {
	Limit int `yaml:"limit,omitempty"` // max clipboard history entries (default: 20)
}

// FxConfig defines a clipboard transform
type FxConfig struct {
	Cmd         []string `yaml:"cmd,omitempty"`         // command and args
	Shell       string   `yaml:"shell,omitempty"`       // shorthand: runs via "sh -c"
	Description string   `yaml:"description,omitempty"` // shown in fx --list
}

type SyncConfig struct {
	Backend    string       `yaml:"backend"`              // "none", "s3", or "local"
	S3         *S3Config    `yaml:"s3,omitempty"`
	Local      *LocalConfig `yaml:"local,omitempty"`
	Encryption string       `yaml:"encryption,omitempty"` // "none" or "aes256"
	Passphrase string       `yaml:"passphrase,omitempty"` // for client-side encryption
	TTLDays    int          `yaml:"ttl_days,omitempty"`   // auto-expire slots after N days (0 = never)
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
	debugLog("loading config from: %s", path)
	if path == "" {
		return nil, fmt.Errorf("could not determine config path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			debugLog("config file not found")
			return nil, fmt.Errorf("config file not found: %s\n\nNo peers configured. Create a config file to enable send/recv.\nSee: pipeboard help", path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyDefaults(&cfg)
	debugLog("config loaded: %d peers defined", len(cfg.Peers))
	return &cfg, nil
}

// loadConfig loads config for sync operations (push/pull/slots/rm/show).
// Requires sync backend to be configured.
func loadConfig() (*Config, error) {
	path := configPath()
	debugLog("loading config from: %s", path)
	if path == "" {
		return nil, fmt.Errorf("could not determine config path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			debugLog("config file not found")
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

	debugLog("config loaded: sync backend=%s", cfg.Sync.Backend)
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
	applyLegacyConfig(cfg)
	applyBackendEnv(cfg)
	applyS3Env(cfg)
}

func applyLegacyConfig(cfg *Config) {
	if cfg.Backend != "" && cfg.Sync == nil {
		cfg.Sync = &SyncConfig{
			Backend: cfg.Backend,
			S3:      cfg.S3,
		}
	}
}

func applyBackendEnv(cfg *Config) {
	if v := os.Getenv("PIPEBOARD_BACKEND"); v != "" {
		if cfg.Sync == nil {
			cfg.Sync = &SyncConfig{}
		}
		cfg.Sync.Backend = v
	}
}

func ensureSyncS3(cfg *Config) {
	if cfg.Sync == nil {
		cfg.Sync = &SyncConfig{S3: &S3Config{}}
	}
	if cfg.Sync.S3 == nil {
		cfg.Sync.S3 = &S3Config{}
	}
}

func applyS3Env(cfg *Config) {
	if v := os.Getenv("PIPEBOARD_S3_BUCKET"); v != "" {
		ensureSyncS3(cfg)
		cfg.Sync.S3.Bucket = v
		if cfg.Sync.Backend == "" {
			cfg.Sync.Backend = "s3"
		}
	}

	if cfg.Sync == nil || cfg.Sync.S3 == nil {
		return
	}

	envMappings := []struct {
		env  string
		dest *string
	}{
		{"PIPEBOARD_S3_REGION", &cfg.Sync.S3.Region},
		{"PIPEBOARD_S3_PREFIX", &cfg.Sync.S3.Prefix},
		{"PIPEBOARD_S3_PROFILE", &cfg.Sync.S3.Profile},
		{"PIPEBOARD_S3_SSE", &cfg.Sync.S3.SSE},
	}

	for _, m := range envMappings {
		if v := os.Getenv(m.env); v != "" {
			*m.dest = v
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
	case "local":
		// Local backend requires no mandatory config (uses defaults)
		if cfg.Sync.Local == nil {
			cfg.Sync.Local = &LocalConfig{}
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

// getDefaultPeer returns the default peer name from config, or error if not set.
func (cfg *Config) getDefaultPeer() (string, error) {
	if cfg.Defaults == nil || cfg.Defaults.Peer == "" {
		return "", fmt.Errorf("no default peer configured; set 'defaults.peer' in config or specify a peer name")
	}
	return cfg.Defaults.Peer, nil
}

// loadConfigForFx loads config for fx commands.
// Returns empty config if file doesn't exist (no fx defined is valid).
func loadConfigForFx() (*Config, error) {
	path := configPath()
	if path == "" {
		return &Config{Fx: make(map[string]FxConfig)}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Fx: make(map[string]FxConfig)}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Fx == nil {
		cfg.Fx = make(map[string]FxConfig)
	}

	return &cfg, nil
}

// getFx looks up an fx transform by name.
func (cfg *Config) getFx(name string) (FxConfig, error) {
	if len(cfg.Fx) == 0 {
		return FxConfig{}, fmt.Errorf("no transforms defined; add 'fx' section to config")
	}
	fx, ok := cfg.Fx[name]
	if !ok {
		return FxConfig{}, fmt.Errorf("unknown transform %q; define it under 'fx' in config", name)
	}
	if len(fx.Cmd) == 0 && fx.Shell == "" {
		return FxConfig{}, fmt.Errorf("transform %q has no 'cmd' or 'shell' defined", name)
	}
	return fx, nil
}

// getCommand returns the command to execute for this transform.
func (fx *FxConfig) getCommand() []string {
	if fx.Shell != "" {
		return []string{"sh", "-c", fx.Shell}
	}
	return fx.Cmd
}
