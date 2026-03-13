package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DomainDataSource{}

type DomainDataSource struct {
	client *ResendClient
}

type DomainDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Status    types.String `tfsdk:"status"`
	Region    types.String `tfsdk:"region"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewDomainDataSource() datasource.DataSource {
	return &DomainDataSource{}
}

func (d *DomainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *DomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Resend domain by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The domain ID.",
				Required:    true,
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
				Description: "The region where emails are sent from.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *DomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := d.client.GetDomain(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}
	if domain == nil {
		resp.Diagnostics.AddError("Domain not found",
			fmt.Sprintf("No domain found with ID %s", config.ID.ValueString()))
		return
	}

	config.Name = types.StringValue(domain.Name)
	config.Status = types.StringValue(domain.Status)
	config.Region = types.StringValue(domain.Region)
	config.CreatedAt = types.StringValue(domain.CreatedAt)

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
