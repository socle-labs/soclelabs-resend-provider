package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ContactResource{}
	_ resource.ResourceWithImportState = &ContactResource{}
)

type ContactResource struct {
	client *ResendClient
}

type ContactResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Email        types.String `tfsdk:"email"`
	FirstName    types.String `tfsdk:"first_name"`
	LastName     types.String `tfsdk:"last_name"`
	Unsubscribed types.Bool   `tfsdk:"unsubscribed"`
	Properties   types.Map    `tfsdk:"properties"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func NewContactResource() resource.Resource {
	return &ContactResource{}
}

func (r *ContactResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact"
}

func (r *ContactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Resend contact.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The contact ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Description: "The email address of the contact.",
				Required:    true,
			},
			"first_name": schema.StringAttribute{
				Description: "The first name of the contact.",
				Optional:    true,
				Computed:    true,
			},
			"last_name": schema.StringAttribute{
				Description: "The last name of the contact.",
				Optional:    true,
				Computed:    true,
			},
			"unsubscribed": schema.BoolAttribute{
				Description: "Whether the contact is globally unsubscribed from all broadcasts.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"properties": schema.MapAttribute{
				Description: "A map of custom property keys and values.",
				Optional:    true,
				ElementType: types.StringType,
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

func (r *ContactResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ContactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ContactResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := CreateContactRequest{
		Email:        plan.Email.ValueString(),
		Unsubscribed: plan.Unsubscribed.ValueBool(),
	}
	if !plan.FirstName.IsNull() && !plan.FirstName.IsUnknown() {
		createReq.FirstName = plan.FirstName.ValueString()
	}
	if !plan.LastName.IsNull() && !plan.LastName.IsUnknown() {
		createReq.LastName = plan.LastName.ValueString()
	}
	if !plan.Properties.IsNull() && !plan.Properties.IsUnknown() {
		props := make(map[string]string)
		diags = plan.Properties.ElementsAs(ctx, &props, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Properties = props
	}

	result, err := r.client.CreateContact(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating contact", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)

	// Read back full state
	contact, err := r.client.GetContact(ctx, result.ID)
	if err == nil && contact != nil {
		plan.Email = types.StringValue(contact.Email)
		plan.FirstName = types.StringValue(contact.FirstName)
		plan.LastName = types.StringValue(contact.LastName)
		plan.Unsubscribed = types.BoolValue(contact.Unsubscribed)
		plan.CreatedAt = types.StringValue(contact.CreatedAt)
	} else {
		plan.CreatedAt = types.StringValue("")
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ContactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ContactResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	contact, err := r.client.GetContact(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading contact", err.Error())
		return
	}
	if contact == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Email = types.StringValue(contact.Email)
	state.FirstName = types.StringValue(contact.FirstName)
	state.LastName = types.StringValue(contact.LastName)
	state.Unsubscribed = types.BoolValue(contact.Unsubscribed)
	state.CreatedAt = types.StringValue(contact.CreatedAt)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ContactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ContactResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ContactResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := UpdateContactRequest{}

	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		updateReq.Email = plan.Email.ValueString()
	}
	if !plan.FirstName.IsNull() && !plan.FirstName.IsUnknown() {
		updateReq.FirstName = plan.FirstName.ValueString()
	}
	if !plan.LastName.IsNull() && !plan.LastName.IsUnknown() {
		updateReq.LastName = plan.LastName.ValueString()
	}
	if !plan.Unsubscribed.IsNull() && !plan.Unsubscribed.IsUnknown() {
		v := plan.Unsubscribed.ValueBool()
		updateReq.Unsubscribed = &v
	}
	if !plan.Properties.IsNull() && !plan.Properties.IsUnknown() {
		props := make(map[string]string)
		diags = plan.Properties.ElementsAs(ctx, &props, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Properties = props
	}

	if err := r.client.UpdateContact(ctx, state.ID.ValueString(), updateReq); err != nil {
		resp.Diagnostics.AddError("Error updating contact", err.Error())
		return
	}

	// Read back
	contact, err := r.client.GetContact(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading contact after update", err.Error())
		return
	}

	plan.ID = state.ID
	plan.CreatedAt = state.CreatedAt
	if contact != nil {
		plan.Email = types.StringValue(contact.Email)
		plan.FirstName = types.StringValue(contact.FirstName)
		plan.LastName = types.StringValue(contact.LastName)
		plan.Unsubscribed = types.BoolValue(contact.Unsubscribed)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ContactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ContactResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteContact(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting contact", err.Error())
	}
}

func (r *ContactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
