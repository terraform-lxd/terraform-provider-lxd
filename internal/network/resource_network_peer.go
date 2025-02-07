package network

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// NetworkPeerModel is a resource data model that matches the schema.
type NetworkPeerModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	// Source network.
	SourceNetwork types.String `tfsdk:"source_network"`
	SourceProject types.String `tfsdk:"source_project"`

	// Target network.
	TargetNetwork types.String `tfsdk:"target_network"`
	TargetProject types.String `tfsdk:"target_project"`

	Remote types.String `tfsdk:"remote"`
	Config types.Map    `tfsdk:"config"`
	Status types.String `tfsdk:"status"`
}

// NetworkPeerResource represent LXD network peer resource.
type NetworkPeerResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewNetworkPeerResource returns a new network peer resource.
func NewNetworkPeerResource() resource.Resource {
	return &NetworkPeerResource{}
}

// Metadata for network peer resource.
func (r NetworkPeerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_peer"
}

// Schema for network peer resource.
func (r NetworkPeerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the network peer",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"description": schema.StringAttribute{
				Description: "Description of the network peer",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},

			"source_network": schema.StringAttribute{
				Description: "Name of the source network.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"source_project": schema.StringAttribute{
				Description: "Project of the source network.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("default"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"target_network": schema.StringAttribute{
				Description: "Name of the target network.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"target_project": schema.StringAttribute{
				Description: "Project of the target network. Defaults to source project.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},

			"status": schema.StringAttribute{
				Description: "Network peer status",
				Computed:    true,
			},
		},
	}
}

func (r *NetworkPeerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r NetworkPeerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkPeerModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	peerName := plan.Name.ValueString()
	remote := plan.Remote.ValueString()
	srcNetwork := plan.SourceNetwork.ValueString()
	srcProject := plan.SourceProject.ValueString()
	dstNetwork := plan.TargetNetwork.ValueString()
	dstProject := plan.TargetProject.ValueString()

	if dstProject == "" {
		dstProject = srcProject
	}

	// Convert network peer config to map.
	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create network source peer.
	server, err := r.provider.InstanceServer(remote, srcProject, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	peer := api.NetworkPeersPost{
		NetworkPeerPut: api.NetworkPeerPut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
		Name:          peerName,
		TargetProject: dstProject,
		TargetNetwork: dstNetwork,
	}

	err = server.CreateNetworkPeer(srcNetwork, peer)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network peer %q", peerName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkPeerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkPeerModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.SourceProject.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkPeerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkPeerModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	peerName := plan.Name.ValueString()
	description := plan.Description.ValueString()
	srcProject := plan.SourceProject.ValueString()
	srcNetwork := plan.SourceNetwork.ValueString()
	remote := plan.Remote.ValueString()

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(remote, srcProject, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Fetch and existing network peer.
	_, etag, err := server.GetNetworkPeer(srcNetwork, peerName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network peer %q", peerName), err.Error())
		return
	}

	srcPeer := api.NetworkPeerPut{
		Config:      config,
		Description: description,
	}

	// Update network peer.
	err = server.UpdateNetworkPeer(srcNetwork, peerName, srcPeer, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network peer %q", peerName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkPeerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkPeerModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	peerName := state.Name.ValueString()
	srcProject := state.SourceProject.ValueString()
	srcNetwork := state.SourceNetwork.ValueString()
	remote := state.Remote.ValueString()

	server, err := r.provider.InstanceServer(remote, srcProject, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Remove source network peer. Retry multiple times, because if
	// network peer is removed immediately after creation, the changes
	// may not be applied in OVN database yet.
	for range 3 {
		err = server.DeleteNetworkPeer(srcNetwork, peerName)
		if err == nil {
			break
		}

		if errors.IsNotFoundError(err) {
			return
		}

		time.Sleep(200 * time.Millisecond)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network peer %q", peerName), err.Error())
	}
}

// SyncState fetches the server's current state for a network peer and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r NetworkPeerResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m NetworkPeerModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	peerName := m.Name.ValueString()

	srcNetwork := m.SourceNetwork.ValueString()
	peer, _, err := server.GetNetworkPeer(srcNetwork, peerName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		return diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve network peer %q", peerName), err.Error()),
		}
	}

	// Convert config state into schema type.
	config, diags := common.ToConfigMapType(ctx, nil, m.Config)
	respDiags.Append(diags...)

	m.Name = types.StringValue(peer.Name)
	m.Description = types.StringValue(peer.Description)
	m.TargetNetwork = types.StringValue(peer.TargetNetwork)
	m.TargetProject = types.StringValue(peer.TargetProject)
	m.Status = types.StringValue(peer.Status)
	m.Config = config

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

func (r NetworkPeerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName: "network_peer",
		RequiredFields: []string{
			"name",
			"source_project",
			"source_network",
			"target_project",
			"target_network",
		},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	// Remove project field because we are extracting source and target
	// projects as required fields.
	delete(fields, "project")

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}
