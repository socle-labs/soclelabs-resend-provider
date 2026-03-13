package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SegmentDataSource{}

type SegmentDataSource struct {
	client *ResendClient
}

type SegmentDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewSegmentDataSource() datasource.DataSource {
	return &SegmentDataSource{}
}

func (d *SegmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment"
}

func (d *SegmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Resend segment by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The segment ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The segment name.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *SegmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SegmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SegmentDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	segment, err := d.client.GetSegment(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading segment", err.Error())
		return
	}
	if segment == nil {
		resp.Diagnostics.AddError("Segment not found",
			fmt.Sprintf("No segment found with ID %s", config.ID.ValueString()))
		return
	}

	config.Name = types.StringValue(segment.Name)
	config.CreatedAt = types.StringValue(segment.CreatedAt)

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
