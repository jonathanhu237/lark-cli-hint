package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	LLM        LLMConfig
	Feishu     FeishuConfig
	Evaluation EvaluationConfig
}

type LLMConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

type FeishuConfig struct {
	Profile         string
	DefaultPushChat string
	SendPushDefault bool
}

type EvaluationConfig struct {
	LogPath string
}

func Load() (Config, error) {
	home, _ := os.UserHomeDir()
	cfg := Config{
		LLM: LLMConfig{
			BaseURL: "https://api.openai.com/v1",
		},
		Evaluation: EvaluationConfig{
			LogPath: filepath.Join(home, ".lark-cue", "evaluations.jsonl"),
		},
	}

	path := filepath.Join(home, ".lark-cue", "config.toml")
	if err := loadFile(path, &cfg); err != nil && !os.IsNotExist(err) {
		applyEnv(&cfg)
		return cfg, err
	}
	applyEnv(&cfg)
	cfg.Evaluation.LogPath = expandHome(cfg.Evaluation.LogPath, home)
	return cfg, nil
}

func loadFile(path string, cfg *Config) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := stripComment(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch section + "." + key {
		case "llm.base_url":
			cfg.LLM.BaseURL = value
		case "llm.api_key":
			cfg.LLM.APIKey = value
		case "llm.model":
			cfg.LLM.Model = value
		case "feishu.default_push_chat":
			cfg.Feishu.DefaultPushChat = value
		case "feishu.profile":
			cfg.Feishu.Profile = value
		case "feishu.send_push_default":
			cfg.Feishu.SendPushDefault, _ = strconv.ParseBool(value)
		case "evaluation.log_path":
			cfg.Evaluation.LogPath = value
		}
	}
	return scanner.Err()
}

func applyEnv(cfg *Config) {
	if value := os.Getenv("LARK_CUE_LLM_BASE_URL"); value != "" {
		cfg.LLM.BaseURL = value
	}
	if value := os.Getenv("LARK_CUE_LLM_API_KEY"); value != "" {
		cfg.LLM.APIKey = value
	}
	if value := os.Getenv("LARK_CUE_LLM_MODEL"); value != "" {
		cfg.LLM.Model = value
	}
	if value := os.Getenv("LARK_CUE_PUSH_CHAT"); value != "" {
		cfg.Feishu.DefaultPushChat = value
	}
	if value := firstNonEmpty(os.Getenv("LARK_CUE_FEISHU_PROFILE"), os.Getenv("LARK_CUE_LARK_PROFILE")); value != "" {
		cfg.Feishu.Profile = value
	}
	if value := os.Getenv("LARK_CUE_SEND_PUSH_DEFAULT"); value != "" {
		cfg.Feishu.SendPushDefault, _ = strconv.ParseBool(value)
	}
	if value := os.Getenv("LARK_CUE_EVAL_LOG"); value != "" {
		cfg.Evaluation.LogPath = value
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stripComment(line string) string {
	if idx := strings.IndexByte(line, '#'); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(line)
}

func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}
