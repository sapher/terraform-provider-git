package provider

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

func TestReadRepositoryInfoReturnsOriginRemote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo := initRepository(t, dir)

	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"git@github.com:example/repo.git"},
	})
	if err != nil {
		t.Fatalf("CreateRemote returned error: %v", err)
	}

	setRemoteHead(t, repo, "origin", "main")
	setCurrentBranchUpstream(t, repo, "feature", "origin", "trunk")

	originURL, defaultBranch, err := readRepositoryInfo(dir)
	if err != nil {
		t.Fatalf("readRepositoryInfo returned error: %v", err)
	}

	if originURL != "git@github.com:example/repo.git" {
		t.Fatalf("originURL = %q, want %q", originURL, "git@github.com:example/repo.git")
	}

	if defaultBranch == nil || *defaultBranch != "main" {
		t.Fatalf("defaultBranch = %v, want %q", defaultBranch, "main")
	}
}

func TestReadRepositoryInfoRequiresOriginRemote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepository(t, dir)

	_, _, err := readRepositoryInfo(dir)
	if err == nil {
		t.Fatal("readRepositoryInfo returned nil error, want missing origin error")
	}
}

func TestReadRepositoryInfoFallsBackToCurrentBranchUpstream(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo := initRepository(t, dir)

	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"git@github.com:example/repo.git"},
	})
	if err != nil {
		t.Fatalf("CreateRemote returned error: %v", err)
	}

	setCurrentBranchUpstream(t, repo, "feature", "origin", "main")

	_, defaultBranch, err := readRepositoryInfo(dir)
	if err != nil {
		t.Fatalf("readRepositoryInfo returned error: %v", err)
	}

	if defaultBranch == nil || *defaultBranch != "main" {
		t.Fatalf("defaultBranch = %v, want %q", defaultBranch, "main")
	}
}

func TestReadRepositoryInfoAllowsUnknownDefaultRemoteBranch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo := initRepository(t, dir)

	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"git@github.com:example/repo.git"},
	})
	if err != nil {
		t.Fatalf("CreateRemote returned error: %v", err)
	}

	_, defaultBranch, err := readRepositoryInfo(dir)
	if err != nil {
		t.Fatalf("readRepositoryInfo returned error: %v", err)
	}

	if defaultBranch != nil {
		t.Fatalf("defaultBranch = %q, want nil", *defaultBranch)
	}
}

func TestExtractNamespace(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		originURL string
		want      string
	}{
		"scp-style SSH URL": {
			originURL: "git@gitlab.com:global-savings-group/tribes/poc/git-provider.git",
			want:      "global-savings-group/tribes/poc/git-provider",
		},
		"HTTPS URL": {
			originURL: "https://gitlab.com/global-savings-group/tribes/poc/git-provider.git",
			want:      "global-savings-group/tribes/poc/git-provider",
		},
		"HTTPS URL without git suffix": {
			originURL: "https://gitlab.com/global-savings-group/tribes/poc/git-provider",
			want:      "global-savings-group/tribes/poc/git-provider",
		},
		"SSH URL without git suffix": {
			originURL: "git@gitlab.com:global-savings-group/tribes/poc/git-provider",
			want:      "global-savings-group/tribes/poc/git-provider",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := extractNamespace(tt.originURL)
			if err != nil {
				t.Fatalf("extractNamespace returned error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("extractNamespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractNamespaceRejectsInvalidURLs(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"empty URL":        "",
		"missing host":     "https:///global-savings-group/tribes/poc/git-provider.git",
		"missing path":     "https://gitlab.com",
		"missing SSH path": "git@gitlab.com:",
		"unsupported URL":  "global-savings-group/tribes/poc/git-provider.git",
	}

	for name, originURL := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := extractNamespace(originURL); err == nil {
				t.Fatal("extractNamespace returned nil error, want invalid URL error")
			}
		})
	}
}

func initRepository(t *testing.T, dir string) *git.Repository {
	t.Helper()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit returned error: %v", err)
	}

	return repo
}

func setRemoteHead(t *testing.T, repo *git.Repository, remote string, branch string) {
	t.Helper()

	remoteBranch := plumbing.NewRemoteReferenceName(remote, branch)
	remoteHead := plumbing.NewRemoteHEADReferenceName(remote)

	err := repo.Storer.SetReference(plumbing.NewHashReference(remoteBranch, plumbing.ZeroHash))
	if err != nil {
		t.Fatalf("set remote branch reference: %v", err)
	}

	err = repo.Storer.SetReference(plumbing.NewSymbolicReference(remoteHead, remoteBranch))
	if err != nil {
		t.Fatalf("set remote HEAD reference: %v", err)
	}
}

func setCurrentBranchUpstream(t *testing.T, repo *git.Repository, branch string, remote string, mergeBranch string) {
	t.Helper()

	branchReference := plumbing.NewBranchReferenceName(branch)
	err := repo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, branchReference))
	if err != nil {
		t.Fatalf("set HEAD reference: %v", err)
	}

	cfg, err := repo.Config()
	if err != nil {
		t.Fatalf("read repository config: %v", err)
	}

	cfg.Branches[branch] = &config.Branch{
		Name:   branch,
		Remote: remote,
		Merge:  plumbing.NewBranchReferenceName(mergeBranch),
	}

	if err := repo.SetConfig(cfg); err != nil {
		t.Fatalf("set repository config: %v", err)
	}
}
