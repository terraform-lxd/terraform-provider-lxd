package auth

import (
	"context"
	"fmt"
	"time"

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
	Type       types.String `tfsdk:"type"`
	AuthMethod types.String `tfsdk:"auth_method"`
	Remote     types.String `tfsdk:"remote"`

	// Computed.
	Identifier  types.String `tfsdk:"identifier"`
	Groups      types.Set    `tfsdk:"groups"`
	Certificate types.String `tfsdk:"tls_certificate"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
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

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("tls", "bearer", "devlxd", "oidc"),
				},
			},

			"auth_method": schema.StringAttribute{
				Optional: true,
				Computed: true,
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

			"expires_at": schema.StringAttribute{
				Computed:    true,
				Description: "Expiry of the identity's credential, in RFC3339 format. For bearer identities this is the expiry of the token that the identity currently bears.",
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

func (r AuthIdentityDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		identityTypeValidator{warnOnAuthMethod: false},
	}
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
	identityType := config.Type.ValueString()
	identityAuthMethod := config.AuthMethod.ValueString()

	// Exactly one of the two is configured, and each derives the other. Every
	// authentication method is also an identity type.
	if identityAuthMethod == "" {
		identityAuthMethod = toAuthMethod(identityType)
	} else {
		identityType = identityAuthMethod
	}

	identity, _, err := server.GetIdentity(identityAuthMethod, identityName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve %q identity %q", identityType, identityName), err.Error())
		return
	}

	// The bearer authentication method returns client and devlxd identities
	// alike, so a lookup by identity type must reject an identity of another
	// type. LXD identity types that the provider cannot name keep the identity
	// type that was asked for.
	serverType := toType(identity.Type)
	if serverType != "" {
		if !config.Type.IsNull() && serverType != identityType {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Identity %q is not of type %q", identityName, identityType),
				fmt.Sprintf("LXD identity type %q corresponds to type %q", identity.Type, serverType),
			)
			return
		}

		identityType = serverType
	}

	groups, diags := common.ToStringSetType(ctx, identity.Groups)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	config.Name = types.StringValue(identity.Name)
	config.Type = types.StringValue(identityType)
	config.AuthMethod = types.StringValue(identity.AuthenticationMethod)
	config.Identifier = types.StringValue(identity.Identifier)
	config.Certificate = types.StringValue(identity.TLSCertificate)
	config.Groups = groups

	// The expiry is reported only for identities whose credential expires.
	config.ExpiresAt = types.StringNull()
	if identity.ExpiresAt != nil {
		config.ExpiresAt = types.StringValue(identity.ExpiresAt.UTC().Format(time.RFC3339))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
