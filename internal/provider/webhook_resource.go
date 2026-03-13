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
	_ resource.Resource                = &WebhookResource{}
	_ resource.ResourceWithImportState = &WebhookResource{}
)

type WebhookResource struct {
	client *ResendClient
}

type WebhookResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Endpoint      types.String `tfsdk:"endpoint"`
	Events        types.List   `tfsdk:"events"`
	SigningSecret  types.String `tfsdk:"signing_secret"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

func (r *WebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Resend webhook for receiving email event notifications.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The webhook ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"endpoint": schema.StringAttribute{
				Description: "The URL where webhook events will be sent.",
				Required:    true,
			},
			"events": schema.ListAttribute{
				Description: "Array of event types to subscribe to (e.g. email.sent, email.delivered, email.bounced, email.complained, email.opened, email.clicked, email.received).",
				Required:    true,
				ElementType: types.StringType,
			},
			"signing_secret": schema.StringAttribute{
				Description: "The webhook signing secret. Only available after creation.",
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

func (r *WebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var events []string
	diags = plan.Events.ElementsAs(ctx, &events, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := CreateWebhookRequest{
		Endpoint: plan.Endpoint.ValueString(),
		Events:   events,
	}

	result, err := r.client.CreateWebhook(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.SigningSecret = types.StringValue(result.SigningSecret)

	// Read back to get created_at
	webhook, err := r.client.GetWebhook(ctx, result.ID)
	if err == nil && webhook != nil {
		plan.CreatedAt = types.StringValue(webhook.CreatedAt)
	} else {
		plan.CreatedAt = types.StringValue("")
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := r.client.GetWebhook(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook", err.Error())
		return
	}
	if webhook == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Endpoint = types.StringValue(webhook.Endpoint)
	state.CreatedAt = types.StringValue(webhook.CreatedAt)

	eventValues := make([]types.String, len(webhook.Events))
	for i, e := range webhook.Events {
		eventValues[i] = types.StringValue(e)
	}
	eventsList, diag := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	resp.Diagnostics.Append(diag...)
	state.Events = eventsList

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WebhookResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var events []string
	diags = plan.Events.ElementsAs(ctx, &events, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := UpdateWebhookRequest{
		Endpoint: plan.Endpoint.ValueString(),
		Events:   events,
	}

	if err := r.client.UpdateWebhook(ctx, state.ID.ValueString(), updateReq); err != nil {
		resp.Diagnostics.AddError("Error updating webhook", err.Error())
		return
	}

	plan.ID = state.ID
	plan.SigningSecret = state.SigningSecret
	plan.CreatedAt = state.CreatedAt

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteWebhook(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting webhook", err.Error())
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
