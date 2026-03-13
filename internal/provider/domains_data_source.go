package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DomainsDataSource{}

type DomainsDataSource struct {
	client *ResendClient
}

type DomainsDataSourceModel struct {
	Domains []DomainDataSourceModel `tfsdk:"domains"`
}

func NewDomainsDataSource() datasource.DataSource {
	return &DomainsDataSource{}
}

func (d *DomainsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

func (d *DomainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Resend domains.",
		Attributes: map[string]schema.Attribute{
			"domains": schema.ListNestedAttribute{
				Description: "List of domains.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The domain ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The domain name.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The domain verification status.",
							Computed:    true,
						},
						"region": schema.StringAttribute{
							Description: "The region.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DomainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ResendClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *ResendClient, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *DomainsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	domains, err := d.client.ListDomains(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing domains", err.Error())
		return
	}

	var state DomainsDataSourceModel
	for _, domain := range domains {
		state.Domains = append(state.Domains, DomainDataSourceModel{
			ID:        types.StringValue(domain.ID),
			Name:      types.StringValue(domain.Name),
			Status:    types.StringValue(domain.Status),
			Region:    types.StringValue(domain.Region),
			CreatedAt: types.StringValue(domain.CreatedAt),
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
