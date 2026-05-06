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

func TestLoadSeedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".lark-cue")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[seed]\nfeishu_profile = \"flowops-demo\"\nwiki_name = \"星桥科技 FlowOps 知识库\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Seed.FeishuProfile != "flowops-demo" {
		t.Fatalf("seed profile = %q, want flowops-demo", cfg.Seed.FeishuProfile)
	}
	if cfg.Seed.WikiName != "星桥科技 FlowOps 知识库" {
		t.Fatalf("seed wiki = %q", cfg.Seed.WikiName)
	}
}

func TestLoadOpenClawDefaultsConfigAndEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load defaults error: %v", err)
	}
	if cfg.OpenClaw.Binary != "openclaw" || cfg.OpenClaw.TimeoutSeconds != 900 {
		t.Fatalf("unexpected defaults: %+v", cfg.OpenClaw)
	}

	configDir := filepath.Join(home, ".lark-cue")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[openclaw]\nbinary = \"oc\"\ntimeout_seconds = 120\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load config error: %v", err)
	}
	if cfg.OpenClaw.Binary != "oc" || cfg.OpenClaw.TimeoutSeconds != 120 {
		t.Fatalf("unexpected config values: %+v", cfg.OpenClaw)
	}

	t.Setenv("LARK_CUE_OPENCLAW_BINARY", "openclaw-test")
	t.Setenv("LARK_CUE_OPENCLAW_TIMEOUT_SECONDS", "30")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load env error: %v", err)
	}
	if cfg.OpenClaw.Binary != "openclaw-test" || cfg.OpenClaw.TimeoutSeconds != 30 {
		t.Fatalf("unexpected env values: %+v", cfg.OpenClaw)
	}
}
