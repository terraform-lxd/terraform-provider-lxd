package server

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// metadataConfigExtension is the API extension that exposes the server
// configuration metadata used to validate and classify config keys.
const metadataConfigExtension = "metadata_configuration"

// serverConfigMutex serializes all changes made by lxd_server resources. LXD server configuration
// is updated by reading the whole configuration, modifying it and writing it back behind an etag.
// Without serialization, concurrent updates to different keys (possibly from different lxd_server
// resources) can race and overwrite each other.
var serverConfigMutex sync.Mutex

// ServerModel represents LXD server resource.
type ServerModel struct {
	Remote          types.String `tfsdk:"remote"`
	Config          types.Map    `tfsdk:"config"`
	MemberOverrides types.Map    `tfsdk:"member_overrides"`
	Members         types.Map    `tfsdk:"members"`
}

// ServerResource represents LXD server resource.
type ServerResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewServerResource returns a new server resource.
func NewServerResource() resource.Resource {
	return &ServerResource{}
}

func (r ServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r ServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// Contains global and default local (member-specific) server configuration.
			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},

			"member_overrides": common.MemberOverridesAttribute(),

			"members": common.MembersAttribute(),
		},
	}
}

func (r *ServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Nothing to do on destroy.
		return
	}

	var plan ServerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cannot expand members if remote or member_overrides are not yet known, or if config
	// (global or within a member override) contains a value that is only known
	// after apply (e.g. sourced from a resource applied later in the same plan).
	if plan.Remote.IsUnknown() || plan.MemberOverrides.IsUnknown() || common.ConfigHasUnknownValue(plan.Config) || common.MemberOverridesHaveUnknownConfig(ctx, plan.MemberOverrides) {
		return
	}

	server, err := r.provider.InstanceServer(plan.Remote.ValueString(), "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	_, memberConfigs, _, err := plan.ParseServerConfigs(ctx, server)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse LXD server configuration", err.Error())
		return
	}

	membersValue, diags := common.ToMembersMapType(ctx, memberConfigs)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Plan.SetAttribute(ctx, path.Root("members"), membersValue)
}

func (r ServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServerModel

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

	err = r.apply(ctx, server, plan)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update LXD server (%q) configuration", remote), err.Error())
		return
	}

	resp.Diagnostics.Append(r.SyncState(ctx, &resp.State, server, plan)...)
}

func (r ServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(state.Remote.ValueString(), "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	err = requireMetadataConfigExtension(server)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported LXD server", err.Error())
		return
	}

	resp.Diagnostics.Append(r.SyncState(ctx, &resp.State, server, state)...)
}

func (r ServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServerModel

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

	err = r.apply(ctx, server, plan)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update LXD server (%q) configuration", remote), err.Error())
		return
	}

	resp.Diagnostics.Append(r.SyncState(ctx, &resp.State, server, plan)...)
}

// Delete leaves the live server configuration untouched and only stops tracking the managed keys.
func (r ServerResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// SyncState fetches the LXD server's current configuration and updates the model, keeping only
// the keys tracked via "config" and "member_overrides". On a clustered server, local keys are
// read back from every cluster member into "members". The updated model is then set as the new
// Terraform state.
func (r ServerResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m ServerModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	apiServer, _, err := server.GetServer()
	if err != nil {
		respDiags.AddError("Failed to retrieve LXD server configuration", err.Error())
		return respDiags
	}

	configKeys, err := serverConfigKeys(apiServer, server)
	if err != nil {
		respDiags.AddError("Failed to retrieve LXD server configuration metadata", err.Error())
		return respDiags
	}

	trackedConfig, diags := common.ToConfigMap(ctx, m.Config)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return respDiags
	}

	// Extract server member-specific configs.
	_, memberConfigs, clustered, err := m.ParseServerConfigs(ctx, server)
	if err != nil {
		respDiags.AddError("Failed to parse LXD server configuration", err.Error())
		return respDiags
	}

	for memberName, memberConfig := range memberConfigs {
		memberAPIServer, _, err := server.UseTarget(memberName).GetServer()
		if err != nil {
			respDiags.AddError(fmt.Sprintf("Failed to retrieve LXD server configuration for member %q", memberName), err.Error())
			return respDiags
		}

		// Apply live values for each managed key.
		memberLive := serverConfigToStringMap(memberAPIServer.Config)
		for k := range memberConfig {
			memberConfig[k] = memberLive[k]
		}
	}

	// An untargeted request reports the global config merged with the local config of the
	// member that answered it, so local keys are read per member instead of from here.
	baseLive := serverConfigToStringMap(apiServer.Config)

	config := make(map[string]string, len(trackedConfig))
	for k, v := range trackedConfig {
		if clustered && configKeys.IsLocal(k) {
			config[k] = v
			continue
		}

		config[k] = baseLive[k]
	}

	configValue, diags := types.MapValueFrom(ctx, types.StringType, config)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return respDiags
	}

	membersValue, diags := common.ToMembersMapType(ctx, memberConfigs)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return respDiags
	}

	m.Config = configValue
	m.Members = membersValue

	return tfState.Set(ctx, &m)
}

// apply writes the tracked configuration keys to the server. Global keys are set on the server
// directly, while local keys are set on each cluster member with that member's resolved config.
// Keys absent from the model are left untouched.
func (r ServerResource) apply(ctx context.Context, server lxd.InstanceServer, m ServerModel) error {
	err := requireMetadataConfigExtension(server)
	if err != nil {
		return err
	}

	globalConfig, memberConfigs, _, err := m.ParseServerConfigs(ctx, server)
	if err != nil {
		return err
	}

	serverConfigMutex.Lock()
	defer serverConfigMutex.Unlock()

	err = applyServerConfig(server, globalConfig)
	if err != nil {
		return err
	}

	for memberName, memberConfig := range memberConfigs {
		err := applyServerConfig(server.UseTarget(memberName), memberConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

// ParseServerConfigs separates global and member-specific server configuration based on the
// server metadata, and reports whether the server is clustered. It returns a map of global
// server configuration and a map containing local server configuration for each member (merged
// with default local configuration from field "config"). On a non-clustered server all keys are
// returned as global configuration.
func (m ServerModel) ParseServerConfigs(ctx context.Context, server lxd.InstanceServer) (globalConfig map[string]string, memberConfigs map[string]map[string]string, clustered bool, err error) {
	config, diags := common.ToConfigMap(ctx, m.Config)

	err = errors.FromDiagnostics(diags)
	if err != nil {
		return nil, nil, false, fmt.Errorf("Unable to convert server config to map: %v", err)
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		return nil, nil, false, err
	}

	configKeys, err := serverConfigKeys(apiServer, server)
	if err != nil {
		return nil, nil, false, err
	}

	for key := range config {
		_, ok := configKeys.Lookup(key)
		if !ok && !strings.HasPrefix(key, "user.") {
			return nil, nil, false, fmt.Errorf("Config key %q is not a valid server configuration key", key)
		}
	}

	clustered = apiServer.Environment.ServerClustered
	hasMemberOverrides := len(m.MemberOverrides.Elements()) > 0

	// Return early if LXD is not clustered. Local keys then apply to the single
	// server and are set together with the global ones.
	if !clustered {
		if hasMemberOverrides {
			return nil, nil, false, fmt.Errorf("LXD server member-specific config overrides are allowed only when LXD is clustered")
		}

		return config, nil, false, nil
	}

	memberNames, err := server.GetClusterMemberNames()
	if err != nil {
		return nil, nil, false, err
	}

	globalConfig, memberConfigs, err = common.ResolveMemberConfigs(ctx, "LXD server", config, m.MemberOverrides, memberNames, configKeys)
	if err != nil {
		return nil, nil, false, err
	}

	return globalConfig, memberConfigs, true, nil
}

// requireMetadataConfigExtension returns an error if the LXD server does not support the metadata
// configuration API extension. This resource relies on that extension to classify configuration
// keys and must not be used without it.
func requireMetadataConfigExtension(server lxd.InstanceServer) error {
	if server.CheckExtension(metadataConfigExtension) != nil {
		return fmt.Errorf("LXD server does not support the %q API extension, which is required to manage server configuration", metadataConfigExtension)
	}

	return nil
}

// serverConfigKeys returns writable server configuration keys and their scope.
func serverConfigKeys(apiServer *api.Server, server lxd.InstanceServer) (common.MetadataConfigKeys, error) {
	meta, err := common.ServerMetadataConfiguration(apiServer.Environment.ServerVersion, server)
	if err != nil {
		return common.MetadataConfigKeys{}, err
	}

	serverConfigs, ok := meta.Configs["server"]
	if !ok {
		return common.MetadataConfigKeys{}, fmt.Errorf("Metadata configuration does not contain a %q section", "server")
	}

	keys := api.MetadataConfigurationConfigKeys{}
	for _, group := range serverConfigs {
		for _, groupKeys := range group.Keys {
			writableKeys := make(map[string]api.MetadataConfigurationConfigKey, len(groupKeys))
			for k, v := range groupKeys {
				if strings.HasPrefix(k, "volatile.") {
					continue
				}

				writableKeys[k] = v
			}

			if len(writableKeys) > 0 {
				keys.Keys = append(keys.Keys, writableKeys)
			}
		}
	}

	return common.NewMetadataConfigKeys(common.MetadataConfigKeySource{Keys: keys}), nil
}

// applyServerConfig overlays config on top of the server's current configuration and applies
// the result. Keys absent from config are left untouched. If config is empty, no request is made.
func applyServerConfig(server lxd.InstanceServer, config map[string]string) error {
	if len(config) == 0 {
		return nil
	}

	apiServer, etag, err := server.GetServer()
	if err != nil {
		return err
	}

	// Start from the server's writable representation so that fields other than
	// the configuration are preserved.
	newServer := apiServer.Writable()

	// Writable returns the server's configuration map by reference, therefore it
	// has to be cloned before being modified.
	newServer.Config = maps.Clone(newServer.Config)
	if newServer.Config == nil {
		newServer.Config = make(map[string]any, len(config))
	}

	for k, v := range config {
		newServer.Config[k] = v
	}

	return server.UpdateServer(newServer, etag)
}

// serverConfigToStringMap converts a LXD server configuration map into a map[string]string.
// Non-string values are not expected, but are converted using their default string representation
// to avoid losing data.
func serverConfigToStringMap(config map[string]any) map[string]string {
	result := make(map[string]string, len(config))

	for k, v := range config {
		s, ok := v.(string)
		if !ok {
			s = fmt.Sprintf("%v", v)
		}

		result[k] = s
	}

	return result
}
