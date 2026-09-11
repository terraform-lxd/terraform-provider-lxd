package auth

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type AuthIdentityProviderGroupDataSourceModel struct {
	Name   types.String `tfsdk:"name"`
	Remote types.String `tfsdk:"remote"`

	// Computed.
	Groups types.Set `tfsdk:"groups"`
}

// AuthIdentityProviderGroupDataSource reads LXD identity provider groups.
type AuthIdentityProviderGroupDataSource struct {
	provider *provider_config.LxdProviderConfig
}

// NewAuthIdentityProviderGroupDataSource returns a new AuthIdentityProviderGroupDataSource.
func NewAuthIdentityProviderGroupDataSource() datasource.DataSource {
	return &AuthIdentityProviderGroupDataSource{}
}

func (r AuthIdentityProviderGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_identity_provider_group"
}

func (r AuthIdentityProviderGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			// Computed.

			"groups": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (r *AuthIdentityProviderGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *AuthIdentityProviderGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AuthIdentityProviderGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := config.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	idpGroupName := config.Name.ValueString()
	idpGroup, _, err := server.GetIdentityProviderGroup(idpGroupName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve identity provider group %q", idpGroupName), err.Error())
		return
	}

	groups, diags := common.ToStringSetType(ctx, idpGroup.Groups)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	config.Name = types.StringValue(idpGroup.Name)
	config.Groups = groups

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
