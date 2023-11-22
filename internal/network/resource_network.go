package network

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// NetworkModel resource data model that matches the schema.
type NetworkModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Target      types.String `tfsdk:"target"`
	Managed     types.Bool   `tfsdk:"managed"`
	Config      types.Map    `tfsdk:"config"`
}

// NetworkResource represent LXD network resource.
type NetworkResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewNetworkResource returns a new network resource.
func NewNetworkResource() resource.Resource {
	return &NetworkResource{}
}

// Metadata for network resource.
func (r NetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network", req.ProviderTypeName)
}

// Schema for network resource.
func (r NetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("bridge", "macvlan", "sriov", "ovn", "physical"),
				},
			},

			"project": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"target": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"managed": schema.BoolAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *NetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r NetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(plan.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := plan.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := plan.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	// Convert network config to map.
	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	network := api.NetworksPost{
		Name: plan.Name.ValueString(),
		Type: plan.Type.ValueString(),
		NetworkPut: api.NetworkPut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
	}

	err = server.CreateNetwork(network)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network %q", network.Name), err.Error())
		return
	}

	_, diags = plan.Sync(ctx, server)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(state.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := state.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := state.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	found, diags := state.Sync(ctx, server)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Remove resource state if resource is not found.
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(plan.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := plan.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := plan.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	networkName := plan.Name.ValueString()
	network, etag, err := server.GetNetwork(networkName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network %q", networkName), err.Error())
		return
	}

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Merge network config state and user config.
	config := common.MergeConfig(network.Config, userConfig, plan.ComputedKeys())

	// Update network.
	newNetwork := api.NetworkPut{
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	err = server.UpdateNetwork(networkName, newNetwork, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network %q", networkName), err.Error())
		return
	}

	_, diags = plan.Sync(ctx, server)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(state.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := state.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := state.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	networkName := state.Name.ValueString()
	err = server.DeleteNetwork(networkName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network %q", networkName), err.Error())
	}
}

func (r NetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	remote, project, name, diag := common.SplitImportID(req.ID, "network")
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	if remote != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote"), remote)...)
	}

	if project != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

// Sync pulls network data from the server and updates the model in-place.
// It returns a boolean indicating whether resource is found and diagnostics
// that contain potential errors.
// This should be called before updating Terraform state.
func (m *NetworkModel) Sync(ctx context.Context, server lxd.InstanceServer) (bool, diag.Diagnostics) {
	networkName := m.Name.ValueString()
	network, _, err := server.GetNetwork(networkName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return false, nil
		}

		return true, diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve network %q", networkName), err.Error()),
		}
	}

	// Extract user defined config and merge it with current config state.
	usrConfig, diags := common.ToConfigMap(ctx, m.Config)
	if diags.HasError() {
		return true, diags
	}

	stateConfig := common.StripConfig(network.Config, usrConfig, m.ComputedKeys())

	// Convert config state into schema type.
	config, diags := common.ToConfigMapType(ctx, stateConfig)
	if diags.HasError() {
		return true, diags
	}

	m.Name = types.StringValue(network.Name)
	m.Description = types.StringValue(network.Description)
	m.Managed = types.BoolValue(network.Managed)
	m.Type = types.StringValue(network.Type)
	m.Config = config

	return true, nil
}

// ComputedKeys returns list of computed LXD config keys.
func (_ NetworkModel) ComputedKeys() []string {
	return []string{
		"bridge.mtu",
		"ipv4.address",
		"ipv4.nat",
		"ipv6.address",
		"ipv6.nat",
		"volatile.",
	}
}
