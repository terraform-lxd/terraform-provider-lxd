package cluster

import (
	"context"
	"fmt"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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

// ClusterGroupModel is a resource data model that matches the schema.
type ClusterGroupModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Members     types.Set    `tfsdk:"members"`
	Remote      types.String `tfsdk:"remote"`
}

// ClusterGroupResource represent LXD cluster group resource.
type ClusterGroupResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewClusterGroupResource returns a new cluster group resource.
func NewClusterGroupResource() resource.Resource {
	return &ClusterGroupResource{}
}

// Metadata for cluster group resource.
func (r ClusterGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_group"
}

// Schema for cluster group resource.
func (r ClusterGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the cluster group",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"description": schema.StringAttribute{
				Description: "Description of the cluster group",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},

			// Defaults to an empty set rather than null, so that an omitted attribute, an empty
			// list in the configuration and a group without members on the server all agree.
			"members": schema.SetAttribute{
				Description: "Names of the cluster members in the group",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, nil)),
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ClusterGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r ClusterGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterGroupModel

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

	groupName := plan.Name.ValueString()

	members := []string{}
	resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupPost := api.ClusterGroupsPost{
		ClusterGroupPut: api.ClusterGroupPut{
			Description: plan.Description.ValueString(),
			Members:     members,
		},
		Name: groupName,
	}

	err = server.CreateClusterGroup(groupPost)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create cluster group %q", groupName), err.Error())
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

func (r ClusterGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterGroupModel

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

func (r ClusterGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ClusterGroupModel

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

	groupName := plan.Name.ValueString()

	members := []string{}
	resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, etag, err := server.GetClusterGroup(groupName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing cluster group %q", groupName), err.Error())
		return
	}

	groupPut := api.ClusterGroupPut{
		Description: plan.Description.ValueString(),
		Members:     members,
	}

	err = server.UpdateClusterGroup(groupName, groupPut, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update cluster group %q", groupName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan, false)
	resp.Diagnostics.Append(diags...)
}

func (r ClusterGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterGroupModel

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

	groupName := state.Name.ValueString()

	group, etag, err := server.GetClusterGroup(groupName)
	if err != nil {
		if !errors.IsNotFoundError(err) {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing cluster group %q", groupName), err.Error())
		}

		return
	}

	// LXD rejects both of these deletions, so reject them here before touching the members.
	if groupName == "default" {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove cluster group %q", groupName), "The default cluster group cannot be deleted")
		return
	}

	// Reported by servers with the clustering_groups_used_by extension.
	if len(group.UsedBy) > 0 {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove cluster group %q", groupName), "Cluster group is in use by: "+strings.Join(group.UsedBy, ", "))
		return
	}

	// LXD removes only empty cluster groups, so the members are removed from the group first.
	if len(group.Members) > 0 {
		groupPut := api.ClusterGroupPut{
			Description: group.Description,
		}

		err = server.UpdateClusterGroup(groupName, groupPut, etag)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove members from cluster group %q", groupName), err.Error())
			return
		}
	}

	err = server.DeleteClusterGroup(groupName)
	if err == nil || errors.IsNotFoundError(err) {
		return
	}

	// The group stays, so put its members back rather than leaving an empty group behind.
	if len(group.Members) > 0 {
		groupPut := api.ClusterGroupPut{
			Description: group.Description,
			Members:     group.Members,
		}

		restoreErr := server.UpdateClusterGroup(groupName, groupPut, "")
		if restoreErr != nil {
			err = fmt.Errorf("%w (restoring the members also failed: %w)", err, restoreErr)
		}
	}

	resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove cluster group %q", groupName), err.Error())
}

// TaintState marks the state with identity fields required to target the cluster group.
func (m ClusterGroupModel) TaintState(ctx context.Context, tfState *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(tfState.SetAttribute(ctx, path.Root("name"), m.Name.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("remote"), m.Remote.ValueString())...)

	return diags
}

// SyncState fetches the server's current state for a cluster group and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r ClusterGroupResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m ClusterGroupModel, forgetOnNotFound bool) diag.Diagnostics {
	var respDiags diag.Diagnostics

	groupName := m.Name.ValueString()

	group, _, err := server.GetClusterGroup(groupName)
	if err != nil {
		if forgetOnNotFound && errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to sync state for cluster group %q", groupName), err.Error())
		return respDiags
	}

	// An empty set rather than null, to match the attribute default.
	members, diags := common.ToStringSetType(ctx, group.Members)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return respDiags
	}

	if members.IsNull() {
		members = types.SetValueMust(types.StringType, nil)
	}

	m.Name = types.StringValue(group.Name)
	m.Description = types.StringValue(group.Description)
	m.Members = members

	return tfState.Set(ctx, &m)
}

func (r ClusterGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "cluster_group",
		RequiredFields: []string{"name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		// Attribute "project" is parsed by default, but cluster groups are not project scoped.
		if k == "project" {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Invalid import ID %q", req.ID),
				"Valid import format:\nimport lxd_cluster_group.<resource> [remote:]<name>",
			)
			break
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}
