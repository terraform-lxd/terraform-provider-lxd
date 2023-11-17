package network

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// LxdNetworkResourceModel resource data model that matches the schema.
type LxdNetworkResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Target      types.String `tfsdk:"target"`
	Managed     types.Bool   `tfsdk:"managed"`
	Config      types.Map    `tfsdk:"config"`
	ConfigState types.Map    `tfsdk:"config_state"`
}

// Sync pulls network data from the server and updates the model in-place.
// It returns a boolean indicating whether resource is found and diagnostics
// that contain potential errors.
// This should be called before updating Terraform state.
func (m *LxdNetworkResourceModel) Sync(server lxd.InstanceServer, networkName string) (bool, diag.Diagnostics) {
	network, _, err := server.GetNetwork(networkName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return false, nil
		}

		return true, diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve network %q", networkName), err.Error()),
		}
	}

	config, diags := common.ToConfigMapType(context.Background(), network.Config)
	if diags.HasError() {
		return true, diags
	}

	m.Name = types.StringValue(network.Name)
	m.Description = types.StringValue(network.Description)
	m.Managed = types.BoolValue(network.Managed)
	m.Type = types.StringValue(network.Type)
	m.ConfigState = config

	return true, nil
}

// LxdNetworkResource represent LXD network resource.
type LxdNetworkResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewLxdNetworkResource returns a new network resource.
func NewLxdNetworkResource() resource.Resource {
	return &LxdNetworkResource{}
}

// Metadata for network resource.
func (r LxdNetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network", req.ProviderTypeName)
}

// Schema for network resource.
func (r LxdNetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			// Config represents user defined LXD config file.
			"config": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},

			// Config state represents actual LXD resource state.
			// It is managed solely by the provider. User config
			// is merged into it.
			"config_state": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *LxdNetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LxdNetworkResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	common.ModifyConfigStatePlan(ctx, req, resp, r.ComputedKeys())
}

func (r LxdNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *LxdNetworkResourceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	// Convert network config to map.
	config, diag := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	network := api.NetworksPost{
		Name: data.Name.ValueString(),
		NetworkPut: api.NetworkPut{
			Description: data.Description.ValueString(),
			Config:      config,
		},
	}

	err = server.CreateNetwork(network)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network %q", network.Name), err.Error())
		return
	}

	found, diags := data.Sync(server, network.Name)
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
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *LxdNetworkResourceModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	networkName := data.Name.ValueString()

	found, diags := data.Sync(server, networkName)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *LxdNetworkResourceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	networkName := data.Name.ValueString()
	_, etag, err := server.GetNetwork(networkName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network %q", networkName), err.Error())
		return
	}

	// Merge LXD state and user configs.
	userConfig, diags := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diags...)

	stateConfig, diags := common.ToConfigMap(ctx, data.ConfigState)
	resp.Diagnostics.Append(diags...)

	config := common.MergeConfig(stateConfig, userConfig, r.ComputedKeys())

	if resp.Diagnostics.HasError() {
		return
	}

	// Update network.
	network := api.NetworkPut{
		Description: data.Description.ValueString(),
		Config:      config,
	}

	err = server.UpdateNetwork(networkName, network, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network %q", networkName), err.Error())
		return
	}

	found, diags := data.Sync(server, networkName)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *LxdNetworkResourceModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	networkName := data.Name.ValueString()
	err = server.DeleteNetwork(networkName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network %q", networkName), err.Error())
	}
}

// ComputedKeys returns list of compuuted LXD config keys.
func (r LxdNetworkResource) ComputedKeys() []string {
	return []string{
		"ipv4.address",
		"ipv4.nat",
		"ipv6.address",
		"ipv6.nat",
	}
}
