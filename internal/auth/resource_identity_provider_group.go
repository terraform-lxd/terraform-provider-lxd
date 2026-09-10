package auth

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// AuthIdentityProviderGroupModel represents the Terraform state model for an LXD identity provider group.
type AuthIdentityProviderGroupModel struct {
	Name   types.String `tfsdk:"name"`
	Groups types.Set    `tfsdk:"groups"`
	Remote types.String `tfsdk:"remote"`
}

// AuthIdentityProviderGroupIdentityModel represents the resource identity of an LXD identity provider group.
type AuthIdentityProviderGroupIdentityModel struct {
	Name   types.String `tfsdk:"name"`
	Remote types.String `tfsdk:"remote"`
}

// AuthIdentityProviderGroupResource manages LXD identity provider groups.
type AuthIdentityProviderGroupResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewAuthIdentityProviderGroupResource returns a new AuthIdentityProviderGroupResource.
func NewAuthIdentityProviderGroupResource() resource.Resource {
	return &AuthIdentityProviderGroupResource{}
}

func (r AuthIdentityProviderGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_identity_provider_group"

	// The identity contains the remote name, which the user can change in place.
	resp.ResourceBehavior.MutableIdentity = true
}

func (r AuthIdentityProviderGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"remote": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r AuthIdentityProviderGroupResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "Name of the identity provider group.",
			},

			"remote": identityschema.StringAttribute{
				OptionalForImport: true,
				Description:       "Remote in which the identity provider group is defined. If not provided, the provider's default remote is used.",
			},
		},
	}
}

func (r *AuthIdentityProviderGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r AuthIdentityProviderGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthIdentityProviderGroupModel
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

	groupNames := []string{}
	resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groupNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idpGroupReq := api.IdentityProviderGroupsPost{
		Name:   plan.Name.ValueString(),
		Groups: groupNames,
	}

	err = server.CreateIdentityProviderGroup(idpGroupReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create identity provider group", err.Error())
		return
	}

	diags := plan.TaintState(ctx, &resp.State)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = plan.SetIdentity(ctx, resp.Identity)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan, false)
	resp.Diagnostics.Append(diags...)
}

func (r AuthIdentityProviderGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthIdentityProviderGroupModel
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

	// Set before syncing the state, so that state imported or upgraded from a
	// version without resource identity also carries the identity.
	diags := state.SetIdentity(ctx, resp.Identity)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, state, true)
	resp.Diagnostics.Append(diags...)
}

func (r AuthIdentityProviderGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuthIdentityProviderGroupModel
	var state AuthIdentityProviderGroupModel

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

	idpGroupName := state.Name.ValueString()
	_, etag, err := server.GetIdentityProviderGroup(idpGroupName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing identity provider group %q", idpGroupName), err.Error())
		return
	}

	groupNames := []string{}
	resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groupNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idpGroupPut := api.IdentityProviderGroupPut{
		Groups: groupNames,
	}

	err = server.UpdateIdentityProviderGroup(idpGroupName, idpGroupPut, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update identity provider group %q", idpGroupName), err.Error())
		return
	}

	diags := plan.SetIdentity(ctx, resp.Identity)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan, false)
	resp.Diagnostics.Append(diags...)
}

func (r AuthIdentityProviderGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthIdentityProviderGroupModel
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

	idpGroupName := state.Name.ValueString()
	err = server.DeleteIdentityProviderGroup(idpGroupName)
	if err != nil && !errors.IsNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete identity provider group %q", idpGroupName), err.Error())
		return
	}
}

// TaintState marks the state with identity fields required to target the identity provider group.
func (m AuthIdentityProviderGroupModel) TaintState(ctx context.Context, tfState *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(tfState.SetAttribute(ctx, path.Root("name"), m.Name.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("remote"), m.Remote.ValueString())...)

	return diags
}

// SetIdentity writes the resource identity that addresses the identity provider group.
func (m AuthIdentityProviderGroupModel) SetIdentity(ctx context.Context, tfIdentity *tfsdk.ResourceIdentity) diag.Diagnostics {
	return tfIdentity.Set(ctx, AuthIdentityProviderGroupIdentityModel{
		Name:   m.Name,
		Remote: m.Remote,
	})
}

func (r AuthIdentityProviderGroupResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m AuthIdentityProviderGroupModel, forgetOnNotFound bool) diag.Diagnostics {
	var respDiags diag.Diagnostics

	idpGroupName := m.Name.ValueString()
	idpGroup, _, err := server.GetIdentityProviderGroup(idpGroupName)
	if err != nil {
		if forgetOnNotFound && errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to sync state for identity provider group %q", idpGroupName), err.Error())
		return respDiags
	}

	groups, diags := common.ToStringSetType(ctx, idpGroup.Groups)
	respDiags.Append(diags...)

	if diags.HasError() {
		return respDiags
	}

	m.Name = types.StringValue(idpGroup.Name)
	m.Groups = groups

	return tfState.Set(ctx, &m)
}

func (r *AuthIdentityProviderGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" {
		// The identity provider group name can contain slashes and colons which cause
		// issues when parsing the import ID. Therefore, we require the resource identity
		// to be used for import.
		resp.Diagnostics.AddError(
			fmt.Sprintf("Resource lxd_auth_identity_provider_group does not support import using ID %q", req.ID),
			"Import the resource using its identity instead of its ID.",
		)

		return
	}

	var identity AuthIdentityProviderGroupIdentityModel
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), identity.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote"), identity.Remote)...)
}
