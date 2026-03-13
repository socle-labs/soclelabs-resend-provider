package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ContactsDataSource{}

type ContactsDataSource struct {
	client *ResendClient
}

type ContactsDataSourceModel struct {
	Contacts []ContactDataSourceModel `tfsdk:"contacts"`
}

func NewContactsDataSource() datasource.DataSource {
	return &ContactsDataSource{}
}

func (d *ContactsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contacts"
}

func (d *ContactsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all Resend contacts.",
		Attributes: map[string]schema.Attribute{
			"contacts": schema.ListNestedAttribute{
				Description: "List of contacts.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The contact ID.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "The email address.",
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
				},
			},
		},
	}
}

func (d *ContactsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContactsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	contacts, err := d.client.ListContacts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing contacts", err.Error())
		return
	}

	var state ContactsDataSourceModel
	for _, c := range contacts {
		state.Contacts = append(state.Contacts, ContactDataSourceModel{
			ID:           types.StringValue(c.ID),
			Email:        types.StringValue(c.Email),
			FirstName:    types.StringValue(c.FirstName),
			LastName:     types.StringValue(c.LastName),
			Unsubscribed: types.BoolValue(c.Unsubscribed),
			CreatedAt:    types.StringValue(c.CreatedAt),
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
