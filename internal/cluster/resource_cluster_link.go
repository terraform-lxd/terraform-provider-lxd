package cluster

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// clusterLinkComputedKeys lists the config keys that LXD manages itself. They are hidden from
// the configuration in state and sent back unchanged on update, because LXD rejects an update
// that adds, removes or changes any of them.
var clusterLinkComputedKeys = []string{"volatile."}

// ClusterLinkModel is a resource data model that matches the schema.
type ClusterLinkModel struct {
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Type          types.String `tfsdk:"type"`
	Config        types.Map    `tfsdk:"config"`
	AuthGroups    types.Set    `tfsdk:"auth_groups"`
	TrustToken    types.String `tfsdk:"trust_token"`
	RemoteAddress types.String `tfsdk:"remote_address"`
	Fingerprint   types.String `tfsdk:"fingerprint"`
	Remote        types.String `tfsdk:"remote"`

	// Computed.
	Addresses types.List   `tfsdk:"addresses"`
	ExpiresAt types.String `tfsdk:"expires_at"`
}

// ClusterLinkResource represent LXD cluster link resource.
type ClusterLinkResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewClusterLinkResource returns a new cluster link resource.
func NewClusterLinkResource() resource.Resource {
	return &ClusterLinkResource{}
}

// Metadata for cluster link resource.
func (r ClusterLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_link"
}

// Schema for cluster link resource.
func (r ClusterLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the cluster link",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"description": schema.StringAttribute{
				Description: "Description of the cluster link",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},

			"type": schema.StringAttribute{
				Description: "Type of the cluster link",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(api.ClusterLinkTypeBidirectional),
				Validators: []validator.String{
					stringvalidator.OneOf(api.ClusterLinkTypeBidirectional, api.ClusterLinkTypeUnidirectional, api.ClusterLinkTypePublic),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					// The only keys LXD accepts from a client. The volatile keys are managed by LXD.
					mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile(`^user\.`), "must be a user.* key")),
				},
			},

			"auth_groups": schema.SetAttribute{
				Description: "Auth groups of the identity that LXD creates for the linked cluster. Applicable only to bidirectional links.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, nil)),
			},

			// Computed, because a bidirectional link created without a trust token issues one for the
			// linked cluster.
			"trust_token": schema.StringAttribute{
				Description: "Trust token issued by the linked cluster. If omitted on a bidirectional link, a trust token is issued instead and exported here.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					// An issued token is returned only on creation and cannot be retrieved from the server, so
					// the state holds the only copy.
					stringplanmodifier.UseStateForUnknown(),
					// A token is consumed on creation only, so a different one means a different link.
					stringplanmodifier.RequiresReplaceIf(
						requiresReplaceUnlessImported,
						requiresReplaceUnlessImportedDescription,
						requiresReplaceUnlessImportedDescription,
					),
				},
			},

			"remote_address": schema.StringAttribute{
				Description: "Address of the linked cluster. Applicable only to public links.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						requiresReplaceUnlessImported,
						requiresReplaceUnlessImportedDescription,
						requiresReplaceUnlessImportedDescription,
					),
				},
			},

			"fingerprint": schema.StringAttribute{
				Description: "SHA-256 fingerprint of the linked cluster's certificate. Applicable only to public links.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						requiresReplaceUnlessImported,
						requiresReplaceUnlessImportedDescription,
						requiresReplaceUnlessImportedDescription,
					),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			// Computed.

			"addresses": schema.ListAttribute{
				Description: "Addresses of the linked cluster's members. Empty while the link is pending.",
				Computed:    true,
				ElementType: types.StringType,
			},

			"expires_at": schema.StringAttribute{
				Description: "Time at which the issued trust token expires.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

const requiresReplaceUnlessImportedDescription = "If the value of this attribute changes, Terraform will destroy and recreate the resource, unless the resource was imported and has no prior value."

// requiresReplaceUnlessImported replaces the cluster link when the attribute changes, except when
// the prior state has no value. The attributes it guards are used only on creation and cannot be
// read back from the server, so an imported link adopts the configured value instead of being
// replaced.
func requiresReplaceUnlessImported(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = !req.StateValue.IsNull()
}

func (r ClusterLinkResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		clusterLinkTypeValidator{},
	}
}

func (r *ClusterLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r ClusterLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterLinkModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	linkName := plan.Name.ValueString()
	linkType := plan.Type.ValueString()

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	authGroups := []string{}
	resp.Diagnostics.Append(plan.AuthGroups.ElementsAs(ctx, &authGroups, false)...)

	// Whether a trust token is set must be determined from the configuration, because the
	// attribute is computed and therefore indistinguishable from an empty value in the plan.
	var configTrustToken types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("trust_token"), &configTrustToken)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A trust token is issued only when creating a pending bidirectional link.
	plan.TrustToken = configTrustToken
	plan.ExpiresAt = types.StringNull()

	// Checked again here, because the configuration validator skips values that are unknown
	// until apply, and the attributes a type requires are checked only on creation.
	err = validateClusterLinkAttributes(linkType, plan, true)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Invalid %q cluster link %q", linkType, linkName), err.Error())
		return
	}

	linkPost := api.ClusterLinksPost{
		ClusterLinkPut: api.ClusterLinkPut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
		Name:       linkName,
		Type:       linkType,
		TrustToken: configTrustToken.ValueString(),
		AuthGroups: authGroups,
	}

	switch {
	case linkType == api.ClusterLinkTypePublic:
		err = createPublicClusterLink(server, linkPost, plan.RemoteAddress.ValueString(), plan.Fingerprint.ValueString())
	case linkType == api.ClusterLinkTypeBidirectional && configTrustToken.IsNull():
		// Without a token, LXD creates the link in a pending state and issues a trust token for the
		// linked cluster. The token is returned only here and can never be retrieved from the server
		// afterwards. The linked cluster redeems it to activate the link on both sides.
		var token *api.CertificateAddToken

		token, err = server.CreateIdentityClusterLinkToken(linkPost)
		if err == nil {
			plan.TrustToken = types.StringValue(token.String())
			plan.ExpiresAt = common.TokenExpiry(token.ExpiresAt)
		}
	default:
		// With a token from the linked cluster, LXD validates its certificate, creates the link and
		// activates the link on the linked cluster during the same request.
		err = server.CreateClusterLink(linkPost)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create cluster link %q", linkName), err.Error())
		return
	}

	diags = plan.TaintState(ctx, &resp.State)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan, false)
	resp.Diagnostics.Append(diags...)
}

func (r ClusterLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterLinkModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, state, true)
	resp.Diagnostics.Append(diags...)
}

func (r ClusterLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ClusterLinkModel
	var state ClusterLinkModel

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

	linkName := plan.Name.ValueString()
	linkType := plan.Type.ValueString()

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	authGroups := []string{}
	resp.Diagnostics.Append(plan.AuthGroups.ElementsAs(ctx, &authGroups, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Checked again here, because the configuration validator skips values that are unknown
	// until apply.
	err = validateClusterLinkAttributes(linkType, plan, false)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Invalid %q cluster link %q", linkType, linkName), err.Error())
		return
	}

	link, etag, err := server.GetClusterLink(linkName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing cluster link %q", linkName), err.Error())
		return
	}

	// LXD rejects an update that adds, removes or changes volatile keys, so the live ones are
	// merged back into the user configuration.
	config := common.MergeConfig(link.Config, userConfig, clusterLinkComputedKeys)

	linkPut := api.ClusterLinkPut{
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	err = server.UpdateClusterLink(linkName, linkPut, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update cluster link %q", linkName), err.Error())
		return
	}

	// Auth groups belong to the identity that LXD created for the linked cluster, which has the
	// same name as the link. The link API has no way to change them.
	if linkType == api.ClusterLinkTypeBidirectional && !plan.AuthGroups.Equal(state.AuthGroups) {
		identity, etag, err := server.GetIdentity(api.AuthenticationMethodTLS, linkName)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve identity of cluster link %q", linkName), err.Error())
			return
		}

		// Carry over the certificate, otherwise the update clears it.
		identityPut := api.IdentityPut{
			Groups:         authGroups,
			TLSCertificate: identity.TLSCertificate,
		}

		err = server.UpdateIdentity(api.AuthenticationMethodTLS, linkName, identityPut, etag)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to update auth groups of cluster link %q", linkName), err.Error())
			return
		}
	}

	// The trust token is returned only on creation and cannot be retrieved from the server, so the
	// prior state holds the only copy. A known planned token is either the same one or, after an
	// import, a configured one being adopted. A different configured token replaces the link instead.
	if plan.TrustToken.IsUnknown() {
		plan.TrustToken = state.TrustToken
	}

	plan.ExpiresAt = state.ExpiresAt

	diags = r.SyncState(ctx, &resp.State, server, plan, false)
	resp.Diagnostics.Append(diags...)
}

func (r ClusterLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterLinkModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	linkName := state.Name.ValueString()

	err = server.DeleteClusterLink(linkName)
	if err != nil && !errors.IsNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove cluster link %q", linkName), err.Error())
	}
}

// TaintState marks the state with identity fields required to target the cluster link.
func (m ClusterLinkModel) TaintState(ctx context.Context, tfState *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(tfState.SetAttribute(ctx, path.Root("name"), m.Name.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("type"), m.Type.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("remote"), m.Remote.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("remote_address"), m.RemoteAddress)...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("fingerprint"), m.Fingerprint)...)

	// The trust token is returned only on creation and cannot be retrieved from the server.
	// Persist it before the remaining state is synced, so that it is not lost if syncing fails.
	diags.Append(tfState.SetAttribute(ctx, path.Root("trust_token"), m.TrustToken)...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("expires_at"), m.ExpiresAt)...)

	return diags
}

// SyncState fetches the server's current state for a cluster link and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r ClusterLinkResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m ClusterLinkModel, forgetOnNotFound bool) diag.Diagnostics {
	var respDiags diag.Diagnostics

	linkName := m.Name.ValueString()

	link, _, err := server.GetClusterLink(linkName)
	if err != nil {
		if forgetOnNotFound && errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to sync state for cluster link %q", linkName), err.Error())
		return respDiags
	}

	config, diags := common.ToConfigMapType(ctx, common.StripConfig(link.Config, m.Config, clusterLinkComputedKeys), m.Config)
	respDiags.Append(diags...)

	addresses, diags := toAddressesListType(ctx, link.Config)
	respDiags.Append(diags...)

	// Only a bidirectional link has an identity on this cluster, whose groups are the link's auth
	// groups. The other types keep an empty set, so an imported link does not plan an update.
	// The identity is found by name, which LXD keeps unique among TLS identities.
	authGroups := m.AuthGroups
	if link.Type == api.ClusterLinkTypeBidirectional {
		identity, _, err := server.GetIdentity(api.AuthenticationMethodTLS, linkName)
		if err != nil {
			respDiags.AddError(fmt.Sprintf("Failed to retrieve identity of cluster link %q", linkName), err.Error())
			return respDiags
		}

		authGroups, diags = common.ToStringSetType(ctx, identity.Groups)
		respDiags.Append(diags...)
	}

	if authGroups.IsNull() || authGroups.IsUnknown() {
		authGroups = types.SetValueMust(types.StringType, nil)
	}

	if respDiags.HasError() {
		return respDiags
	}

	// The issued token is single use. When the linked cluster redeems it, the link turns from
	// pending to active and the token no longer refers to anything. Only the issuing side ever
	// sees that transition, so a token configured on the consuming side is never touched.
	if isPendingClusterLink(m.Addresses) && len(addresses.Elements()) > 0 {
		m.TrustToken = types.StringNull()
		m.ExpiresAt = types.StringNull()
	}

	m.Name = types.StringValue(link.Name)
	m.Description = types.StringValue(link.Description)
	m.Type = types.StringValue(link.Type)
	m.Config = config
	m.AuthGroups = authGroups
	m.Addresses = addresses

	return tfState.Set(ctx, &m)
}

func (r ClusterLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "cluster_link",
		RequiredFields: []string{"name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		// Attribute "project" is parsed by default, but cluster links are not project scoped.
		if k == "project" {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Invalid import ID %q", req.ID),
				"Valid import format:\nimport lxd_cluster_link.<resource> [remote:]<name>",
			)
			break
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

// ModifyPlan forces replacement of a pending cluster link whose trust token has expired. LXD has
// no endpoint to reissue a token for an existing link, so the only way to get a new one is to
// delete the link and create it again.
func (r ClusterLinkResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip on create (no prior state) and destroy (no plan).
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state ClusterLinkModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only a pending link has an outstanding token.
	if !isPendingClusterLink(state.Addresses) || !common.IsTokenExpired(state.ExpiresAt, time.Now()) {
		return
	}

	resp.Diagnostics.AddWarning(
		"Trust token expired",
		fmt.Sprintf("The trust token for pending cluster link %q has expired, so Terraform must replace the link to issue a new token.", state.Name.ValueString()),
	)

	// The replacement issues a new token, which is not known before apply.
	resp.RequiresReplace = append(resp.RequiresReplace, path.Root("trust_token"))
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("trust_token"), types.StringUnknown())...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("expires_at"), types.StringUnknown())...)
}

// isPendingClusterLink reports whether a link is still pending, that is whether the linked
// cluster has not yet redeemed the issued token. LXD records the linked cluster's addresses when
// it does, so a link without addresses is pending.
func isPendingClusterLink(addresses types.List) bool {
	return !addresses.IsNull() && !addresses.IsUnknown() && len(addresses.Elements()) == 0
}

// createPublicClusterLink creates a public cluster link. LXD first contacts the remote address,
// fetches the certificate served there and returns its fingerprint. The link is inert until the
// fingerprint is confirmed in a second request, which pins the certificate. The CLI asks the user
// to confirm interactively, which Terraform cannot, so the expected fingerprint must be configured.
func createPublicClusterLink(server lxd.InstanceServer, linkPost api.ClusterLinksPost, remoteAddress string, fingerprint string) error {
	// LXD refreshes an existing pending public link of the same name instead of rejecting the
	// request, and the cleanup below would then remove a link that Terraform does not manage.
	_, _, err := server.GetClusterLink(linkPost.Name)
	if err == nil {
		return fmt.Errorf("Cluster link %q already exists. Import it, or remove it if it is a leftover pending link", linkPost.Name)
	}

	if !errors.IsNotFoundError(err) {
		return err
	}

	linkPost.RemoteAddress = remoteAddress

	cert, err := server.CreateClusterLinkPendingPublic(linkPost)
	if err != nil {
		return err
	}

	// Any failure before confirmation leaves an inert pending link behind, so remove it.
	abort := func(cause error) error {
		err := server.DeleteClusterLink(linkPost.Name)
		if err != nil {
			return fmt.Errorf("%w (removing the pending cluster link also failed: %w)", cause, err)
		}

		return cause
	}

	if !strings.EqualFold(fingerprint, cert.Fingerprint) {
		return abort(fmt.Errorf("Fingerprint %q does not match fingerprint %q of the certificate served at %q", fingerprint, cert.Fingerprint, remoteAddress))
	}

	// Confirming takes only the name and the fingerprint. The address was used in the pending
	// phase and is rejected here.
	confirmPost := api.ClusterLinksPost{
		Name:        linkPost.Name,
		Type:        api.ClusterLinkTypePublic,
		Fingerprint: cert.Fingerprint,
	}

	err = server.CreateClusterLink(confirmPost)
	if err != nil {
		// A conflict means the link is already confirmed, so it must not be removed.
		if errors.IsConflictError(err) {
			return fmt.Errorf("%w. The cluster link is already active and can be imported instead", err)
		}

		return abort(err)
	}

	return nil
}

// clusterLinkTypeValidator ensures that only the attributes applicable to the configured link
// type are set, and that the ones the type requires are present.
type clusterLinkTypeValidator struct{}

func (v clusterLinkTypeValidator) Description(_ context.Context) string {
	return "Ensures that the configured attributes match the cluster link type."
}

func (v clusterLinkTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v clusterLinkTypeValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ClusterLinkModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A type that is unknown until apply is checked again on creation.
	if config.Type.IsUnknown() {
		return
	}

	// The default is applied only after validation.
	linkType := api.ClusterLinkTypeBidirectional
	if !config.Type.IsNull() {
		linkType = config.Type.ValueString()
	}

	err := validateClusterLinkAttributes(linkType, config, false)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("type"), fmt.Sprintf("Invalid %q cluster link", linkType), err.Error())
	}
}

// validateClusterLinkAttributes reports an attribute that is set but not applicable to the link
// type. On creation it also reports an attribute that the type requires but is missing; those
// attributes are used only on creation and cannot be read back, so an imported link may omit
// them. An unknown value is neither set nor missing, because it may still resolve to either;
// Create and Update check the resolved values again.
//
// LXD rejects most of these combinations itself, but not all: without a trust token it creates a
// pending identity instead of a unidirectional link, and it ignores auth groups on links that
// create no identity.
func validateClusterLinkAttributes(linkType string, m ClusterLinkModel, create bool) error {
	isSet := func(v types.String) bool {
		return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
	}

	hasAuthGroups := !m.AuthGroups.IsNull() && !m.AuthGroups.IsUnknown() && len(m.AuthGroups.Elements()) > 0

	if linkType != api.ClusterLinkTypePublic && (isSet(m.RemoteAddress) || isSet(m.Fingerprint)) {
		return fmt.Errorf("Remote address and fingerprint can only be set for %q cluster links", api.ClusterLinkTypePublic)
	}

	if linkType != api.ClusterLinkTypeBidirectional && hasAuthGroups {
		return fmt.Errorf("Auth groups cannot be set for %q cluster links, because no identity is created for the linked cluster", linkType)
	}

	if linkType == api.ClusterLinkTypePublic && isSet(m.TrustToken) {
		return fmt.Errorf("Trust token cannot be set for public cluster links")
	}

	if !create {
		return nil
	}

	switch linkType {
	case api.ClusterLinkTypeUnidirectional:
		if !isSet(m.TrustToken) {
			return fmt.Errorf("Unidirectional cluster links require a trust token issued by the linked cluster")
		}
	case api.ClusterLinkTypePublic:
		if !isSet(m.RemoteAddress) {
			return fmt.Errorf("Public cluster links require the remote address of the linked cluster")
		}

		if !isSet(m.Fingerprint) {
			return fmt.Errorf("Public cluster links require the fingerprint of the linked cluster's certificate")
		}
	}

	return nil
}

// toAddressesListType converts the comma-separated "volatile.addresses" config value into a list.
// The list is empty rather than null while the link is pending, so that its length tells whether
// the link is active.
func toAddressesListType(ctx context.Context, config map[string]string) (types.List, diag.Diagnostics) {
	addresses := shared.SplitNTrimSpace(config["volatile.addresses"], ",", -1, true)
	if addresses == nil {
		addresses = []string{}
	}

	return types.ListValueFrom(ctx, types.StringType, addresses)
}
