package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TopicDataSource{}

type TopicDataSource struct {
	client *ResendClient
}

type TopicDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	DefaultSubscription types.String `tfsdk:"default_subscription"`
	Visibility          types.String `tfsdk:"visibility"`
	CreatedAt           types.String `tfsdk:"created_at"`
}

func NewTopicDataSource() datasource.DataSource {
	return &TopicDataSource{}
}

func (d *TopicDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic"
}

func (d *TopicDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Resend topic by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The topic ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The topic name.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The topic description.",
				Computed:    true,
			},
			"default_subscription": schema.StringAttribute{
				Description: "Default subscription state: opt_in or opt_out.",
				Computed:    true,
			},
			"visibility": schema.StringAttribute{
				Description: "Visibility: public or private.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *TopicDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TopicDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TopicDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := d.client.GetTopic(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading topic", err.Error())
		return
	}
	if topic == nil {
		resp.Diagnostics.AddError("Topic not found",
			fmt.Sprintf("No topic found with ID %s", config.ID.ValueString()))
		return
	}

	config.Name = types.StringValue(topic.Name)
	config.Description = types.StringValue(topic.Description)
	config.DefaultSubscription = types.StringValue(topic.DefaultSubscription)
	config.Visibility = types.StringValue(topic.Visibility)
	config.CreatedAt = types.StringValue(topic.CreatedAt)

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
