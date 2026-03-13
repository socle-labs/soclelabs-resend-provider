package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &WebhooksDataSource{}

type WebhooksDataSource struct {
	client *ResendClient
}

type WebhooksDataSourceItemModel struct {
	ID        types.String `tfsdk:"id"`
	Endpoint  types.String `tfsdk:"endpoint"`
	Events    types.List   `tfsdk:"events"`
	CreatedAt types.String `tfsdk:"created_at"`
}

type WebhooksDataSourceModel struct {
	Webhooks []WebhooksDataSourceItemModel `tfsdk:"webhooks"`
}

func NewWebhooksDataSource() datasource.DataSource {
	return &WebhooksDataSource{}
}

func (d *WebhooksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhooks"
}

func (d *WebhooksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Resend webhooks.",
		Attributes: map[string]schema.Attribute{
			"webhooks": schema.ListNestedAttribute{
				Description: "List of webhooks.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The webhook ID.",
							Computed:    true,
						},
						"endpoint": schema.StringAttribute{
							Description: "The webhook endpoint URL.",
							Computed:    true,
						},
						"events": schema.ListAttribute{
							Description: "Subscribed event types.",
							Computed:    true,
							ElementType: types.StringType,
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

func (d *WebhooksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WebhooksDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	webhooks, err := d.client.ListWebhooks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing webhooks", err.Error())
		return
	}

	var state WebhooksDataSourceModel
	for _, w := range webhooks {
		eventsList, diag := types.ListValueFrom(ctx, types.StringType, w.Events)
		resp.Diagnostics.Append(diag...)

		state.Webhooks = append(state.Webhooks, WebhooksDataSourceItemModel{
			ID:        types.StringValue(w.ID),
			Endpoint:  types.StringValue(w.Endpoint),
			Events:    eventsList,
			CreatedAt: types.StringValue(w.CreatedAt),
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
