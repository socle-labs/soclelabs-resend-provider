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
	_ resource.Resource                = &ContactPropertyResource{}
	_ resource.ResourceWithImportState = &ContactPropertyResource{}
)

type ContactPropertyResource struct {
	client *ResendClient
}

type ContactPropertyResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Key           types.String `tfsdk:"key"`
	Type          types.String `tfsdk:"type"`
	Description   types.String `tfsdk:"description"`
	FallbackValue types.String `tfsdk:"fallback_value"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func NewContactPropertyResource() resource.Resource {
	return &ContactPropertyResource{}
}

func (r *ContactPropertyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact_property"
}

func (r *ContactPropertyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Resend contact property (custom attribute for contacts).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The contact property ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Description: "The property key. Max 50 characters. Only alphanumeric characters and underscores.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The property type: string, number, or boolean.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the property.",
				Optional:    true,
				Computed:    true,
			},
			"fallback_value": schema.StringAttribute{
				Description: "The default value used when a contact does not have this property set.",
				Optional:    true,
				Computed:    true,
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

func (r *ContactPropertyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ContactPropertyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ContactPropertyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := CreateContactPropertyRequest{
		Key:  plan.Key.ValueString(),
		Type: plan.Type.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}
	if !plan.FallbackValue.IsNull() && !plan.FallbackValue.IsUnknown() {
		createReq.FallbackValue = plan.FallbackValue.ValueString()
	}

	result, err := r.client.CreateContactProperty(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating contact property", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)

	// Read back full state
	prop, err := r.client.GetContactProperty(ctx, result.ID)
	if err == nil && prop != nil {
		plan.Key = types.StringValue(prop.Key)
		plan.Type = types.StringValue(prop.Type)
		plan.Description = types.StringValue(prop.Description)
		plan.FallbackValue = types.StringValue(prop.FallbackValue)
		plan.CreatedAt = types.StringValue(prop.CreatedAt)
	} else {
		plan.CreatedAt = types.StringValue("")
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ContactPropertyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ContactPropertyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	prop, err := r.client.GetContactProperty(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading contact property", err.Error())
		return
	}
	if prop == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Key = types.StringValue(prop.Key)
	state.Type = types.StringValue(prop.Type)
	state.Description = types.StringValue(prop.Description)
	state.FallbackValue = types.StringValue(prop.FallbackValue)
	state.CreatedAt = types.StringValue(prop.CreatedAt)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ContactPropertyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ContactPropertyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ContactPropertyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := UpdateContactPropertyRequest{}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		updateReq.Description = plan.Description.ValueString()
	}
	if !plan.FallbackValue.IsNull() && !plan.FallbackValue.IsUnknown() {
		updateReq.FallbackValue = plan.FallbackValue.ValueString()
	}

	if err := r.client.UpdateContactProperty(ctx, state.ID.ValueString(), updateReq); err != nil {
		resp.Diagnostics.AddError("Error updating contact property", err.Error())
		return
	}

	// Read back
	prop, err := r.client.GetContactProperty(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading contact property after update", err.Error())
		return
	}

	plan.ID = state.ID
	plan.CreatedAt = state.CreatedAt
	plan.Key = state.Key
	plan.Type = state.Type
	if prop != nil {
		plan.Description = types.StringValue(prop.Description)
		plan.FallbackValue = types.StringValue(prop.FallbackValue)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ContactPropertyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ContactPropertyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteContactProperty(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting contact property", err.Error())
	}
}

func (r *ContactPropertyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
