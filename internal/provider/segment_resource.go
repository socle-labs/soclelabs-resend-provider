package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &SegmentResource{}
	_ resource.ResourceWithImportState = &SegmentResource{}
)

type SegmentResource struct {
	client *ResendClient
}

type SegmentResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func NewSegmentResource() resource.Resource {
	return &SegmentResource{}
}

func (r *SegmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment"
}

func (r *SegmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Resend segment (formerly audience) for organizing contacts.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The segment ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The segment name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SegmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ResendClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ResendClient, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *SegmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SegmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := CreateSegmentRequest{
		Name: plan.Name.ValueString(),
	}

	result, err := r.client.CreateSegment(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating segment", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Name = types.StringValue(result.Name)
	if result.CreatedAt != "" {
		plan.CreatedAt = types.StringValue(result.CreatedAt)
	} else {
		plan.CreatedAt = types.StringValue("")
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *SegmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SegmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	segment, err := r.client.GetSegment(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading segment", err.Error())
		return
	}
	if segment == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(segment.Name)
	state.CreatedAt = types.StringValue(segment.CreatedAt)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *SegmentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Segments cannot be updated — name requires replacement.
	resp.Diagnostics.AddError("Update not supported", "Segments cannot be updated. Changes require replacement.")
}

func (r *SegmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SegmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSegment(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting segment", err.Error())
	}
}

func (r *SegmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
