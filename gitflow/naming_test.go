package gitflow

import "testing"

func TestBuildFeatureBranch(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "simple", input: "checkout redesign", want: "feature/checkout-redesign"},
		{name: "mixed chars", input: "API@v2 migration", want: "feature/api-v2-migration"},
		{name: "empty", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := BuildFeatureBranch(tt.input, cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildFeatureBranch returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected branch: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestBuildReleaseBranch(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	tests := []struct {
		name    string
		version string
		want    string
		wantErr bool
	}{
		{name: "valid semver", version: "1.2.3", want: "release/1.2.3"},
		{name: "valid prerelease", version: "1.2.3-rc.1", want: "release/1.2.3-rc.1"},
		{name: "invalid", version: "1.2", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := BuildReleaseBranch(tt.version, cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildReleaseBranch returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected branch: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestExtractVersion(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	tests := []struct {
		name    string
		branch  string
		want    string
		wantErr bool
	}{
		{name: "release", branch: "release/2.3.4", want: "2.3.4"},
		{name: "hotfix", branch: "hotfix/2.3.5", want: "2.3.5"},
		{name: "feature", branch: "feature/foo", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ExtractVersion(tt.branch, cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ExtractVersion returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected version: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestDetectKind(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	tests := []struct {
		branch string
		want   BranchKind
	}{
		{branch: "main", want: KindMain},
		{branch: "develop", want: KindDevelop},
		{branch: "feature/add-auth", want: KindFeature},
		{branch: "feat/add-auth", want: KindFeature},
		{branch: "release/1.0.0", want: KindRelease},
		{branch: "hotfix/1.0.1", want: KindHotfix},
		{branch: "chore/cleanup", want: KindUnknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.branch, func(t *testing.T) {
			t.Parallel()
			got := DetectKind(tt.branch, cfg)
			if got != tt.want {
				t.Fatalf("unexpected kind for %q: got %v want %v", tt.branch, got, tt.want)
			}
		})
	}
}
