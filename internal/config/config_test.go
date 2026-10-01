package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("expected non-nil default config")
	}
	if len(cfg.Profiles) == 0 {
		t.Fatal("expected default profiles")
	}
	if !cfg.Settings.ConfirmDestructive {
		t.Fatal("expected ConfirmDestructive to be true by default")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	cfg := DefaultConfig()
	cfg.Profiles = append(cfg.Profiles, &Profile{
		ID:          "custom-backup",
		Name:        "Custom Backup",
		Source:      "local:~/Test",
		Destination: "s3:mybucket",
		Operation:   OpCopy,
	})

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(loaded.Profiles) != len(cfg.Profiles) {
		t.Errorf("expected %d profiles, got %d", len(cfg.Profiles), len(loaded.Profiles))
	}

	found := false
	for _, p := range loaded.Profiles {
		if p.ID == "custom-backup" {
			found = true
			break
		}
	}
	if !found {
		t.Error("custom profile not found in loaded config")
	}

	configFile := filepath.Join(tempDir, ".config", "lazyrclone", "config.yaml")
	if _, err := os.Stat(configFile); err != nil {
		t.Errorf("expected config file at %s, got: %v", configFile, err)
	}
}

func TestCustomConfigDir(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv("LAZYRCLONE_CONFIG_DIR", customDir)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != customDir {
		t.Errorf("expected %s, got %s", customDir, dir)
	}
}
