package git

import "strings"

// BranchType represents the type of a Git branch.
type BranchType int

type Branch struct {
	Name    string
	Type    BranchType
	Current bool
	Remote  bool
	Hash    string // short commit hash associated with the branch
}

const (
	BranchMain BranchType = iota
	BranchDevelop
	BranchFeature
	BranchRelease
	BranchHotfix
	BranchSupport
	BranchOther
)

// String returns the display name for a BranchType.
func (bt BranchType) String() string {
	switch bt {
	case BranchMain:
		return "Main"
	case BranchDevelop:
		return "Develop"
	case BranchFeature:
		return "Feature"
	case BranchRelease:
		return "Release"
	case BranchHotfix:
		return "Hotfix"
	case BranchSupport:
		return "Support"
	default:
		return "Other"
	}
}

// DetectType determines the BranchType based on the branch name.
func DetectType(name string) BranchType {
	switch {
	case name == "main" || name == "master":
		return BranchMain
	case name == "develop" || name == "dev" || name == "development":
		return BranchDevelop
	case strings.HasPrefix(name, "feature/") || strings.HasPrefix(name, "feat/"):
		return BranchFeature
	case strings.HasPrefix(name, "release/"):
		return BranchRelease
	case strings.HasPrefix(name, "hotfix/"):
		return BranchHotfix
	case strings.HasPrefix(name, "support/"):
		return BranchSupport
	default:
		return BranchOther
	}
}

