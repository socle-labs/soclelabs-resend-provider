package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ContactPropertiesDataSource{}

type ContactPropertiesDataSource struct {
	client *ResendClient
}

type ContactPropertiesDataSourceModel struct {
	ContactProperties []ContactPropertyDataSourceModel `tfsdk:"contact_properties"`
}

func NewContactPropertiesDataSource() datasource.DataSource {
	return &ContactPropertiesDataSource{}
}

func (d *ContactPropertiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact_properties"
}

func (d *ContactPropertiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Resend contact properties.",
		Attributes: map[string]schema.Attribute{
			"contact_properties": schema.ListNestedAttribute{
				Description: "List of contact properties.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The contact property ID.",
							Computed:    true,
						},
						"key": schema.StringAttribute{
							Description: "The property key.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The property type.",
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
				},
			},
		},
	}
}

func (d *ContactPropertiesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContactPropertiesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	props, err := d.client.ListContactProperties(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing contact properties", err.Error())
		return
	}

	var state ContactPropertiesDataSourceModel
	for _, p := range props {
		state.ContactProperties = append(state.ContactProperties, ContactPropertyDataSourceModel{
			ID:            types.StringValue(p.ID),
			Key:           types.StringValue(p.Key),
			Type:          types.StringValue(p.Type),
			Description:   types.StringValue(p.Description),
			FallbackValue: types.StringValue(p.FallbackValue),
			CreatedAt:     types.StringValue(p.CreatedAt),
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
