package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ContactDataSource{}

type ContactDataSource struct {
	client *ResendClient
}

type ContactDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Email        types.String `tfsdk:"email"`
	FirstName    types.String `tfsdk:"first_name"`
	LastName     types.String `tfsdk:"last_name"`
	Unsubscribed types.Bool   `tfsdk:"unsubscribed"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func NewContactDataSource() datasource.DataSource {
	return &ContactDataSource{}
}

func (d *ContactDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact"
}

func (d *ContactDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Resend contact by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The contact ID.",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "The contact email address.",
				Computed:    true,
			},
			"first_name": schema.StringAttribute{
				Description: "The first name.",
				Computed:    true,
			},
			"last_name": schema.StringAttribute{
				Description: "The last name.",
				Computed:    true,
			},
			"unsubscribed": schema.BoolAttribute{
				Description: "Whether the contact is globally unsubscribed.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (d *ContactDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContactDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ContactDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	contact, err := d.client.GetContact(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading contact", err.Error())
		return
	}
	if contact == nil {
		resp.Diagnostics.AddError("Contact not found",
			fmt.Sprintf("No contact found with ID %s", config.ID.ValueString()))
		return
	}

	config.Email = types.StringValue(contact.Email)
	config.FirstName = types.StringValue(contact.FirstName)
	config.LastName = types.StringValue(contact.LastName)
	config.Unsubscribed = types.BoolValue(contact.Unsubscribed)
	config.CreatedAt = types.StringValue(contact.CreatedAt)

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
