package auth

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type AuthIdentityDataSourceModel struct {
	Name       types.String `tfsdk:"name"`
	AuthMethod types.String `tfsdk:"auth_method"`
	Remote     types.String `tfsdk:"remote"`

	// Computed.
	Identifier  types.String `tfsdk:"identifier"`
	Groups      types.Set    `tfsdk:"groups"`
	Certificate types.String `tfsdk:"tls_certificate"`
}

// AuthIdentityDataSource reads LXD identities.
type AuthIdentityDataSource struct {
	provider *provider_config.LxdProviderConfig
}

// NewAuthIdentityDataSource returns a new [AuthIdentityDataSource].
func NewAuthIdentityDataSource() datasource.DataSource {
	return &AuthIdentityDataSource{}
}

func (r AuthIdentityDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_identity"
}

func (r AuthIdentityDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},

			"auth_method": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("tls", "bearer", "oidc"),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			// Computed.

			"groups": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},

			"tls_certificate": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},

			"identifier": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *AuthIdentityDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *AuthIdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AuthIdentityDataSourceModel
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

	identityName := config.Name.ValueString()
	identityAuthMethod := config.AuthMethod.ValueString()
	identity, _, err := server.GetIdentity(identityAuthMethod, identityName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve %q identity %q", identityAuthMethod, identityName), err.Error())
		return
	}

	groups, diags := common.ToStringSetType(ctx, identity.Groups)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	config.Name = types.StringValue(identity.Name)
	config.AuthMethod = types.StringValue(identity.AuthenticationMethod)
	config.Identifier = types.StringValue(identity.Identifier)
	config.Certificate = types.StringValue(identity.TLSCertificate)
	config.Groups = groups

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
