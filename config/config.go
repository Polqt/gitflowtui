package config

import "os"

// Config contains runtime settings for the TUI.
type Config struct {
	MainBranch    string
	DevelopBranch string
	RemoteName    string
	LogLimit      int
}

func Default() Config {
	return Config{
		MainBranch:    "main",
		DevelopBranch: "develop",
		RemoteName:    "origin",
		LogLimit:      80,
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

	return cfg
}
