package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type OperationType string

const (
	OpSync   OperationType = "sync"
	OpCopy   OperationType = "copy"
	OpMove   OperationType = "move"
	OpBisync OperationType = "bisync"
	OpCheck  OperationType = "check"
)

type Profile struct {
	ID             string        `yaml:"id"`
	Name           string        `yaml:"name"`
	Source         string        `yaml:"source"`
	Destination    string        `yaml:"destination"`
	Operation      OperationType `yaml:"operation"`
	Flags          []string      `yaml:"flags,omitempty"`
	Exclude        []string      `yaml:"exclude,omitempty"`
	Include        []string      `yaml:"include,omitempty"`
	BandwidthLimit string        `yaml:"bandwidth_limit,omitempty"`
	Transfers      int           `yaml:"transfers,omitempty"`
	Checkers       int           `yaml:"checkers,omitempty"`
	LastRun        *time.Time    `yaml:"last_run,omitempty"`
	LastStatus     string        `yaml:"last_status,omitempty"` // success, failed, dry-run
}

type Settings struct {
	ConfirmDestructive bool   `yaml:"confirm_destructive"`
	DefaultView        string `yaml:"default_view"` // profiles, explorer, transfers, remotes
	Theme              string `yaml:"theme"`
	RclonePath         string `yaml:"rclone_path"`
}

type Config struct {
	Settings Settings   `yaml:"settings"`
	Profiles []*Profile `yaml:"profiles"`
}

func DefaultConfig() *Config {
	return &Config{
		Settings: Settings{
			ConfirmDestructive: true,
			DefaultView:        "profiles",
			Theme:              "catppuccin-mocha",
			RclonePath:         "rclone",
		},
		Profiles: []*Profile{
			{
				ID:          "backup-documents",
				Name:        "Backup Documents -> GDrive",
				Source:      "local:~/Documents",
				Destination: "gdrive:Backup/Documents",
				Operation:   OpCopy,
				Flags:       []string{"--fast-list"},
				Exclude:     []string{"*.tmp", "node_modules/**", ".git/**"},
				Transfers:   4,
				Checkers:    8,
			},
			{
				ID:          "sync-photos",
				Name:        "Mirror Photos -> S3 Storage",
				Source:      "local:~/Pictures",
				Destination: "s3:photos-backup",
				Operation:   OpSync,
				Flags:       []string{"--delete-excluded"},
				Transfers:   8,
				Checkers:    16,
			},
			{
				ID:          "check-vault",
				Name:        "Verify Vault Integrity",
				Source:      "local:~/SecureVault",
				Destination: "dropbox:Vault",
				Operation:   OpCheck,
				Transfers:   4,
			},
		},
	}
}

func ConfigDir() (string, error) {
	if custom := os.Getenv("LAZYRCLONE_CONFIG_DIR"); custom != "" {
		return custom, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lazyrclone"), nil
}

func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func LoadConfig() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			_ = SaveConfig(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config yaml: %w", err)
	}

	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	path := filepath.Join(dir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}
