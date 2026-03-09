package gitflow

import "regexp"

type Config struct {
	MainBranch      string
	DevelopBranch   string
	RemoteName      string
	FeaturePrefixes []string
	ReleasePrefix   string
	HotfixPrefix    string
	TagPrefix       string
}

func DefaultConfig() Config {
	return Config{
		MainBranch:      "main",
		DevelopBranch:   "develop",
		RemoteName:      "origin",
		FeaturePrefixes: []string{"feature/", "feat/"},
		ReleasePrefix:   "release/",
		HotfixPrefix:    "hotfix/",
		TagPrefix:       "v",
	}
}

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
