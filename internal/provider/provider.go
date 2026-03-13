package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ResendProvider{}

type ResendProvider struct {
	version string
}

type ResendProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ResendProvider{
			version: version,
		}
	}
}

func (p *ResendProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "resend"
	resp.Version = p.version
}

func (p *ResendProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Resend email platform.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "Resend API key. Can also be set via the RESEND_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *ResendProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ResendProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("RESEND_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The Resend API key must be set in the provider configuration or via the RESEND_API_KEY environment variable.",
		)
		return
	}

	client := NewResendClient(apiKey)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ResendProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainResource,
		NewAPIKeyResource,
		NewWebhookResource,
		NewContactResource,
		NewSegmentResource,
		NewTopicResource,
		NewContactPropertyResource,
	}
}

func (p *ResendProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDomainDataSource,
		NewDomainsDataSource,
		NewAPIKeysDataSource,
		NewWebhookDataSource,
		NewWebhooksDataSource,
		NewContactDataSource,
		NewContactsDataSource,
		NewSegmentDataSource,
		NewSegmentsDataSource,
		NewTopicDataSource,
		NewTopicsDataSource,
		NewContactPropertyDataSource,
		NewContactPropertiesDataSource,
	}
}
