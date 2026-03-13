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
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

type DomainResource struct {
	client *ResendClient
}

type DomainResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Region           types.String `tfsdk:"region"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
	CustomReturnPath types.String `tfsdk:"custom_return_path"`
	OpenTracking     types.Bool   `tfsdk:"open_tracking"`
	ClickTracking    types.Bool   `tfsdk:"click_tracking"`
	TLS              types.String `tfsdk:"tls"`
}

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Resend domain for sending emails.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The domain ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The domain name (e.g. example.com).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: "The region where emails will be sent from. Possible values: us-east-1, eu-west-1, sa-east-1, ap-northeast-1.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The domain verification status.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"custom_return_path": schema.StringAttribute{
				Description: "Custom subdomain for the Return-Path address. Defaults to 'send'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"open_tracking": schema.BoolAttribute{
				Description: "Track the open rate of each email.",
				Optional:    true,
				Computed:    true,
			},
			"click_tracking": schema.BoolAttribute{
				Description: "Track clicks within the body of each HTML email.",
				Optional:    true,
				Computed:    true,
			},
			"tls": schema.StringAttribute{
				Description: "TLS setting. Possible values: enforced, opportunistic.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := CreateDomainRequest{
		Name: plan.Name.ValueString(),
	}
	if !plan.Region.IsNull() && !plan.Region.IsUnknown() {
		createReq.Region = plan.Region.ValueString()
	}
	if !plan.CustomReturnPath.IsNull() && !plan.CustomReturnPath.IsUnknown() {
		createReq.CustomReturnPath = plan.CustomReturnPath.ValueString()
	}

	result, err := r.client.CreateDomain(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Name = types.StringValue(result.Name)
	plan.Status = types.StringValue(result.Status)
	plan.CreatedAt = types.StringValue(result.CreatedAt)
	if result.Region != "" {
		plan.Region = types.StringValue(result.Region)
	}
	if result.CustomReturnPath != "" {
		plan.CustomReturnPath = types.StringValue(result.CustomReturnPath)
	}

	// Apply tracking settings if specified
	if (!plan.OpenTracking.IsNull() && !plan.OpenTracking.IsUnknown()) ||
		(!plan.ClickTracking.IsNull() && !plan.ClickTracking.IsUnknown()) ||
		(!plan.TLS.IsNull() && !plan.TLS.IsUnknown()) {
		updateReq := UpdateDomainRequest{}
		if !plan.OpenTracking.IsNull() && !plan.OpenTracking.IsUnknown() {
			v := plan.OpenTracking.ValueBool()
			updateReq.OpenTracking = &v
		}
		if !plan.ClickTracking.IsNull() && !plan.ClickTracking.IsUnknown() {
			v := plan.ClickTracking.ValueBool()
			updateReq.ClickTracking = &v
		}
		if !plan.TLS.IsNull() && !plan.TLS.IsUnknown() {
			updateReq.TLS = plan.TLS.ValueString()
		}
		if err := r.client.UpdateDomain(ctx, result.ID, updateReq); err != nil {
			resp.Diagnostics.AddError("Error updating domain settings", err.Error())
			return
		}
	}

	// Read back the full state
	domain, err := r.client.GetDomain(ctx, result.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain after create", err.Error())
		return
	}
	if domain != nil {
		plan.Status = types.StringValue(domain.Status)
		plan.Region = types.StringValue(domain.Region)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetDomain(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}
	if domain == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(domain.Name)
	state.Status = types.StringValue(domain.Status)
	state.CreatedAt = types.StringValue(domain.CreatedAt)
	state.Region = types.StringValue(domain.Region)
	if domain.CustomReturnPath != "" {
		state.CustomReturnPath = types.StringValue(domain.CustomReturnPath)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DomainResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := UpdateDomainRequest{}
	if !plan.OpenTracking.IsNull() && !plan.OpenTracking.IsUnknown() {
		v := plan.OpenTracking.ValueBool()
		updateReq.OpenTracking = &v
	}
	if !plan.ClickTracking.IsNull() && !plan.ClickTracking.IsUnknown() {
		v := plan.ClickTracking.ValueBool()
		updateReq.ClickTracking = &v
	}
	if !plan.TLS.IsNull() && !plan.TLS.IsUnknown() {
		updateReq.TLS = plan.TLS.ValueString()
	}

	if err := r.client.UpdateDomain(ctx, state.ID.ValueString(), updateReq); err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	// Read back
	domain, err := r.client.GetDomain(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain after update", err.Error())
		return
	}

	plan.ID = state.ID
	plan.CreatedAt = state.CreatedAt
	if domain != nil {
		plan.Status = types.StringValue(domain.Status)
		plan.Region = types.StringValue(domain.Region)
		plan.Name = types.StringValue(domain.Name)
		if domain.CustomReturnPath != "" {
			plan.CustomReturnPath = types.StringValue(domain.CustomReturnPath)
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteDomain(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting domain", err.Error())
	}
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
