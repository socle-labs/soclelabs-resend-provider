package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ContactPropertyDataSource{}

type ContactPropertyDataSource struct {
	client *ResendClient
}

type ContactPropertyDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Key           types.String `tfsdk:"key"`
	Type          types.String `tfsdk:"type"`
	Description   types.String `tfsdk:"description"`
	FallbackValue types.String `tfsdk:"fallback_value"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func NewContactPropertyDataSource() datasource.DataSource {
	return &ContactPropertyDataSource{}
}

func (d *ContactPropertyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact_property"
}

func (d *ContactPropertyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Resend contact property by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The contact property ID.",
				Required:    true,
			},
			"key": schema.StringAttribute{
				Description: "The property key.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The property type: string, number, or boolean.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The property description.",
				Computed:    true,
			},
			"fallback_value": schema.StringAttribute{
				Description: "The fallback value.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *ContactPropertyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContactPropertyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ContactPropertyDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	prop, err := d.client.GetContactProperty(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading contact property", err.Error())
		return
	}
	if prop == nil {
		resp.Diagnostics.AddError("Contact property not found",
			fmt.Sprintf("No contact property found with ID %s", config.ID.ValueString()))
		return
	}

	config.Key = types.StringValue(prop.Key)
	config.Type = types.StringValue(prop.Type)
	config.Description = types.StringValue(prop.Description)
	config.FallbackValue = types.StringValue(prop.FallbackValue)
	config.CreatedAt = types.StringValue(prop.CreatedAt)

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
