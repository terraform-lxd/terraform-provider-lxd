package network

import (
	"context"
	"fmt"
	"maps"
	"slices"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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

// NetworkModel resource data model that matches the schema.
type NetworkModel struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Type            types.String `tfsdk:"type"`
	Project         types.String `tfsdk:"project"`
	Remote          types.String `tfsdk:"remote"`
	Config          types.Map    `tfsdk:"config"`
	MemberOverrides types.Map    `tfsdk:"member_overrides"`
	Members         types.Map    `tfsdk:"members"`

	// Computed.
	Managed types.Bool   `tfsdk:"managed"`
	IPv4    types.String `tfsdk:"ipv4_address"`
	IPv6    types.String `tfsdk:"ipv6_address"`
}

// NetworkMemberModel represents a per-member network configuration override.
type NetworkMemberModel struct {
	Config types.Map `tfsdk:"config"`
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
				Default:  stringdefault.StaticString("bridge"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("bridge", "macvlan", "sriov", "ovn", "physical"),
				},
			},

			"project": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(provider_config.DefaultProject),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			// Contains global and default local (member-specific) network configuration.
			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},

			// Contains only local (member-specific) network configuration that
			// overrides the default values defined in "config".
			"member_overrides": schema.MapNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"config": schema.MapAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},

			// Contains the resolved local (member-specific) config for all cluster members.
			"members": schema.MapNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"config": schema.MapAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},

			"managed": schema.BoolAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},

			"ipv4_address": schema.StringAttribute{
				Computed: true,
			},

			"ipv6_address": schema.StringAttribute{
				Computed: true,
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

func (r *NetworkResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Nothing to do on destroy.
		return
	}

	var plan NetworkModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cannot expand members if type or member_overrides are not yet known.
	if plan.Type.IsUnknown() || plan.MemberOverrides.IsUnknown() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()

	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	networkType := plan.Type.ValueString()
	_, memberNetworkConfigs, err := plan.ParseNetworkConfigs(ctx, server, networkType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse network configuration", err.Error())
		return
	}

	members := make(map[string]NetworkMemberModel, len(memberNetworkConfigs))
	for memberName, memberConfig := range memberNetworkConfigs {
		memberConfigType, diags := types.MapValueFrom(ctx, types.StringType, common.ToNullableConfig(memberConfig))
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		members[memberName] = NetworkMemberModel{Config: memberConfigType}
	}

	memberObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"config": types.MapType{ElemType: types.StringType},
	}}

	membersValue, diags := types.MapValueFrom(ctx, memberObjType, members)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Plan.SetAttribute(ctx, path.Root("members"), membersValue)
}

func (r NetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	networkName := plan.Name.ValueString()
	networkType := plan.Type.ValueString()

	networkConfig, memberNetworkConfigs, err := plan.ParseNetworkConfigs(ctx, server, networkType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse network configuration", err.Error())
		return
	}

	// Create per-member network definitions.
	for memberName, memberNetworkConfig := range memberNetworkConfigs {
		memberServer := server.UseTarget(memberName)

		memberNetwork := api.NetworksPost{
			Name: networkName,
			Type: networkType,
			NetworkPut: api.NetworkPut{
				Config: memberNetworkConfig,
			},
		}

		op, err := memberServer.CreateNetwork(memberNetwork)
		if err == nil {
			err = op.WaitContext(ctx)
		}

		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network %q on member %q", networkName, memberName), err.Error())
			return
		}

		diags := plan.TaintState(ctx, &resp.State)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	// Create cluster-wide network definition.
	network := api.NetworksPost{
		Name: networkName,
		Type: networkType,
		NetworkPut: api.NetworkPut{
			Description: plan.Description.ValueString(),
			Config:      networkConfig,
		},
	}

	op, err := server.CreateNetwork(network)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network %q", network.Name), err.Error())
		return
	}

	diags = plan.TaintState(ctx, &resp.State)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
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

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	networkName := plan.Name.ValueString()
	networkType := plan.Type.ValueString()
	computedKeys := plan.ComputedKeys()

	// Extract network config from the plan.
	networkConfig, memberNetworkConfigs, err := plan.ParseNetworkConfigs(ctx, server, networkType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse network configuration", err.Error())
		return
	}

	// Update all members present in the plan.
	for memberName, memberNetworkConfig := range memberNetworkConfigs {
		memberServer := server.UseTarget(memberName)

		network, etag, err := memberServer.GetNetwork(networkName)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve network %q on member %q", networkName, memberName), err.Error())
			return
		}

		newConfig := common.MergeConfig(network.Config, memberNetworkConfig, computedKeys)
		networkUpdateReq := api.NetworkPut{
			Config: newConfig,
		}

		op, err := memberServer.UpdateNetwork(networkName, networkUpdateReq, etag)
		if err == nil {
			err = op.WaitContext(ctx)
		}

		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network %q on member %q", networkName, memberName), err.Error())
			return
		}
	}

	// Update the cluster-wide network definition.
	network, etag, err := server.GetNetwork(networkName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network %q", networkName), err.Error())
		return
	}

	config := common.MergeConfig(network.Config, networkConfig, computedKeys)
	networkUpdateReq := api.NetworkPut{
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	op, err := server.UpdateNetwork(networkName, networkUpdateReq, etag)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network %q", networkName), err.Error())
		return
	}

	// Update Terraform state.
	diags := r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	networkName := state.Name.ValueString()
	op, err := server.DeleteNetwork(networkName)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		// When clustered network is removed, per member networks
		// will no longer exist.
		if errors.IsNotFoundError(err) {
			return
		}

		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network %q", networkName), err.Error())
	}
}

func (r NetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "network",
		RequiredFields: []string{"name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	if fields["project"] == "" {
		fields["project"] = provider_config.DefaultProject
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

// SyncState fetches the server's current state for a network and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r NetworkResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m NetworkModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	networkName := m.Name.ValueString()
	network, _, err := server.GetNetwork(networkName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve network %q", networkName), err.Error())
		return respDiags
	}

	networkState, err := server.GetNetworkState(networkName)
	if err != nil && !errors.IsNotFoundError(err) {
		respDiags.AddError(fmt.Sprintf("Failed to retrieve state of network %q", networkName), err.Error())
		return respDiags
	}

	var ipv4, ipv6 string
	if networkState != nil {
		ipv4, ipv6 = findGlobalCIDRs(networkState.Addresses)
	}

	// Extract network member-specific configs.
	_, memberNetworkConfigs, err := m.ParseNetworkConfigs(ctx, server, network.Type)
	if err != nil {
		respDiags.AddError(fmt.Sprintf("Failed to parse network %q configuration", networkName), err.Error())
		return respDiags
	}

	members := make(map[string]NetworkMemberModel, len(memberNetworkConfigs))
	for memberName, memberConfig := range memberNetworkConfigs {
		memberServer := server.UseTarget(memberName)

		memberNetwork, _, err := memberServer.GetNetwork(networkName)
		if err != nil {
			respDiags.AddError(fmt.Sprintf("Failed to retrieve network %q on member %q", networkName, memberName), err.Error())
			return respDiags
		}

		// Apply live values for each managed key.
		for k := range memberConfig {
			v, ok := memberNetwork.Config[k]
			if ok {
				memberConfig[k] = v
			}
		}

		memberConfigType, diags := types.MapValueFrom(ctx, types.StringType, common.ToNullableConfig(memberConfig))
		if diags.HasError() {
			return diags
		}

		members[memberName] = NetworkMemberModel{Config: memberConfigType}
	}

	// Merge current network configuration with user provided configuration, stripping away
	// computed fields that were not set by the user.
	networkConfig := common.StripConfig(network.Config, m.Config, m.ComputedKeys())
	configValue, diags := common.ToConfigMapType(ctx, networkConfig, m.Config)
	if diags.HasError() {
		return diags
	}

	memberObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"config": types.MapType{ElemType: types.StringType},
		},
	}

	membersValue, diags := types.MapValueFrom(ctx, memberObjType, members)
	if diags.HasError() {
		return diags
	}

	m.Name = types.StringValue(network.Name)
	m.Description = types.StringValue(network.Description)
	m.Managed = types.BoolValue(network.Managed)
	m.Type = types.StringValue(network.Type)
	m.Config = configValue
	m.Members = membersValue

	m.IPv4 = types.StringValue(ipv4)
	m.IPv6 = types.StringValue(ipv6)

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

// TaintState marks the state with identity fields required to target the network.
func (m NetworkModel) TaintState(ctx context.Context, tfState *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(tfState.SetAttribute(ctx, path.Root("name"), m.Name.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("project"), m.Project.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("remote"), m.Remote.ValueString())...)

	return diags
}

// ParseNetworkConfigs separates global and member-specific network configuration based on the
// server metadata. It returns two maps, a map of global network configuration and a map
// containing local network configuration for each member (merged with default local
// configuration from field "config").
func (m NetworkModel) ParseNetworkConfigs(ctx context.Context, server lxd.InstanceServer, networkType string) (networkConfig map[string]string, memberConfigs map[string]map[string]string, err error) {
	networkName := m.Name.ValueString()

	// Convert base network config to map.
	networkConfig, diags := common.ToConfigMap(ctx, m.Config)
	err = errors.FromDiagnostics(diags)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to convert network config to map: %v", err)
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		return nil, nil, err
	}

	serverVersion := apiServer.Environment.ServerVersion
	isServerClustered := apiServer.Environment.ServerClustered

	// Extract member-specific config keys from server metadata.
	// Use server version as metadata configuration cache key, as metadata configuration is the
	// same across LXD servers with the same version.
	allNetworkKeys, localNetworkKeys, err := m.networkConfigKeys(serverVersion, server, networkType)
	if err != nil {
		return nil, nil, err
	}

	if len(allNetworkKeys) > 0 {
		for k := range networkConfig {
			if !slices.Contains(allNetworkKeys, k) {
				return nil, nil, fmt.Errorf("Network %q (%s) does not support config key %q", networkName, networkType, k)
			}
		}
	}

	hasMemberOverrides := len(m.MemberOverrides.Elements()) > 0

	// Return early if LXD is not clustered or network type is OVN.
	if !isServerClustered || networkType == "ovn" {
		if hasMemberOverrides {
			return nil, nil, fmt.Errorf(`Network %q (%s) cannot use member-specific config overrides unless LXD is clustered and the network type is not "ovn"`, networkName, networkType)
		}

		// Return early with global network config.
		return networkConfig, nil, nil
	}

	memberNames, err := server.GetClusterMemberNames()
	if err != nil {
		return nil, nil, err
	}

	// Separate global and member-specific network configuration.
	memberNetworkConfig := make(map[string]string)
	for k, v := range networkConfig {
		if slices.Contains(localNetworkKeys, k) {
			memberNetworkConfig[k] = v
			delete(networkConfig, k)
		}
	}

	// Set member-specific config from global config to all members by default.
	memberNetworkConfigs := make(map[string]map[string]string)
	for _, memberName := range memberNames {
		memberNetworkConfigs[memberName] = maps.Clone(memberNetworkConfig)
	}

	// Extract member-specific config overrides.
	memberOverrides := map[string]NetworkMemberModel{}
	err = errors.FromDiagnostics(m.MemberOverrides.ElementsAs(ctx, &memberOverrides, true))
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to extract member-specific config overrides: %v", err)
	}

	for memberName, override := range memberOverrides {
		memberNetworkConfig, ok := memberNetworkConfigs[memberName]
		if !ok {
			return nil, nil, fmt.Errorf("Network %q (%s) contains member-specific config override for a non-existent cluster member %q!", networkName, networkType, memberName)
		}

		// Parse and apply member-specific override.
		configMap, diags := common.ToConfigMap(ctx, override.Config)
		err := errors.FromDiagnostics(diags)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable to convert member-specific config override to map: %v", err)
		}

		maps.Copy(memberNetworkConfig, configMap)

		// Ensure member-specific config does not contain global keys.
		for k := range memberNetworkConfig {
			if !slices.Contains(localNetworkKeys, k) {
				return nil, nil, fmt.Errorf("Invalid config key %q for network member %q: Only member-specific keys are allowed in per-member configuration", k, memberName)
			}
		}

		// Store resolved config.
		memberNetworkConfigs[memberName] = memberNetworkConfig
	}

	return networkConfig, memberNetworkConfigs, nil
}

// networkConfigKeys retrieves a list of network configuration keys and their scope.
func (m NetworkModel) networkConfigKeys(serverName string, server lxd.InstanceServer, networkType string) (allKeys []string, localKeys []string, err error) {
	if server.CheckExtension("metadata_configuration") != nil {
		localKeys = m.MemberSpecificKeys(networkType)
		return nil, localKeys, nil
	}

	meta, err := common.ServerMetadataConfiguration(serverName, server)
	if err != nil {
		return nil, nil, err
	}

	typeConfigKey := "network-" + networkType
	typeConfig, ok := meta.Configs[typeConfigKey]
	if !ok {
		return nil, nil, fmt.Errorf("Metadata configuration %q not found", typeConfigKey)
	}

	networkConfigKey := "network-conf"
	networkConfig, ok := typeConfig[networkConfigKey]
	if !ok {
		return nil, nil, fmt.Errorf("Metadata configuration %q does not contain %q key", typeConfigKey, networkConfigKey)
	}

	for _, configKeys := range networkConfig.Keys {
		for k, v := range configKeys {
			allKeys = append(allKeys, k)
			if v.Scope == "local" {
				localKeys = append(localKeys, k)
			}
		}
	}

	return allKeys, localKeys, nil
}

// ComputedKeys returns list of computed LXD config keys.
func (m NetworkModel) ComputedKeys() []string {
	return []string{
		"bridge.mtu",
		"ipv4.address",
		"ipv4.nat",
		"ipv6.address",
		"ipv6.nat",
		"volatile.",
	}
}

// MemberSpecificKeys returns list of member-specific config keys for the given network type.
// For network types that do not have member-specific keys, nil is returned.
//
// This is mainly used for LXD servers that do not support the metadata configuration endpoint,
// which allows determining member-specific config keys dynamically (LXD <= 5.0).
func (m NetworkModel) MemberSpecificKeys(networkType string) []string {
	switch networkType {
	case "bridge":
		return []string{
			"bgp.ipv4.nexthop",
			"bgp.ipv6.nexthop",
			"bridge.external_interfaces",
		}
	case "macvlan":
		return []string{"parent"}
	case "physical":
		return []string{"parent"}
	case "sriov":
		return []string{"parent"}
	default:
		return nil
	}
}

// findGlobalCIDRs returns first global IPv4 and IPv6 network addresses (CIDRs)
// of the provided network interface. If an IP address is not found, an empty
// string is returned.
func findGlobalCIDRs(addrs []api.NetworkStateAddress) (ipv4 string, ipv6 string) {
	for _, addr := range addrs {
		if addr.Scope != "global" {
			continue
		}

		ip := addr.Address + "/" + addr.Netmask

		if ipv4 == "" && addr.Family == "inet" {
			ipv4 = ip
		}

		if ipv6 == "" && addr.Family == "inet6" {
			ipv6 = ip
		}
	}

	return ipv4, ipv6
}
