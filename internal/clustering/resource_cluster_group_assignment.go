package clustering

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

type ClusterGroupAssignmentModel struct {
	Remote       types.String `tfsdk:"remote"`
	ClusterGroup types.String `tfsdk:"cluster_group"`
	Member       types.String `tfsdk:"member"`
}

type ClusterGroupAssignmentResource struct {
	provider *provider_config.IncusProviderConfig
}

func NewClusterGroupAssignmentResource() resource.Resource {
	return &ClusterGroupAssignmentResource{}
}

func (r *ClusterGroupAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_cluster_group_assignment", req.ProviderTypeName)
}

func (r *ClusterGroupAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"cluster_group": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"member": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ClusterGroupAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.IncusProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	r.provider = provider
}

func (r *ClusterGroupAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterGroupAssignmentModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterMemberName := plan.Member.ValueString()
	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	clusterMemberNames, err := server.GetClusterMemberNames()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve cluster member names"), err.Error())
		return
	}

	if !containsClusterMember(clusterMemberNames, clusterMemberName) {
		resp.Diagnostics.AddError(fmt.Sprintf("Member with name %q is not part of the cluster", clusterMemberName), "")
		return
	}

	clusterGroupName := plan.ClusterGroup.ValueString()
	clusterGroup, etag, err := server.GetClusterGroup(clusterGroupName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve cluster group %q", clusterGroupName), err.Error())
		return
	}

	if containsClusterMember(clusterGroup.Members, clusterMemberName) {
		resp.Diagnostics.AddError(fmt.Sprintf("Member with name %q is already assigned", clusterMemberName), "")
		return
	}

	updatedClusterMembers := append(clusterGroup.Members, clusterMemberName)
	updatedClusterGroup := api.ClusterGroupPut{
		Description: clusterGroup.Description,
		Members:     updatedClusterMembers,
	}

	err = server.UpdateClusterGroup(clusterGroupName, updatedClusterGroup, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to assign member %q to cluster group %q", clusterMemberName, clusterGroupName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ClusterGroupAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterGroupAssignmentModel

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

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r *ClusterGroupAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// A replacement is always forced for this resource if a value is changed in the model.
}

func (r *ClusterGroupAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterGroupAssignmentModel

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

	clusterGroupName := state.ClusterGroup.ValueString()
	clusterGroup, etag, err := server.GetClusterGroup(clusterGroupName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve cluster group %q", clusterGroupName), err.Error())
		return
	}

	clusterMemberName := state.Member.ValueString()
	updatedClusterMembers := removeClusterMember(clusterGroup.Members, clusterMemberName)
	updatedClusterGroup := api.ClusterGroupPut{
		Description: clusterGroup.Description,
		Members:     updatedClusterMembers,
	}

	err = server.UpdateClusterGroup(clusterGroupName, updatedClusterGroup, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove member %q from cluster group %q", clusterMemberName, clusterGroupName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r *ClusterGroupAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "cluster_group_assignment",
		RequiredFields: []string{"cluster_group", "member"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

// SyncState fetches the server's current state for a cluster group and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r *ClusterGroupAssignmentResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m ClusterGroupAssignmentModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	clusterGroupName := m.ClusterGroup.ValueString()
	clusterGroup, _, err := server.GetClusterGroup(clusterGroupName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve cluster group %q", clusterGroupName), err.Error())
		return respDiags
	}

	m.ClusterGroup = types.StringValue(clusterGroup.Name)

	return tfState.Set(ctx, &m)
}

func containsClusterMember(clusterMembers []string, clusterMemberName string) bool {
	for _, member := range clusterMembers {
		if member == clusterMemberName {
			return true
		}
	}
	return false
}

func removeClusterMember(clusterMembers []string, clusterMemberName string) []string {
	var result []string
	for _, item := range clusterMembers {
		if item != clusterMemberName {
			result = append(result, item)
		}
	}
	return result
}
