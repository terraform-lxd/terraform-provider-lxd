package auth

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// AuthGroupModel represents the Terraform state model for an LXD auth group.
type AuthGroupModel struct {
	Name        types.String      `tfsdk:"name"`
	Description types.String      `tfsdk:"description"`
	Permissions []PermissionModel `tfsdk:"permissions"`
}

// AuthGroupResource manages LXD auth groups.
type AuthGroupResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewAuthGroupResource returns a new AuthGroupResource.
func NewAuthGroupResource() resource.Resource {
	return &AuthGroupResource{}
}

func (r AuthGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_group"
}

func (r AuthGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},

			"permissions": schema.SetNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  setdefault.StaticValue(types.SetValueMust(PermissionModelType, nil)),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"entitlement": schema.StringAttribute{
							Required: true,
						},
						"entity_type": schema.StringAttribute{
							Required: true,
						},
						"entity_args": schema.MapAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *AuthGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r AuthGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer("", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	permissions, err := PermissionsToAPI(plan.Permissions)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert permissions to API format", err.Error())
		return
	}

	groupReq := api.AuthGroupsPost{
		AuthGroupPost: api.AuthGroupPost{
			Name: plan.Name.ValueString(),
		},
		AuthGroupPut: api.AuthGroupPut{
			Description: plan.Description.ValueString(),
			Permissions: permissions,
		},
	}

	err = server.CreateAuthGroup(groupReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create auth group", err.Error())
		return
	}

	diags := r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r AuthGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthGroupModel
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

func (r AuthGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuthGroupModel
	var state AuthGroupModel

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

	_, etag, err := server.GetAuthGroup(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing auth group %q", state.Name.ValueString()), err.Error())
		return
	}

	permissions, err := PermissionsToAPI(plan.Permissions)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert permissions to API format", err.Error())
		return
	}

	groupPut := api.AuthGroupPut{
		Description: plan.Description.ValueString(),
		Permissions: permissions,
	}

	authGroupName := state.Name.ValueString()
	err = server.UpdateAuthGroup(authGroupName, groupPut, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update auth group %q", authGroupName), err.Error())
		return
	}

	diags := r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r AuthGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer("", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	authGroupName := state.Name.ValueString()
	err = server.DeleteAuthGroup(authGroupName)
	if err != nil && !errors.IsNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete auth group %q", authGroupName), err.Error())
		return
	}
}

func (r AuthGroupResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m AuthGroupModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	authGroupName := m.Name.ValueString()
	authGroup, _, err := server.GetAuthGroup(authGroupName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve auth group %q", authGroupName), err.Error())
		return respDiags
	}

	m.Name = types.StringValue(authGroup.Name)
	m.Description = types.StringValue(authGroup.Description)
	permissions, err := PermissionsFromAPI(ctx, authGroup.Permissions)
	if err != nil {
		respDiags.AddError("Failed to convert permissions from API format", err.Error())
		return respDiags
	}

	m.Permissions = permissions

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

func (r *AuthGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "auth_group",
		RequiredFields: []string{"name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		// Attribute "project" is parsed by default, but is not allowed for auth group.
		if k == "project" {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Invalid import ID %q", req.ID),
				"Valid import format:\nimport lxd_auth_group.<resource> [remote:]<name>",
			)
			break
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}
