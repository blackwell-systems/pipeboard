package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Backend string    `yaml:"backend"`
	S3      *S3Config `yaml:"s3,omitempty"`
}

type S3Config struct {
	Bucket  string `yaml:"bucket"`
	Region  string `yaml:"region"`
	Prefix  string `yaml:"prefix,omitempty"`
	Profile string `yaml:"profile,omitempty"`
	SSE     string `yaml:"sse,omitempty"` // "AES256" or "aws:kms"
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

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("PIPEBOARD_BACKEND"); v != "" {
		cfg.Backend = v
	}

	if cfg.S3 == nil {
		cfg.S3 = &S3Config{}
	}

	if v := os.Getenv("PIPEBOARD_S3_BUCKET"); v != "" {
		cfg.S3.Bucket = v
		if cfg.Backend == "" {
			cfg.Backend = "s3"
		}
	}
	if v := os.Getenv("PIPEBOARD_S3_REGION"); v != "" {
		cfg.S3.Region = v
	}
	if v := os.Getenv("PIPEBOARD_S3_PREFIX"); v != "" {
		cfg.S3.Prefix = v
	}
	if v := os.Getenv("PIPEBOARD_S3_PROFILE"); v != "" {
		cfg.S3.Profile = v
	}
	if v := os.Getenv("PIPEBOARD_S3_SSE"); v != "" {
		cfg.S3.SSE = v
	}
}

func validateConfig(cfg *Config) error {
	if cfg.Backend == "" {
		return fmt.Errorf("backend not specified in config")
	}

	switch cfg.Backend {
	case "s3":
		if cfg.S3 == nil {
			return fmt.Errorf("s3 backend selected but s3 config missing")
		}
		if cfg.S3.Bucket == "" {
			return fmt.Errorf("s3.bucket is required")
		}
		if cfg.S3.Region == "" {
			return fmt.Errorf("s3.region is required")
		}
	default:
		return fmt.Errorf("unsupported backend: %s", cfg.Backend)
	}

	return nil
}
