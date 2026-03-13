package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &APIKeyResource{}

type APIKeyResource struct {
	client *ResendClient
}

type APIKeyResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Permission types.String `tfsdk:"permission"`
	DomainID   types.String `tfsdk:"domain_id"`
	Token      types.String `tfsdk:"token"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

func (r *APIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Resend API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The API key ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The API key name. Maximum 50 characters.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission": schema.StringAttribute{
				Description: "Permission level: full_access or sending_access.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Description: "Restrict sending to a specific domain. Only used when permission is sending_access.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				Description: "The API key token. Only available after creation.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

func (r *APIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan APIKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := CreateAPIKeyRequest{
		Name: plan.Name.ValueString(),
	}
	if !plan.Permission.IsNull() && !plan.Permission.IsUnknown() {
		createReq.Permission = plan.Permission.ValueString()
	}
	if !plan.DomainID.IsNull() && !plan.DomainID.IsUnknown() {
		createReq.DomainID = plan.DomainID.ValueString()
	}

	result, err := r.client.CreateAPIKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating API key", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Token = types.StringValue(result.Token)

	// Look up created_at from list endpoint
	keys, err := r.client.ListAPIKeys(ctx)
	if err == nil && keys != nil {
		for _, k := range keys.Data {
			if k.ID == result.ID {
				plan.CreatedAt = types.StringValue(k.CreatedAt)
				break
			}
		}
	}
	if plan.CreatedAt.IsNull() || plan.CreatedAt.IsUnknown() {
		plan.CreatedAt = types.StringValue("")
	}

	if plan.Permission.IsNull() || plan.Permission.IsUnknown() {
		plan.Permission = types.StringValue("full_access")
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state APIKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Resend API doesn't have a get-by-ID endpoint for API keys.
	// We use the list endpoint and find our key.
	keys, err := r.client.ListAPIKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing API keys", err.Error())
		return
	}

	var found bool
	for _, k := range keys.Data {
		if k.ID == state.ID.ValueString() {
			state.Name = types.StringValue(k.Name)
			state.CreatedAt = types.StringValue(k.CreatedAt)
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *APIKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// API keys cannot be updated — all attributes require replacement.
	resp.Diagnostics.AddError("Update not supported", "API keys cannot be updated. All changes require replacement.")
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state APIKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAPIKey(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting API key", err.Error())
	}
}
