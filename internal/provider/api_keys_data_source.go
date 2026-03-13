package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &APIKeysDataSource{}

type APIKeysDataSource struct {
	client *ResendClient
}

type APIKeyDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
}

type APIKeysDataSourceModel struct {
	APIKeys []APIKeyDataSourceModel `tfsdk:"api_keys"`
}

func NewAPIKeysDataSource() datasource.DataSource {
	return &APIKeysDataSource{}
}

func (d *APIKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_keys"
}

func (d *APIKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Resend API keys.",
		Attributes: map[string]schema.Attribute{
			"api_keys": schema.ListNestedAttribute{
				Description: "List of API keys.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The API key ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The API key name.",
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

func (d *APIKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *APIKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.client.ListAPIKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing API keys", err.Error())
		return
	}

	var state APIKeysDataSourceModel
	for _, k := range keys.Data {
		state.APIKeys = append(state.APIKeys, APIKeyDataSourceModel{
			ID:        types.StringValue(k.ID),
			Name:      types.StringValue(k.Name),
			CreatedAt: types.StringValue(k.CreatedAt),
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
