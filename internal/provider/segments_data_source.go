package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SegmentsDataSource{}

type SegmentsDataSource struct {
	client *ResendClient
}

type SegmentsDataSourceModel struct {
	Segments []SegmentDataSourceModel `tfsdk:"segments"`
}

func NewSegmentsDataSource() datasource.DataSource {
	return &SegmentsDataSource{}
}

func (d *SegmentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segments"
}

func (d *SegmentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Resend segments.",
		Attributes: map[string]schema.Attribute{
			"segments": schema.ListNestedAttribute{
				Description: "List of segments.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The segment ID.",
							Computed:    true,
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
				},
			},
		},
	}
}

func (d *SegmentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SegmentsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	segments, err := d.client.ListSegments(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing segments", err.Error())
		return
	}

	var state SegmentsDataSourceModel
	for _, s := range segments {
		state.Segments = append(state.Segments, SegmentDataSourceModel{
			ID:        types.StringValue(s.ID),
			Name:      types.StringValue(s.Name),
			CreatedAt: types.StringValue(s.CreatedAt),
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
