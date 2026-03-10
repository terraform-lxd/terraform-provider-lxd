package auth

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type AuthGroupDataSourceModel struct {
	Name types.String `tfsdk:"name"`

	// Computed.
	Description types.String      `tfsdk:"description"`
	Permissions []PermissionModel `tfsdk:"permissions"`
}

// AuthGroupDataSource reads LXD auth groups.
type AuthGroupDataSource struct {
	provider *provider_config.LxdProviderConfig
}

// NewAuthGroupDataSource returns a new AuthGroupDataSource.
func NewAuthGroupDataSource() datasource.DataSource {
	return &AuthGroupDataSource{}
}

func (r AuthGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_group"
}

func (r AuthGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},

			// Computed.

			"description": schema.StringAttribute{
				Computed: true,
			},

			"permissions": schema.SetNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"entitlement": schema.StringAttribute{
							Computed: true,
						},
						"entity_type": schema.StringAttribute{
							Computed: true,
						},
						"entity_args": schema.MapAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *AuthGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.LxdProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	r.provider = provider
}

func (r *AuthGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AuthGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer("", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	authGroupName := config.Name.ValueString()
	authGroup, _, err := server.GetAuthGroup(authGroupName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve auth group %q", authGroupName), err.Error())
		return
	}

	permissions, err := PermissionsFromAPI(ctx, authGroup.Permissions)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert permissions from API format", err.Error())
		return
	}

	config.Name = types.StringValue(authGroup.Name)
	config.Description = types.StringValue(authGroup.Description)
	config.Permissions = permissions

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
