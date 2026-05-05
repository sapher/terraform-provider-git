package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &repositoryDataSource{}

func NewRepositoryDataSource() datasource.DataSource {
	return &repositoryDataSource{}
}

type repositoryDataSource struct{}

type repositoryDataSourceModel struct {
	Namespace           types.String `tfsdk:"namespace"`
	OriginURL           types.String `tfsdk:"origin_url"`
	DefaultRemoteBranch types.String `tfsdk:"default_remote_branch"`
}

func (d *repositoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (d *repositoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads remote-origin metadata from a local Git repository.",
		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Repository namespace and name extracted from `origin_url`, without a trailing `.git` suffix.",
			},
			"origin_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "First configured URL for the `origin` remote.",
			},
			"default_remote_branch": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Best-effort default branch for `origin`, resolved from local Git metadata when available.",
			},
		},
	}
}

func (d *repositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repositoryPath, err := os.Getwd()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read working directory", fmt.Sprintf("read current Terraform module directory: %s", err))
		return
	}

	originURL, defaultRemoteBranch, err := readRepositoryInfo(repositoryPath)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Git repository", err.Error())
		return
	}

	namespace, err := extractNamespace(originURL)
	if err != nil {
		resp.Diagnostics.AddError("Unable to extract repository namespace", err.Error())
		return
	}

	data.Namespace = types.StringValue(namespace)
	data.OriginURL = types.StringValue(originURL)
	if defaultRemoteBranch == nil {
		data.DefaultRemoteBranch = types.StringNull()
	} else {
		data.DefaultRemoteBranch = types.StringValue(*defaultRemoteBranch)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func readRepositoryInfo(repositoryPath string) (string, *string, error) {
	repo, err := git.PlainOpenWithOptions(repositoryPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", nil, fmt.Errorf("open repository at %q: %w", repositoryPath, err)
	}

	cfg, err := repo.Config()
	if err != nil {
		return "", nil, fmt.Errorf("read repository config: %w", err)
	}

	origin := cfg.Remotes["origin"]
	if origin == nil || len(origin.URLs) == 0 {
		return "", nil, fmt.Errorf("remote %q is not configured", "origin")
	}

	defaultBranch, err := readDefaultRemoteBranch(repo, cfg)
	if err != nil {
		return "", nil, err
	}

	return origin.URLs[0], defaultBranch, nil
}

func extractNamespace(originURL string) (string, error) {
	originURL = strings.TrimSpace(originURL)
	if originURL == "" {
		return "", fmt.Errorf("origin URL is empty")
	}

	var namespace string
	if strings.Contains(originURL, "://") {
		parsedURL, err := url.Parse(originURL)
		if err != nil {
			return "", fmt.Errorf("parse origin URL %q: %w", originURL, err)
		}
		if parsedURL.Host == "" {
			return "", fmt.Errorf("origin URL %q does not include a host", originURL)
		}

		namespace = strings.TrimPrefix(parsedURL.Path, "/")
	} else if separatorIndex := strings.Index(originURL, ":"); separatorIndex > 0 && strings.Contains(originURL[:separatorIndex], "@") {
		namespace = originURL[separatorIndex+1:]
	} else {
		return "", fmt.Errorf("origin URL %q is not a supported Git remote URL", originURL)
	}

	namespace = strings.Trim(namespace, "/")
	namespace = strings.TrimSuffix(namespace, ".git")
	if namespace == "" {
		return "", fmt.Errorf("origin URL %q does not include a repository namespace", originURL)
	}

	return namespace, nil
}

func readDefaultRemoteBranch(repo *git.Repository, cfg *config.Config) (*string, error) {
	remoteHead, err := repo.Reference(plumbing.NewRemoteHEADReferenceName("origin"), true)
	if err != nil {
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			return nil, fmt.Errorf("resolve %q: %w", plumbing.NewRemoteHEADReferenceName("origin"), err)
		}

		return readDefaultRemoteBranchFromUpstream(repo, cfg), nil
	}

	defaultBranch := strings.TrimPrefix(remoteHead.Name().String(), "refs/remotes/origin/")
	if target := remoteHead.Target(); target != "" {
		defaultBranch = strings.TrimPrefix(target.String(), "refs/remotes/origin/")
	}
	if defaultBranch == "" || strings.HasPrefix(defaultBranch, "refs/") {
		return nil, fmt.Errorf("remote %q HEAD does not point to a branch", "origin")
	}

	return &defaultBranch, nil
}

func readDefaultRemoteBranchFromUpstream(repo *git.Repository, cfg *config.Config) *string {
	head, err := repo.Reference(plumbing.HEAD, false)
	if err != nil || head.Target() == "" || !head.Target().IsBranch() {
		return nil
	}

	branchName := head.Target().Short()
	branch := cfg.Branches[branchName]
	if branch == nil || branch.Remote != "origin" || !branch.Merge.IsBranch() {
		return nil
	}

	defaultBranch := branch.Merge.Short()
	if defaultBranch == "" {
		return nil
	}

	return &defaultBranch
}
