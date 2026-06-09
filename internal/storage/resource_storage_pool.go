package storage

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
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

// StoragePoolModel represents a LXD storage pool.
type StoragePoolModel struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Driver          types.String `tfsdk:"driver"`
	Project         types.String `tfsdk:"project"`
	Remote          types.String `tfsdk:"remote"`
	Config          types.Map    `tfsdk:"config"`
	MemberOverrides types.Map    `tfsdk:"member_overrides"`
	Members         types.Map    `tfsdk:"members"`
}

// StoragePoolMemberModel represents a per-member storage pool configuration override.
type StoragePoolMemberModel struct {
	Config types.Map `tfsdk:"config"`
}

// StoragePoolResource represents LXD storage pool resource.
type StoragePoolResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewStoragePoolResource returns a new storage pool resource.
func NewStoragePoolResource() resource.Resource {
	return &StoragePoolResource{}
}

func (r StoragePoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_pool"
}

func (r StoragePoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"driver": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("dir", "zfs", "lvm", "btrfs", "ceph", "cephfs", "cephobject"),
				},
			},

			"project": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			// Contains global and default local (member-specific) storage pool configuration.
			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},

			// Contains only local (member-specific) storage pool configuration that
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
		},
	}
}

func (r *StoragePoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StoragePoolResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Nothing to do on destroy.
		return
	}

	var plan StoragePoolModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cannot expand members if driver or member_overrides are not yet known.
	if plan.Driver.IsUnknown() || plan.MemberOverrides.IsUnknown() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()

	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	driver := plan.Driver.ValueString()
	_, memberPoolConfigs, err := plan.ParsePoolConfigs(ctx, server, driver)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse storage pool configuration", err.Error())
		return
	}

	members := make(map[string]StoragePoolMemberModel, len(memberPoolConfigs))
	for memberName, memberConfig := range memberPoolConfigs {
		memberConfigType, diags := types.MapValueFrom(ctx, types.StringType, common.ToNullableConfig(memberConfig))
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		members[memberName] = StoragePoolMemberModel{Config: memberConfigType}
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

func (r StoragePoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StoragePoolModel

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

	poolName := plan.Name.ValueString()
	driver := plan.Driver.ValueString()

	poolConfig, memberPoolConfigs, err := plan.ParsePoolConfigs(ctx, server, driver)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse storage pool configuration", err.Error())
		return
	}

	// Create per-member pool definitions.
	for memberName, memberPoolConfig := range memberPoolConfigs {
		memberServer := server.UseTarget(memberName)

		memberPool := api.StoragePoolsPost{
			Name:   poolName,
			Driver: driver,
			StoragePoolPut: api.StoragePoolPut{
				Description: plan.Description.ValueString(),
				Config:      memberPoolConfig,
			},
		}

		op, err := memberServer.CreateStoragePool(memberPool)
		if err == nil {
			err = op.WaitContext(ctx)
		}

		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage pool %q on member %q", poolName, memberName), err.Error())
			return
		}

		diags := plan.TaintState(ctx, &resp.State)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	// Create cluster-wide pool definition.
	pool := api.StoragePoolsPost{
		Name:   poolName,
		Driver: driver,
		StoragePoolPut: api.StoragePoolPut{
			Description: plan.Description.ValueString(),
			Config:      poolConfig,
		},
	}

	op, err := server.CreateStoragePool(pool)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage pool %q", pool.Name), err.Error())
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

func (r StoragePoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StoragePoolModel

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

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r StoragePoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StoragePoolModel

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

	poolName := plan.Name.ValueString()
	driver := plan.Driver.ValueString()
	computedKeys := plan.ComputedKeys(driver)

	// Extract pool config from the plan.
	poolConfig, memberPoolConfigs, err := plan.ParsePoolConfigs(ctx, server, driver)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse storage pool configuration", err.Error())
		return
	}

	// Update all members present in the plan.
	for memberName, memberPoolConfig := range memberPoolConfigs {
		memberServer := server.UseTarget(memberName)

		pool, etag, err := memberServer.GetStoragePool(poolName)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve storage pool %q on member %q", poolName, memberName), err.Error())
			return
		}

		newConfig := common.MergeConfig(pool.Config, memberPoolConfig, computedKeys)
		poolUpdateReq := api.StoragePoolPut{
			Config: newConfig,
		}

		op, err := memberServer.UpdateStoragePool(poolName, poolUpdateReq, etag)
		if err == nil {
			err = op.WaitContext(ctx)
		}

		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to update storage pool %q on member %q", poolName, memberName), err.Error())
			return
		}
	}

	// Update the cluster-wide pool definition.
	pool, etag, err := server.GetStoragePool(poolName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing storage pool %q", poolName), err.Error())
		return
	}

	config := common.MergeConfig(pool.Config, poolConfig, computedKeys)
	poolUpdateReq := api.StoragePoolPut{
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	op, err := server.UpdateStoragePool(poolName, poolUpdateReq, etag)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update storage pool %q", poolName), err.Error())
		return
	}

	// Update Terraform state.
	diags := r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r StoragePoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StoragePoolModel

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

	poolName := state.Name.ValueString()
	op, err := server.DeleteStoragePool(poolName)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		// When clustered storage pool is removed, per target storage
		// pools will no longer exist.
		if errors.IsNotFoundError(err) {
			return
		}

		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove storage pool %q", poolName), err.Error())
	}
}

func (r StoragePoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "storage_pool",
		RequiredFields: []string{"name"},
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

// SyncState fetches the server's current state for a storage pool and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r StoragePoolResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m StoragePoolModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	poolName := m.Name.ValueString()
	pool, _, err := server.GetStoragePool(poolName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve storage pool %q", poolName), err.Error())
		return respDiags
	}

	// Extract storage pool member-specific configs.
	_, memberPoolConfigs, err := m.ParsePoolConfigs(ctx, server, pool.Driver)
	if err != nil {
		respDiags.AddError(fmt.Sprintf("Failed to parse storage pool %q configuration", poolName), err.Error())
		return respDiags
	}

	members := make(map[string]StoragePoolMemberModel, len(memberPoolConfigs))
	for memberName, memberConfig := range memberPoolConfigs {
		memberServer := server.UseTarget(memberName)

		memberPool, _, err := memberServer.GetStoragePool(poolName)
		if err != nil {
			respDiags.AddError(fmt.Sprintf("Failed to retrieve storage pool %q on member %q", poolName, memberName), err.Error())
			return respDiags
		}

		// Apply live values for each managed key.
		// Ignore source, which can differ from user input and cause state drift.
		for k := range memberConfig {
			v, ok := memberPool.Config[k]
			if ok && k != "source" {
				memberConfig[k] = v
			}
		}

		memberConfigType, diags := types.MapValueFrom(ctx, types.StringType, common.ToNullableConfig(memberConfig))
		if diags.HasError() {
			return diags
		}

		members[memberName] = StoragePoolMemberModel{Config: memberConfigType}
	}

	// LXD can modify the "source" config key, even if user provided the value.
	// This can cause state drift, therefore, remove it from the retrieved storage pool config
	// and persist user-defined value, if any.
	delete(pool.Config, "source")

	// Merge current storage pool configuration with user provided configuration, stripping away
	// computed fields that were not set by the user.
	poolConfig := common.StripConfig(pool.Config, m.Config, m.ComputedKeys(pool.Driver))
	configValue, diags := common.ToConfigMapType(ctx, poolConfig, m.Config)
	if diags.HasError() {
		return diags
	}

	memberObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"config": types.MapType{ElemType: types.StringType},
	}}

	membersValue, diags := types.MapValueFrom(ctx, memberObjType, members)
	if diags.HasError() {
		return diags
	}

	m.Name = types.StringValue(pool.Name)
	m.Description = types.StringValue(pool.Description)
	m.Driver = types.StringValue(pool.Driver)
	m.Config = configValue
	m.Members = membersValue

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

// TaintState marks the state with identity fields required to target the storage pool.
func (m StoragePoolModel) TaintState(ctx context.Context, tfState *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(tfState.SetAttribute(ctx, path.Root("name"), m.Name.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("project"), m.Project.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("remote"), m.Remote.ValueString())...)

	return diags
}

// ParsePoolConfigs separates global and member-specific storage pool configuration based on the
// server metadata. It returns two maps, a map of global storage pool configuration and a map
// containing local storage pool configuration for each member (merged with default local
// configuration from field "config").
func (m StoragePoolModel) ParsePoolConfigs(ctx context.Context, server lxd.InstanceServer, driver string) (poolConfig map[string]string, memberConfigs map[string]map[string]string, err error) {
	poolName := m.Name.ValueString()

	// Convert base pool config to map.
	poolConfig, diags := common.ToConfigMap(ctx, m.Config)
	err = errors.FromDiagnostics(diags)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to convert pool config to map: %v", err)
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		return nil, nil, err
	}

	serverVersion := apiServer.Environment.ServerVersion
	isServerClustered := apiServer.Environment.ServerClustered
	isDriverSupported := false

	for _, d := range apiServer.Environment.StorageSupportedDrivers {
		if d.Name == driver {
			isDriverSupported = true
			break
		}
	}

	if !isDriverSupported {
		return nil, nil, fmt.Errorf("Storage pool driver %q is not supported by the target LXD server", driver)
	}

	// Extract member-specific config keys from server metadata.
	// Use server version as metadata configuration cache key, as metadata configuration is the
	// same across LXD servers with the same version.
	allPoolKeys, localPoolKeys, err := m.storagePoolConfigKeys(serverVersion, server, driver)
	if err != nil {
		return nil, nil, err
	}

	if len(allPoolKeys) > 0 {
		for k := range poolConfig {
			if !slices.Contains(allPoolKeys, k) {
				return nil, nil, fmt.Errorf("Storage pool %q (%s) does not support config key %q", poolName, driver, k)
			}
		}
	}

	hasMemberOverrides := len(m.MemberOverrides.Elements()) > 0

	// Return early if LXD is not clustered.
	if !isServerClustered {
		if hasMemberOverrides {
			return nil, nil, fmt.Errorf("Storage pool %q (%s) member-specific config overrides are allowed only when LXD is clustered", poolName, driver)
		}

		// Return early with global storage pool config.
		return poolConfig, nil, nil
	}

	memberNames, err := server.GetClusterMemberNames()
	if err != nil {
		return nil, nil, err
	}

	// Separate global and member-specific pool configuration.
	memberPoolConfig := make(map[string]string)
	for k, v := range poolConfig {
		if slices.Contains(localPoolKeys, k) {
			memberPoolConfig[k] = v
			delete(poolConfig, k)
		}
	}

	// Set member-specific config from global config to all members by default.
	memberPoolConfigs := make(map[string]map[string]string)
	for _, memberName := range memberNames {
		memberPoolConfigs[memberName] = maps.Clone(memberPoolConfig)
	}

	// Extract member-specific config overrides.
	memberOverrides := map[string]StoragePoolMemberModel{}
	err = errors.FromDiagnostics(m.MemberOverrides.ElementsAs(ctx, &memberOverrides, true))
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to extract member-specific config overrides: %v", err)
	}

	for memberName, override := range memberOverrides {
		memberPoolConfig, ok := memberPoolConfigs[memberName]
		if !ok {
			return nil, nil, fmt.Errorf("Storage pool %q (%s) contains member-specific config override for a non-existing cluster member %q!", poolName, driver, memberName)
		}

		// Parse and apply member-specific override.
		configMap, diags := common.ToConfigMap(ctx, override.Config)
		err := errors.FromDiagnostics(diags)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable to convert member-specific config override to map: %v", err)
		}

		maps.Copy(memberPoolConfig, configMap)

		// Ensure member-specific config does not contain global keys.
		for k := range memberPoolConfig {
			if !slices.Contains(localPoolKeys, k) {
				return nil, nil, fmt.Errorf("Invalid config key %q for storage pool member %q: Only member-specific keys are allowed in per-member configuration", k, memberName)
			}
		}

		// Store resolved config.
		memberPoolConfigs[memberName] = memberPoolConfig
	}

	return poolConfig, memberPoolConfigs, nil
}

// storagePoolConfigKeys retrieves a map of storage pool configuration keys and their scope.
func (m StoragePoolModel) storagePoolConfigKeys(serverName string, server lxd.InstanceServer, driver string) (allKeys []string, localKeys []string, err error) {
	if server.CheckExtension("metadata_configuration") != nil {
		localKeys = m.MemberSpecificKeys(driver)
		return nil, localKeys, nil
	}

	meta, err := common.ServerMetadataConfiguration(serverName, server)
	if err != nil {
		return nil, nil, err
	}

	driverConfigKey := "storage-" + driver
	driverConfig, ok := meta.Configs[driverConfigKey]
	if !ok {
		return nil, nil, fmt.Errorf("Metadata configuration %q not found", driverConfigKey)
	}

	// Parse pool config keys.
	poolConfigKey := "pool-conf"
	poolConfig, ok := driverConfig[poolConfigKey]
	if !ok {
		return nil, nil, fmt.Errorf("Metadata configuration %q does not contain %q keys", driverConfigKey, poolConfigKey)
	}

	for _, configKeys := range poolConfig.Keys {
		for k, v := range configKeys {
			allKeys = append(allKeys, k)
			if v.Scope == "local" {
				localKeys = append(localKeys, k)
			}
		}
	}

	// Parse volume config keys.
	volConfig, ok := driverConfig["volume-conf"]
	if ok {
		for _, configKeys := range volConfig.Keys {
			for k, v := range configKeys {
				// Pool accepts volume keys only with the "volume." prefix.
				key := "volume." + k

				allKeys = append(allKeys, key)
				if v.Scope == "local" {
					localKeys = append(localKeys, key)
				}
			}
		}
	}

	return allKeys, localKeys, nil
}

// ComputedKeys returns list of computed config keys.
func (m StoragePoolModel) ComputedKeys(driver string) []string {
	var keys []string

	switch driver {
	case "dir":
		keys = []string{
			"source",
		}
	case "zfs":
		keys = []string{
			"source",
			"size",
			"zfs.pool_name",
		}
	case "lvm":
		keys = []string{
			"source",
			"size",
			"lvm.vg_name",
			"lvm.thinpool_name",
		}
	case "btrfs":
		keys = []string{
			"source",
			"size",
		}
	case "ceph":
		keys = []string{
			"source",
			"ceph.cluster_name",
			"ceph.user.name",
			"ceph.osd.pg_num",
			"ceph.osd.pool_name",
			"ceph.osd.pool_size",
		}
	case "cephfs":
		keys = []string{
			"source",
			"cephfs.cluster_name",
			"cephfs.user.name",
			"cephfs.osd_pg_num",
			"cephfs.osd_pool_size",
		}
	case "cephobject":
		keys = []string{
			"cephobject.cluster_name",
			"cephobject.user.name",
		}
	}

	return append(keys, "volatile.")
}

// MemberSpecificKeys returns list of member-specific config keys.
// For storage pool drivers that do not have member-specific keys, nil is returned.
//
// This is mainly used for LXD servers that do not support metadata configuration
// endpoint, which allows to determine member-specific config keys dynamically.
func (m StoragePoolModel) MemberSpecificKeys(driver string) []string {
	switch driver {
	case "dir":
		return []string{
			"source",
			"source.recover",
		}
	case "zfs":
		return []string{
			"size",
			"source",
			"source.recover",
			"source.wipe",
			"zfs.pool_name",
		}
	case "lvm":
		return []string{
			"lvm.thinpool_name",
			"lvm.vg_name",
			"size",
			"source",
			"source.recover",
			"source.wipe",
		}
	case "btrfs":
		return []string{
			"size",
			"source",
			"source.recover",
			"source.wipe",
		}
	case "ceph":
		return []string{
			"source",
			"source.recover",
		}
	case "cephfs":
		return []string{
			"source",
			"source.recover",
		}
	default:
		return nil
	}
}
