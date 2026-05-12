package config

import "os"

// Config contains runtime settings for the TUI.
type Config struct {
	MainBranch    string
	DevelopBranch string
	RemoteName    string
	LogLimit      int
	WSAddr        string
	WSPath        string
	AIKey         string
	OllamaBaseURL string
	OllamaModel   string
}

func Default() Config {
	return Config{
		MainBranch:    "main",
		DevelopBranch: "develop",
		RemoteName:    "origin",
		LogLimit:      80,
		WSAddr:        "127.0.0.1:7373",
		WSPath:        "/ws",
	}
}

// Load reads environment overrides and returns effective config.
func Load() Config {
	cfg := Default()

	if v := os.Getenv("GITFLOW_TUI_MAIN"); v != "" {
		cfg.MainBranch = v
	}
	if v := os.Getenv("GITFLOW_TUI_DEVELOP"); v != "" {
		cfg.DevelopBranch = v
	}
	if v := os.Getenv("GITFLOW_TUI_REMOTE"); v != "" {
		cfg.RemoteName = v
	}

	if os.Getenv("GITFLOW_TUI_WS_DISABLE") == "1" {
		cfg.WSAddr = ""
	}
	if v := os.Getenv("GITFLOW_TUI_WS_ADDR"); v != "" {
		cfg.WSAddr = v
	}
	if v := os.Getenv("GITFLOW_TUI_WS_PATH"); v != "" {
		cfg.WSPath = v
	}

	cfg.AIKey = os.Getenv("ANTHROPIC_API_KEY")
	cfg.OllamaBaseURL = os.Getenv("GITFLOW_TUI_OLLAMA_BASE_URL")
	cfg.OllamaModel = os.Getenv("GITFLOW_TUI_OLLAMA_MODEL")

	return cfg
}
