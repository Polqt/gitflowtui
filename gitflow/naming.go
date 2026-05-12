package gitflow

import (
	"fmt"
	"regexp"
	"strings"
)

type BranchKind int

const (
	KindUnknown BranchKind = iota
	KindMain
	KindDevelop
	KindFeature
	KindRelease
	KindHotfix
)

func DetectKind(name string, cfg Config) BranchKind {
	switch {
	case name == cfg.MainBranch || name == "master":
		return KindMain
	case name == cfg.DevelopBranch:
		return KindDevelop
	case hasAnyPrefix(name, cfg.FeaturePrefixes):
		return KindFeature
	case strings.HasPrefix(name, cfg.ReleasePrefix):
		return KindRelease
	case strings.HasPrefix(name, cfg.HotfixPrefix):
		return KindHotfix
	default:
		return KindUnknown
	}
}

func ValidateBranchName(name string, cfg Config) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	switch DetectKind(name, cfg) {
	case KindFeature:
		return validateFeature(name, cfg)
	case KindRelease:
		return validateRelease(name, cfg)
	case KindHotfix:
		return validateHotfix(name, cfg)
	case KindMain, KindDevelop:
		return nil
	default:
		return fmt.Errorf("unsupported branch pattern: %q", name)
	}
}

func BuildFeatureBranch(raw string, cfg Config) (string, error) {
	slug := slugify(raw)
	if slug == "" {
		return "", fmt.Errorf("feature name cannot be empty")
	}
	prefix := defaultFeaturePrefix
	if len(cfg.FeaturePrefixes) > 0 {
		prefix = cfg.FeaturePrefixes[0]
	}
	return prefix + slug, nil
}

func BuildReleaseBranch(version string, cfg Config) (string, error) {
	version = strings.TrimSpace(version)
	if !semverPattern.MatchString(version) {
		return "", fmt.Errorf("release version must be semver, got %q", version)
	}
	return cfg.ReleasePrefix + version, nil
}

func BuildHotfixBranch(version string, cfg Config) (string, error) {
	version = strings.TrimSpace(version)
	if !semverPattern.MatchString(version) {
		return "", fmt.Errorf("hotfix version must be semver, got %q", version)
	}
	return cfg.HotfixPrefix + version, nil
}

func ExtractVersion(name string, cfg Config) (string, error) {
	var version string
	switch {
	case strings.HasPrefix(name, cfg.ReleasePrefix):
		version = strings.TrimPrefix(name, cfg.ReleasePrefix)
	case strings.HasPrefix(name, cfg.HotfixPrefix):
		version = strings.TrimPrefix(name, cfg.HotfixPrefix)
	default:
		return "", fmt.Errorf("branch %q does not contain release/hotfix version", name)
	}
	if !semverPattern.MatchString(version) {
		return "", fmt.Errorf("invalid semver in %q", name)
	}
	return version, nil
}

func validateFeature(name string, cfg Config) error {
	if !hasAnyPrefix(name, cfg.FeaturePrefixes) {
		return fmt.Errorf("feature branches must start with %v", cfg.FeaturePrefixes)
	}
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("feature branch requires suffix, got %q", name)
	}
	return nil
}

func validateRelease(name string, cfg Config) error {
	if !strings.HasPrefix(name, cfg.ReleasePrefix) {
		return fmt.Errorf("release branches must start with %q", cfg.ReleasePrefix)
	}
	version := strings.TrimPrefix(name, cfg.ReleasePrefix)
	if !semverPattern.MatchString(version) {
		return fmt.Errorf("release branch requires semver suffix, got %q", name)
	}
	return nil
}

func validateHotfix(name string, cfg Config) error {
	if !strings.HasPrefix(name, cfg.HotfixPrefix) {
		return fmt.Errorf("hotfix branches must start with %q", cfg.HotfixPrefix)
	}
	version := strings.TrimPrefix(name, cfg.HotfixPrefix)
	if !semverPattern.MatchString(version) {
		return fmt.Errorf("hotfix branch requires semver suffix, got %q", name)
	}
	return nil
}

func hasAnyPrefix(name string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

var slugPattern = regexp.MustCompile(`[^a-z0-9._-]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = slugPattern.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-./")
	return s
}
