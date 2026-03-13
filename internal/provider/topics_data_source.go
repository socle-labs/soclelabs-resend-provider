package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TopicsDataSource{}

type TopicsDataSource struct {
	client *ResendClient
}

type TopicsDataSourceModel struct {
	Topics []TopicDataSourceModel `tfsdk:"topics"`
}

func NewTopicsDataSource() datasource.DataSource {
	return &TopicsDataSource{}
}

func (d *TopicsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topics"
}

func (d *TopicsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Resend topics.",
		Attributes: map[string]schema.Attribute{
			"topics": schema.ListNestedAttribute{
				Description: "List of topics.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The topic ID.",
							Computed:    true,
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
							Description: "Default subscription state.",
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
				},
			},
		},
	}
}

func (d *TopicsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TopicsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	topics, err := d.client.ListTopics(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing topics", err.Error())
		return
	}

	var state TopicsDataSourceModel
	for _, t := range topics {
		state.Topics = append(state.Topics, TopicDataSourceModel{
			ID:                  types.StringValue(t.ID),
			Name:                types.StringValue(t.Name),
			Description:         types.StringValue(t.Description),
			DefaultSubscription: types.StringValue(t.DefaultSubscription),
			Visibility:          types.StringValue(t.Visibility),
			CreatedAt:           types.StringValue(t.CreatedAt),
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
