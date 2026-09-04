package auth

import (
	"context"
	"fmt"
	"regexp"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// expiryPattern matches a space separated list of durations, as accepted by the
// LXD API, for example "30d" or "1H 30M".
var expiryPattern = regexp.MustCompile(`^\d+[SMHdwmy]( \d+[SMHdwmy])*$`)

// expiryNonZeroPattern requires at least one non-zero digit. LXD resolves an
// all-zero duration such as "0S" to the token creation time, so it issues an
// already-expired token that the provider would then recreate on every apply.
var expiryNonZeroPattern = regexp.MustCompile(`[1-9]`)

// AuthIdentityTokenModel represents the Terraform state model for a token that
// is issued for an LXD bearer identity.
type AuthIdentityTokenModel struct {
	Identity types.String `tfsdk:"identity"`
	Expiry   types.String `tfsdk:"expiry"`
	Remote   types.String `tfsdk:"remote"`

	// Computed.
	Token     types.String `tfsdk:"token"`
	ExpiresAt types.String `tfsdk:"expires_at"`
}

// AuthIdentityTokenResource manages tokens of LXD bearer identities.
type AuthIdentityTokenResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewAuthIdentityTokenResource returns a new [AuthIdentityTokenResource].
func NewAuthIdentityTokenResource() resource.Resource {
	return &AuthIdentityTokenResource{}
}

func (r AuthIdentityTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_identity_token"
}

func (r AuthIdentityTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"identity": schema.StringAttribute{
				Required:    true,
				Description: "Name of the bearer identity for which the token is issued.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"expiry": schema.StringAttribute{
				Optional:    true,
				Description: `Token expiry as a space separated list of durations in the form (\d)+(S|M|H|d|w|m|y). If not provided, the server's default expiry is used.`,
				Validators: []validator.String{
					stringvalidator.RegexMatches(expiryPattern, `Expiry must be a space separated list of durations in the form (\d)+(S|M|H|d|w|m|y)`),
					stringvalidator.RegexMatches(expiryNonZeroPattern, "Expiry must be greater than zero"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"remote": schema.StringAttribute{
				Optional:    true,
				Description: "The remote in which the token is issued. If not provided, the provider's default remote is used.",
				PlanModifiers: []planmodifier.String{
					// Force replace because resource does not implement "Update".
					stringplanmodifier.RequiresReplace(),
				},
			},

			// Computed.

			"token": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Issued bearer token.",
			},

			"expires_at": schema.StringAttribute{
				Computed:    true,
				Description: "Time at which the token expires, in RFC3339 format.",
			},
		},
	}
}

func (r *AuthIdentityTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r AuthIdentityTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthIdentityTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	identityName := plan.Identity.ValueString()

	// Bearer tokens are issued only by servers that report their expiry, which
	// is what the provider tracks the issued token by.
	err = server.CheckExtension("access_management_expiry")
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to issue token for identity %q", identityName), err.Error())
		return
	}

	tokenReq := api.IdentityBearerTokenPost{
		Expiry: plan.Expiry.ValueString(),
	}

	token, err := server.IssueBearerIdentityToken(identityName, tokenReq)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to issue token for identity %q", identityName), err.Error())
		return
	}

	// The token is issued and active on the server. Persist it into the state before the
	// expiry is read, so that a transient read failure does not leave the credential unmanaged.
	plan.Token = types.StringValue(token.Token)
	plan.ExpiresAt = types.StringNull()

	diags := plan.TaintState(ctx, &resp.State)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	expiresAt, err := identityTokenExpiry(server, identityName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve expiry of the token issued for identity %q", identityName), err.Error())
		return
	}

	plan.ExpiresAt = expiresAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// TaintState marks the state with fields required to target the issued token.
func (m AuthIdentityTokenModel) TaintState(ctx context.Context, tfState *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(tfState.SetAttribute(ctx, path.Root("identity"), m.Identity.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("remote"), m.Remote.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("token"), m.Token)...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("expires_at"), m.ExpiresAt)...)

	return diags
}

func (r AuthIdentityTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthIdentityTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	identityName := state.Identity.ValueString()

	expiresAt, err := identityTokenExpiry(server, identityName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			// Tokens of a removed identity are revoked along with the identity.
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve the token expiry of identity %q", identityName), err.Error())
		return
	}

	// The identity reports the expiry of the token it currently bears. If no
	// expiry is reported, or the reported expiry differs from the recorded one,
	// the token in the state has been revoked or replaced. An expired token is
	// dropped as well, because the server keeps reporting its expiry.
	if expiresAt != state.ExpiresAt || isExpired(expiresAt) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r AuthIdentityTokenResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// Not supported. Every attribute of this resource requires replacement.
}

func (r AuthIdentityTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthIdentityTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	identityName := state.Identity.ValueString()

	// Revoke only while the server still reports the expiry this state recorded, otherwise
	// the revoke could drop a token issued outside the Terraform provider.
	expiresAt, err := identityTokenExpiry(server, identityName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			// The identity, and with it its token, is already gone.
			return
		}

		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve the token expiry of identity %q", identityName), err.Error())
		return
	}

	if expiresAt != state.ExpiresAt {
		return
	}

	err = server.RevokeBearerIdentityToken(identityName)
	if err != nil && !errors.IsNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to revoke token of identity %q", identityName), err.Error())
		return
	}
}

// identityTokenExpiry returns the expiry of the token that the given bearer
// identity currently bears, as reported by the server. A null value is returned
// if the identity bears no token, or if its token has no expiry.
func identityTokenExpiry(server lxd.InstanceServer, identityName string) (types.String, error) {
	identity, _, err := server.GetIdentity(api.AuthenticationMethodBearer, identityName)
	if err != nil {
		return types.StringNull(), err
	}

	if identity.ExpiresAt == nil {
		return types.StringNull(), nil
	}

	return types.StringValue(identity.ExpiresAt.UTC().Format(time.RFC3339)), nil
}

// isExpired reports whether the given expiry has passed. The server keeps
// reporting the expiry of a token that has already expired.
func isExpired(expiresAt types.String) bool {
	if expiresAt.IsNull() {
		return false
	}

	expiry, err := time.Parse(time.RFC3339, expiresAt.ValueString())
	if err != nil {
		return false
	}

	return time.Now().After(expiry)
}
