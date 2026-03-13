package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &WebhookDataSource{}

type WebhookDataSource struct {
	client *ResendClient
}

type WebhookDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Endpoint  types.String `tfsdk:"endpoint"`
	Events    types.List   `tfsdk:"events"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewWebhookDataSource() datasource.DataSource {
	return &WebhookDataSource{}
}

func (d *WebhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (d *WebhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Resend webhook by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The webhook ID.",
				Required:    true,
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
	}
}

func (d *WebhookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WebhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WebhookDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := d.client.GetWebhook(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook", err.Error())
		return
	}
	if webhook == nil {
		resp.Diagnostics.AddError("Webhook not found",
			fmt.Sprintf("No webhook found with ID %s", config.ID.ValueString()))
		return
	}

	config.Endpoint = types.StringValue(webhook.Endpoint)
	config.CreatedAt = types.StringValue(webhook.CreatedAt)

	eventsList, diag := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	resp.Diagnostics.Append(diag...)
	config.Events = eventsList

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
