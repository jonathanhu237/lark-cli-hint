package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFeishuProfileFromConfigAndEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".lark-cue")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[feishu]\nprofile = \"file-profile\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Feishu.Profile != "file-profile" {
		t.Fatalf("profile = %q, want file-profile", cfg.Feishu.Profile)
	}

	t.Setenv("LARK_CUE_FEISHU_PROFILE", "env-profile")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load with env error: %v", err)
	}
	if cfg.Feishu.Profile != "env-profile" {
		t.Fatalf("profile = %q, want env-profile", cfg.Feishu.Profile)
	}
}
