package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &TopicResource{}
	_ resource.ResourceWithImportState = &TopicResource{}
)

type TopicResource struct {
	client *ResendClient
}

type TopicResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	DefaultSubscription types.String `tfsdk:"default_subscription"`
	Visibility          types.String `tfsdk:"visibility"`
	CreatedAt           types.String `tfsdk:"created_at"`
}

func NewTopicResource() resource.Resource {
	return &TopicResource{}
}

func (r *TopicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic"
}

func (r *TopicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Resend topic for email subscription preferences.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The topic ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The topic name. Max 50 characters.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The topic description. Max 200 characters.",
				Optional:    true,
				Computed:    true,
			},
			"default_subscription": schema.StringAttribute{
				Description: "Default subscription state for new contacts: opt_in or opt_out.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("opt_in"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"visibility": schema.StringAttribute{
				Description: "Visibility on the unsubscribe page: public or private.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("public"),
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

func (r *TopicResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TopicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TopicResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := CreateTopicRequest{
		Name: plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createReq.Description = plan.Description.ValueString()
	}
	if !plan.DefaultSubscription.IsNull() && !plan.DefaultSubscription.IsUnknown() {
		createReq.DefaultSubscription = plan.DefaultSubscription.ValueString()
	}
	if !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		createReq.Visibility = plan.Visibility.ValueString()
	}

	result, err := r.client.CreateTopic(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating topic", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)

	// Read back full state
	topic, err := r.client.GetTopic(ctx, result.ID)
	if err == nil && topic != nil {
		plan.Name = types.StringValue(topic.Name)
		plan.Description = types.StringValue(topic.Description)
		plan.DefaultSubscription = types.StringValue(topic.DefaultSubscription)
		plan.Visibility = types.StringValue(topic.Visibility)
		plan.CreatedAt = types.StringValue(topic.CreatedAt)
	} else {
		plan.CreatedAt = types.StringValue("")
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *TopicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TopicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := r.client.GetTopic(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading topic", err.Error())
		return
	}
	if topic == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(topic.Name)
	state.Description = types.StringValue(topic.Description)
	state.DefaultSubscription = types.StringValue(topic.DefaultSubscription)
	state.Visibility = types.StringValue(topic.Visibility)
	state.CreatedAt = types.StringValue(topic.CreatedAt)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *TopicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TopicResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TopicResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := UpdateTopicRequest{}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		updateReq.Name = plan.Name.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		updateReq.Description = plan.Description.ValueString()
	}
	if !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		updateReq.Visibility = plan.Visibility.ValueString()
	}

	if err := r.client.UpdateTopic(ctx, state.ID.ValueString(), updateReq); err != nil {
		resp.Diagnostics.AddError("Error updating topic", err.Error())
		return
	}

	// Read back
	topic, err := r.client.GetTopic(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading topic after update", err.Error())
		return
	}

	plan.ID = state.ID
	plan.CreatedAt = state.CreatedAt
	if topic != nil {
		plan.Name = types.StringValue(topic.Name)
		plan.Description = types.StringValue(topic.Description)
		plan.DefaultSubscription = types.StringValue(topic.DefaultSubscription)
		plan.Visibility = types.StringValue(topic.Visibility)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *TopicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TopicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTopic(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting topic", err.Error())
	}
}

func (r *TopicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
