package auth

import (
	"context"
	"fmt"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// AuthIdentityModel represents the Terraform state model for an LXD identity.
type AuthIdentityModel struct {
	Name        types.String `tfsdk:"name"`
	Groups      types.Set    `tfsdk:"groups"`
	Type        types.String `tfsdk:"type"`
	AuthMethod  types.String `tfsdk:"auth_method"`
	Certificate types.String `tfsdk:"tls_certificate"`
	Remote      types.String `tfsdk:"remote"`

	// Computed.
	TrustToken types.String `tfsdk:"trust_token"`
	ExpiresAt  types.String `tfsdk:"expires_at"`
}

// AuthIdentityResource manages LXD identity entries.
type AuthIdentityResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewAuthIdentityResource returns a new [AuthIdentityResource].
func NewAuthIdentityResource() resource.Resource {
	return &AuthIdentityResource{}
}

func (r AuthIdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_identity"
}

func (r AuthIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"groups": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, nil)),
			},

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("tls", "bearer", "devlxd"),
				},
				PlanModifiers: []planmodifier.String{
					// Derivation must run before the replacement condition.
					identityTypeModifier{},
					stringplanmodifier.RequiresReplaceIf(
						requiresReplaceIdentityType,
						requiresReplaceIdentityTypeDescription,
						requiresReplaceIdentityTypeDescription,
					),
				},
			},

			"auth_method": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("tls", "bearer"),
				},
				PlanModifiers: []planmodifier.String{
					// Derivation must run before the replacement modifier.
					authMethodModifier{},
					stringplanmodifier.RequiresReplace(),
				},
			},

			// Computed, because a pending TLS identity receives its certificate when an untrusted client
			// redeems the issued trust token.
			"tls_certificate": schema.StringAttribute{
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			// Computed.

			"trust_token": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},

			"expires_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *AuthIdentityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r AuthIdentityResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		identityTypeValidator{warnOnAuthMethod: true},
	}
}

func (r AuthIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthIdentityModel
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

	identityName := plan.Name.ValueString()
	identityType := plan.Type.ValueString()
	identityTLSCertificate := plan.Certificate.ValueString()
	identityGroupNames := []string{}

	resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &identityGroupNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Whether the certificate is set must be determined from the configuration, because the
	// attribute is computed and therefore indistinguishable from an empty value in the plan.
	var configTLSCertificate types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("tls_certificate"), &configTLSCertificate)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasTLSCertificate := !configTLSCertificate.IsNull()

	// A trust token is issued only when creating a pending TLS identity.
	plan.TrustToken = types.StringNull()
	plan.ExpiresAt = types.StringNull()

	switch identityType {
	case "tls":
		if hasTLSCertificate {
			req := api.IdentitiesTLSPost{
				Name:        identityName,
				Groups:      identityGroupNames,
				Certificate: identityTLSCertificate,
			}

			err = server.CreateIdentityTLS(req)
		} else {
			// Without a certificate, LXD creates the identity in a pending state and issues a trust token.
			// The token is returned only here and can never be retrieved from the server afterwards.
			// An untrusted client redeems it to enroll its own certificate, which moves the identity out of
			// the pending state.
			req := api.IdentitiesTLSPost{
				Name:   identityName,
				Groups: identityGroupNames,
				Token:  true,
			}

			var token *api.CertificateAddToken

			token, err = server.CreateIdentityTLSToken(req)
			if err == nil {
				plan.TrustToken = types.StringValue(token.String())
				plan.ExpiresAt = tokenExpiry(token.ExpiresAt)
			}
		}
	case "bearer", "devlxd":
		if hasTLSCertificate {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Invalid %q identity %q", identityType, identityName),
				fmt.Sprintf("Certificate must not be set for identities of type %q", identityType),
			)
			return
		}

		bearerType := api.IdentityTypeBearerTokenClient
		if identityType == "devlxd" {
			bearerType = api.IdentityTypeBearerTokenDevLXD
		}

		req := api.IdentitiesBearerPost{
			Name:   identityName,
			Groups: identityGroupNames,
			Type:   bearerType,
		}

		err = server.CreateIdentityBearer(req)
	default:
		resp.Diagnostics.AddError(
			fmt.Sprintf("Invalid %q identity %q", identityType, identityName),
			fmt.Sprintf("Identity type %q is not supported", identityType),
		)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create %q identity %q", identityType, identityName), err.Error())
		return
	}

	diags := plan.TaintState(ctx, &resp.State)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan, false)
	resp.Diagnostics.Append(diags...)
}

func (r AuthIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthIdentityModel
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

	diags := r.SyncState(ctx, &resp.State, server, state, true)
	resp.Diagnostics.Append(diags...)
}

func (r AuthIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuthIdentityModel
	var state AuthIdentityModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	identityName := plan.Name.ValueString()
	identityType := plan.Type.ValueString()
	identityAuthMethod := plan.AuthMethod.ValueString()
	identityTLSCertificate := plan.Certificate.ValueString()
	identityGroupNames := []string{}

	resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &identityGroupNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Whether the certificate is set must be determined from the configuration, because the
	// attribute is computed and therefore indistinguishable from an empty value in the plan.
	var configTLSCertificate types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("tls_certificate"), &configTLSCertificate)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identity, etag, err := server.GetIdentity(identityAuthMethod, identityName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing %q identity %q", identityType, identityName), err.Error())
		return
	}

	if configTLSCertificate.IsNull() {
		// Carry over the certificate that is stored on the server.
		// Without this, an update that touches only the groups would clear the certificate of an identity
		// that obtained it by redeeming a trust token.
		identityTLSCertificate = identity.TLSCertificate
	}

	identityUpdateReq := api.IdentityPut{
		Groups:         identityGroupNames,
		TLSCertificate: identityTLSCertificate,
	}

	err = server.UpdateIdentity(identityAuthMethod, identityName, identityUpdateReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update %q identity %q", identityType, identityName), err.Error())
		return
	}

	// The trust token is returned only on creation and cannot be retrieved from the server, so the
	// prior state holds the only copy.
	// Both attributes are computed and therefore unknown in the plan, which would discard the token
	// of an identity that is still pending.
	plan.TrustToken = state.TrustToken
	plan.ExpiresAt = state.ExpiresAt

	diags := r.SyncState(ctx, &resp.State, server, plan, false)
	resp.Diagnostics.Append(diags...)
}

func (r AuthIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthIdentityModel
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

	identityName := state.Name.ValueString()
	identityType := state.identityType()
	identityAuthMethod := toAuthMethod(identityType)

	err = server.DeleteIdentity(identityAuthMethod, identityName)
	if err != nil && !errors.IsNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete %q identity %q", identityType, identityName), err.Error())
		return
	}
}

// ModifyPlan forces replacement of a pending TLS identity in two cases: its trust token has
// expired, or a certificate has been configured for it.
// LXD has no endpoint to reissue a token for an existing identity, and it refuses to set a
// certificate on an identity that is still pending.
// Either way the only way forward is to delete the identity and create it again.
func (r AuthIdentityResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip on create (no prior state) and destroy (no plan).
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state AuthIdentityModel
	var config AuthIdentityModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A pending identity is the only TLS identity that LXD reports without a certificate.
	isPending := state.identityType() == "tls" && state.Certificate.IsNull()

	// The attribute that is reported as forcing the replacement.
	var replaceOn path.Path

	switch {
	case isTrustTokenExpired(isPending, state.ExpiresAt, time.Now()):
		replaceOn = path.Root("trust_token")
		resp.Diagnostics.AddWarning(
			"Trust token expired",
			fmt.Sprintf("The trust token for pending TLS identity %q has expired, so Terraform must replace the identity to issue a new token.", state.Name.ValueString()),
		)
	case isCertificateSetOnPending(isPending, config.Certificate):
		replaceOn = path.Root("tls_certificate")
	default:
		return
	}

	// The replacement is created from scratch, so no computed attribute can be carried over from the
	// prior state.
	// An identity created with a certificate is active and has no trust token, one created without is
	// pending and has a new one, and neither is known before apply.
	resp.RequiresReplace = append(resp.RequiresReplace, replaceOn)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("trust_token"), types.StringUnknown())...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("expires_at"), types.StringUnknown())...)
}

// identityType returns the identity type recorded in the state. State written
// before the type attribute existed has only the authentication method, which
// is also an identity type. An imported identity has only the type, until the
// read that follows the import fills the authentication method.
func (m AuthIdentityModel) identityType() string {
	if m.Type.IsNull() {
		return m.AuthMethod.ValueString()
	}

	return m.Type.ValueString()
}

// TaintState marks the state with identity fields required to target the identity.
func (m AuthIdentityModel) TaintState(ctx context.Context, tfState *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(tfState.SetAttribute(ctx, path.Root("name"), m.Name.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("type"), m.Type.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("auth_method"), m.AuthMethod.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("remote"), m.Remote.ValueString())...)

	// The trust token is returned only on creation and cannot be retrieved from the server.
	// Persist it before the remaining state is synced, so that it is not lost if syncing fails.
	diags.Append(tfState.SetAttribute(ctx, path.Root("trust_token"), m.TrustToken)...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("expires_at"), m.ExpiresAt)...)

	return diags
}

func (r AuthIdentityResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m AuthIdentityModel, forgetOnNotFound bool) diag.Diagnostics {
	var respDiags diag.Diagnostics

	identityName := m.Name.ValueString()
	identityType := m.identityType()
	identityAuthMethod := toAuthMethod(identityType)

	identity, _, err := server.GetIdentity(identityAuthMethod, identityName)
	if err != nil {
		if forgetOnNotFound && errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to sync state for %q identity %q", identityType, identityName), err.Error())
		return respDiags
	}

	// The bearer authentication method returns client and devlxd identities
	// alike, so the identity type is taken from the server. LXD identity types
	// that the provider cannot name keep the identity type already in state.
	serverType := toType(identity.Type)
	if serverType != "" {
		identityType = serverType
	}

	m.Name = types.StringValue(identity.Name)
	m.Type = types.StringValue(identityType)
	m.AuthMethod = types.StringValue(identity.AuthenticationMethod)

	if identity.TLSCertificate != "" {
		m.Certificate = types.StringValue(identity.TLSCertificate)
	} else {
		m.Certificate = types.StringNull()
	}

	// The trust token is single use.
	// Once the identity leaves the pending state the token has been consumed and no longer refers to
	// anything.
	if identity.Type != api.IdentityTypeCertificateClientPending {
		m.TrustToken = types.StringNull()
		m.ExpiresAt = types.StringNull()
	}

	groups, diags := common.ToStringSetType(ctx, identity.Groups)
	respDiags.Append(diags...)

	if diags.HasError() {
		return respDiags
	}

	m.Groups = groups

	return tfState.Set(ctx, &m)
}

func (r *AuthIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "auth_identity",
		RequiredFields: []string{"type", "name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		// Attribute "project" is parsed by default, but is not allowed for auth identity.
		if k == "project" {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Invalid import ID %q", req.ID),
				"Valid import format:\nimport lxd_auth_identity.<resource> [remote:]/<type>/<name>",
			)
			break
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

// tokenExpiry converts a trust token expiry into a Terraform value.
// LXD returns a zero time when core.remote_token_expiry is disabled, which means the issued token
// does not expire.
func tokenExpiry(expiresAt time.Time) types.String {
	if expiresAt.IsZero() {
		return types.StringNull()
	}

	return types.StringValue(expiresAt.UTC().Format(time.RFC3339))
}

// isTrustTokenExpired reports whether an identity is still pending and its trust token can no
// longer be redeemed.
// Such an identity is unusable, and because LXD cannot reissue a token, it has to be replaced.
func isTrustTokenExpired(isPending bool, expiresAt types.String, now time.Time) bool {
	if !isPending {
		return false
	}

	// A token without an expiry remains valid indefinitely.
	if expiresAt.IsNull() || expiresAt.IsUnknown() {
		return false
	}

	parsed, err := time.Parse(time.RFC3339, expiresAt.ValueString())
	if err != nil {
		return false
	}

	return now.After(parsed)
}

// isCertificateSetOnPending reports whether a certificate is configured for an identity that is
// still pending.
// LXD rejects such an update, so the identity has to be replaced, which also invalidates the
// outstanding trust token.
//
// The certificate is taken from the configuration rather than the plan.
// While the identity is pending its certificate is null in state, and because the attribute is
// computed the framework marks it unknown in the plan whether or not one was configured, which
// makes the two cases indistinguishable there.
func isCertificateSetOnPending(isPending bool, configCertificate types.String) bool {
	if !isPending {
		return false
	}

	// An unknown certificate is one that is configured but not yet computed.
	// It still means the identity is about to receive one.
	return !configCertificate.IsNull()
}
