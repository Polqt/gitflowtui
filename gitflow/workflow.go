package gitflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/Polqt/gitflowtui/git"
)

// Workflow orchestrates gitflow operations on top of git CLI wrapper.
type Workflow struct {
	repo *git.Repo
	cfg  Config
}

type FinishResult struct {
	MergedInto []string
	Tag        string
}

func New(repo *git.Repo, cfg Config) *Workflow {
	if len(cfg.FeaturePrefixes) == 0 {
		cfg.FeaturePrefixes = []string{"feature/", "feat/"}
	}
	if cfg.MainBranch == "" {
		cfg.MainBranch = "main"
	}
	if cfg.DevelopBranch == "" {
		cfg.DevelopBranch = "develop"
	}
	if cfg.RemoteName == "" {
		cfg.RemoteName = "origin"
	}
	if cfg.ReleasePrefix == "" {
		cfg.ReleasePrefix = "release/"
	}
	if cfg.HotfixPrefix == "" {
		cfg.HotfixPrefix = "hotfix/"
	}
	if cfg.TagPrefix == "" {
		cfg.TagPrefix = "v"
	}
	return &Workflow{repo: repo, cfg: cfg}
}

func (w *Workflow) Config() Config {
	return w.cfg
}

func (w *Workflow) StartFeature(ctx context.Context, rawName string) (string, error) {
	name, err := BuildFeatureBranch(rawName, w.cfg)
	if err != nil {
		return "", err
	}
	if err := w.repo.Checkout(ctx, w.cfg.DevelopBranch); err != nil {
		return "", err
	}
	if err := w.repo.CreateBranch(ctx, name, ""); err != nil {
		return "", err
	}
	return name, nil
}

func (w *Workflow) StartRelease(ctx context.Context, version string) (string, error) {
	name, err := BuildReleaseBranch(version, w.cfg)
	if err != nil {
		return "", err
	}
	if err := w.repo.Checkout(ctx, w.cfg.DevelopBranch); err != nil {
		return "", err
	}
	if err := w.repo.CreateBranch(ctx, name, ""); err != nil {
		return "", err
	}
	return name, nil
}

func (w *Workflow) StartHotfix(ctx context.Context, version string) (string, error) {
	name, err := BuildHotfixBranch(version, w.cfg)
	if err != nil {
		return "", err
	}
	if err := w.repo.Checkout(ctx, w.cfg.MainBranch); err != nil {
		return "", err
	}
	if err := w.repo.CreateBranch(ctx, name, ""); err != nil {
		return "", err
	}
	return name, nil
}

func (w *Workflow) FinishFeature(ctx context.Context, branch string) (FinishResult, error) {
	if strings.TrimSpace(branch) == "" {
		var err error
		branch, err = w.repo.CurrentBranch(ctx)
		if err != nil {
			return FinishResult{}, err
		}
	}
	if DetectKind(branch, w.cfg) != KindFeature {
		return FinishResult{}, fmt.Errorf("%q is not a feature branch", branch)
	}
	if err := w.ensureProtectedBranchesUpToDate(ctx); err != nil {
		return FinishResult{}, err
	}

	if err := w.repo.Checkout(ctx, w.cfg.DevelopBranch); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.MergeBranch(ctx, branch, true); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.DeleteBranch(ctx, branch, false); err != nil {
		return FinishResult{}, err
	}

	return FinishResult{MergedInto: []string{w.cfg.DevelopBranch}}, nil
}

func (w *Workflow) FinishRelease(ctx context.Context, branch string) (FinishResult, error) {
	if strings.TrimSpace(branch) == "" {
		var err error
		branch, err = w.repo.CurrentBranch(ctx)
		if err != nil {
			return FinishResult{}, err
		}
	}
	if DetectKind(branch, w.cfg) != KindRelease {
		return FinishResult{}, fmt.Errorf("%q is not a release branch", branch)
	}
	if err := w.ensureProtectedBranchesUpToDate(ctx); err != nil {
		return FinishResult{}, err
	}

	version, err := ExtractVersion(branch, w.cfg)
	if err != nil {
		return FinishResult{}, err
	}
	tag := w.cfg.TagPrefix + version

	if err := w.repo.Checkout(ctx, w.cfg.MainBranch); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.MergeBranch(ctx, branch, true); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.TagAnnotated(ctx, tag, "release "+version); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.Checkout(ctx, w.cfg.DevelopBranch); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.MergeBranch(ctx, branch, true); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.DeleteBranch(ctx, branch, false); err != nil {
		return FinishResult{}, err
	}

	return FinishResult{MergedInto: []string{w.cfg.MainBranch, w.cfg.DevelopBranch}, Tag: tag}, nil
}

func (w *Workflow) FinishHotfix(ctx context.Context, branch string) (FinishResult, error) {
	if strings.TrimSpace(branch) == "" {
		var err error
		branch, err = w.repo.CurrentBranch(ctx)
		if err != nil {
			return FinishResult{}, err
		}
	}
	if DetectKind(branch, w.cfg) != KindHotfix {
		return FinishResult{}, fmt.Errorf("%q is not a hotfix branch", branch)
	}
	if err := w.ensureProtectedBranchesUpToDate(ctx); err != nil {
		return FinishResult{}, err
	}

	version, err := ExtractVersion(branch, w.cfg)
	if err != nil {
		return FinishResult{}, err
	}
	tag := w.cfg.TagPrefix + version

	if err := w.repo.Checkout(ctx, w.cfg.MainBranch); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.MergeBranch(ctx, branch, true); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.TagAnnotated(ctx, tag, "hotfix "+version); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.Checkout(ctx, w.cfg.DevelopBranch); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.MergeBranch(ctx, branch, true); err != nil {
		return FinishResult{}, err
	}
	if err := w.repo.DeleteBranch(ctx, branch, false); err != nil {
		return FinishResult{}, err
	}

	return FinishResult{MergedInto: []string{w.cfg.MainBranch, w.cfg.DevelopBranch}, Tag: tag}, nil
}

func (w *Workflow) ensureProtectedBranchesUpToDate(ctx context.Context) error {
	if err := w.repo.Fetch(ctx); err != nil {
		return err
	}

	branches := []string{w.cfg.MainBranch, w.cfg.DevelopBranch}
	for _, local := range branches {
		remote := w.cfg.RemoteName + "/" + local
		if _, err := w.repo.RevParse(ctx, remote); err != nil {
			// Remote branch might not exist in local-only repos.
			continue
		}
		_, behind, err := w.repo.AheadBehind(ctx, local, remote)
		if err != nil {
			return err
		}
		if behind > 0 {
			return fmt.Errorf("%s is behind %s by %d commit(s)", local, remote, behind)
		}
	}
	return nil
}
