package gitflow

import "regexp"

const (
	defaultMainBranch    = "main"
	defaultDevelopBranch = "develop"
	defaultFeaturePrefix = "feature/"
	defaultFeatPrefix    = "feat/"
	defaultReleasePrefix = "release/"
	defaultHotfixPrefix  = "hotfix/"
)

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
		MainBranch:      defaultMainBranch,
		DevelopBranch:   defaultDevelopBranch,
		RemoteName:      "origin",
		FeaturePrefixes: []string{defaultFeaturePrefix, defaultFeatPrefix},
		ReleasePrefix:   defaultReleasePrefix,
		HotfixPrefix:    defaultHotfixPrefix,
		TagPrefix:       "v",
	}
}

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
