package auth

import (
	"context"
	"fmt"

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
	AuthMethod  types.String `tfsdk:"auth_method"`
	Certificate types.String `tfsdk:"tls_certificate"`
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

			"auth_method": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("tls", "bearer"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"tls_certificate": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
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

func (r AuthIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthIdentityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer("", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	identityName := plan.Name.ValueString()
	identityAuthMethod := plan.AuthMethod.ValueString()
	identityTLSCertificate := plan.Certificate.ValueString()
	identityGroupNames := []string{}

	resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &identityGroupNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch identityAuthMethod {
	case "tls":
		req := api.IdentitiesTLSPost{
			Name:        identityName,
			Groups:      identityGroupNames,
			Certificate: identityTLSCertificate,
		}

		err = server.CreateIdentityTLS(req)
	case "bearer":
		if identityTLSCertificate != "" {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Invalid %q identity %q", identityName, identityAuthMethod),
				"Certificate must not be set for identities with authentication method bearer",
			)
			return
		}

		req := api.IdentitiesBearerPost{
			Name:   identityName,
			Groups: identityGroupNames,
			Type:   api.IdentityTypeBearerTokenClient,
		}

		err = server.CreateIdentityBearer(req)
	default:
		resp.Diagnostics.AddError(
			fmt.Sprintf("Invalid %q identity %q", identityName, identityAuthMethod),
			fmt.Sprintf("Authentication method %q is not supported", identityAuthMethod),
		)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create %q identity %q", identityAuthMethod, identityName), err.Error())
		return
	}

	diags := r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r AuthIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthIdentityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer("", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	diags := r.SyncState(ctx, &resp.State, server, state)
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

	server, err := r.provider.InstanceServer("", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	identityName := plan.Name.ValueString()
	identityAuthMethod := plan.AuthMethod.ValueString()
	identityTLSCertificate := plan.Certificate.ValueString()
	identityGroupNames := []string{}

	resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &identityGroupNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, etag, err := server.GetIdentity(identityAuthMethod, identityName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing %q identity %q", identityAuthMethod, identityName), err.Error())
		return
	}

	identityUpdateReq := api.IdentityPut{
		Groups:         identityGroupNames,
		TLSCertificate: identityTLSCertificate,
	}

	err = server.UpdateIdentity(identityAuthMethod, identityName, identityUpdateReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update %q identity %q", identityAuthMethod, identityName), err.Error())
		return
	}

	diags := r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r AuthIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthIdentityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer("", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	identityName := state.Name.ValueString()
	identityAuthMethod := state.AuthMethod.ValueString()

	err = server.DeleteIdentity(identityAuthMethod, identityName)
	if err != nil && !errors.IsNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete %q identity %q", identityAuthMethod, identityName), err.Error())
		return
	}
}

func (r AuthIdentityResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m AuthIdentityModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	identityName := m.Name.ValueString()
	identityAuthMethod := m.AuthMethod.ValueString()

	identity, _, err := server.GetIdentity(identityAuthMethod, identityName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve existing %q identity %q", identityAuthMethod, identityName), err.Error())
		return respDiags
	}

	m.Name = types.StringValue(identity.Name)
	m.AuthMethod = types.StringValue(identityAuthMethod)
	if identity.TLSCertificate != "" {
		m.Certificate = types.StringValue(identity.TLSCertificate)
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
		RequiredFields: []string{"auth_method", "name"},
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
				"Valid import format:\nimport lxd_auth_identity.<resource> [remote:]<auth_method>/<name>",
			)
			break
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}
