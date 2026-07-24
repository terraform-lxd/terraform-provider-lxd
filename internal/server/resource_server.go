package server

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	Remote types.String `tfsdk:"remote"`
	Config types.Map    `tfsdk:"config"`
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

			// Contains global server configuration keys. On a clustered server,
			// local keys are applied to every cluster member with the same value.
			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
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
// the keys tracked via "config". The updated model is then set as the new Terraform state.
func (r ServerResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m ServerModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	apiServer, _, err := server.GetServer()
	if err != nil {
		respDiags.AddError("Failed to retrieve LXD server configuration", err.Error())
		return respDiags
	}

	_, localKeys, err := serverConfigKeys(apiServer, server)
	if err != nil {
		respDiags.AddError("Failed to retrieve LXD server configuration metadata", err.Error())
		return respDiags
	}

	trackedConfig, diags := common.ToConfigMap(ctx, m.Config)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return respDiags
	}

	baseLive := serverConfigToStringMap(apiServer.Config)

	// On a clustered server, local keys are stored per cluster member. They are read back from
	// every member, because a divergence on any single member has to be detected as a drift.
	var memberLive []map[string]string
	if apiServer.Environment.ServerClustered {
		memberNames, err := server.GetClusterMemberNames()
		if err != nil {
			respDiags.AddError("Failed to retrieve LXD cluster member names", err.Error())
			return respDiags
		}

		for _, name := range memberNames {
			memberAPIServer, _, err := server.UseTarget(name).GetServer()
			if err != nil {
				respDiags.AddError(fmt.Sprintf("Failed to retrieve LXD server configuration for member %q", name), err.Error())
				return respDiags
			}

			memberLive = append(memberLive, serverConfigToStringMap(memberAPIServer.Config))
		}
	}

	config := make(map[string]string, len(trackedConfig))
	for k, v := range trackedConfig {
		if len(memberLive) == 0 || !slices.Contains(localKeys, k) {
			config[k] = baseLive[k]
			continue
		}

		// A local key is in sync only if every cluster member holds the tracked value.
		// If any member diverges, its value is recorded in the state, which makes the
		// difference visible in the plan and reapplies the key to all members.
		config[k] = v
		for _, live := range memberLive {
			if live[k] != v {
				config[k] = live[k]
				break
			}
		}
	}

	configValue, diags := types.MapValueFrom(ctx, types.StringType, config)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return respDiags
	}

	m.Config = configValue

	return tfState.Set(ctx, &m)
}

// apply writes the tracked configuration keys to the server. Global keys are set on the server
// directly, while local keys are set on every cluster member when the server is clustered.
// Keys absent from the model are left untouched.
func (r ServerResource) apply(ctx context.Context, server lxd.InstanceServer, m ServerModel) error {
	err := requireMetadataConfigExtension(server)
	if err != nil {
		return err
	}

	globalConfig, localConfig, clustered, err := m.splitConfig(ctx, server)
	if err != nil {
		return err
	}

	serverConfigMutex.Lock()
	defer serverConfigMutex.Unlock()

	if !clustered {
		merged := maps.Clone(globalConfig)
		maps.Copy(merged, localConfig)

		err := applyServerConfig(server, merged)
		if err != nil {
			return err
		}

		return nil
	}

	err = applyServerConfig(server, globalConfig)
	if err != nil {
		return err
	}

	memberNames, err := server.GetClusterMemberNames()
	if err != nil {
		return err
	}

	for _, name := range memberNames {
		err := applyServerConfig(server.UseTarget(name), localConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

// splitConfig classifies the model's "config" into global and local (member-specific) configuration
// and reports whether the server is clustered. It returns an error if any key is not a valid server
// configuration key.
func (m ServerModel) splitConfig(ctx context.Context, server lxd.InstanceServer) (globalConfig map[string]string, localConfig map[string]string, clustered bool, err error) {
	config, diags := common.ToConfigMap(ctx, m.Config)

	err = errors.FromDiagnostics(diags)
	if err != nil {
		return nil, nil, false, fmt.Errorf("Unable to convert server config to map: %v", err)
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		return nil, nil, false, err
	}

	globalKeys, localKeys, err := serverConfigKeys(apiServer, server)
	if err != nil {
		return nil, nil, false, err
	}

	globalConfig = make(map[string]string)
	localConfig = make(map[string]string)
	for k, v := range config {
		switch {
		case slices.Contains(localKeys, k):
			localConfig[k] = v
		case slices.Contains(globalKeys, k):
			globalConfig[k] = v
		case strings.HasPrefix(k, "user."):
			// Free-form "user." keys are accepted by LXD but are not
			// enumerated in the metadata configuration. They are global.
			globalConfig[k] = v
		default:
			return nil, nil, false, fmt.Errorf("Config key %q is not a valid server configuration key", k)
		}
	}

	return globalConfig, localConfig, apiServer.Environment.ServerClustered, nil
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

// serverConfigKeys returns the list of global (cluster-wide) and member-specific (local) server
// configuration keys, derived from the LXD server's metadata configuration.
// Read-only "volatile." keys are excluded from both lists.
func serverConfigKeys(apiServer *api.Server, server lxd.InstanceServer) (globalKeys []string, localKeys []string, err error) {
	meta, err := common.ServerMetadataConfiguration(apiServer.Environment.ServerVersion, server)
	if err != nil {
		return nil, nil, err
	}

	serverConfigs, ok := meta.Configs["server"]
	if !ok {
		return nil, nil, fmt.Errorf("Metadata configuration does not contain a %q section", "server")
	}

	for _, group := range serverConfigs {
		for _, keys := range group.Keys {
			for k, v := range keys {
				if strings.HasPrefix(k, "volatile.") {
					continue
				}

				if v.Scope == "local" {
					localKeys = append(localKeys, k)
					continue
				}

				globalKeys = append(globalKeys, k)
			}
		}
	}

	return globalKeys, localKeys, nil
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
