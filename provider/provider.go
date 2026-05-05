package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &gitProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &gitProvider{
			version: version,
		}
	}
}

type gitProvider struct {
	version string
}

func (p *gitProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "git"
	resp.Version = p.version
}

func (p *gitProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Git provider exposes information about local Git repositories.",
	}
}

func (p *gitProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *gitProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRepositoryDataSource,
	}
}

func (p *gitProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}
