package utils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ModifyConfigStatePlan is used to determines which configuration changes
// are computed by LXD and left untouched, and which should be considered
// modified externally and therefore removed. User defined configuration
// has precedence.
func ModifyConfigStatePlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, computedKeys []string) {
	// Ignore plan modification on create (state == nil)
	// or destroy (plan == nil).
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	// Read user defined configuration, and config_state configuration
	// from Terraform state.
	var config, configState map[string]string
	req.Config.GetAttribute(ctx, path.Root("config"), &config)
	req.State.GetAttribute(ctx, path.Root("config_state"), &configState)

	// Final config_state plan should be evaluated from user defined
	// configuration and current config_state.
	newConfigState := ComputeConfig(configState, config, computedKeys)
	resp.Plan.SetAttribute(ctx, path.Root("config_state"), newConfigState)
}

// ComputeConfig merges resource (existing) configuration with user defined
// configuration. Resource config entries that are contained in computed keys
// are inserted in the user config as they are treated as computed values.
func ComputeConfig(resConfig map[string]string, usrConfig map[string]string, computedKeys []string) map[string]string {
	config := make(map[string]string)

	// Add new (user) entries to the config.
	for k, v := range usrConfig {
		// Empty values in LXD configuration are considered
		// null (unset).
		if v != "" {
			config[k] = v
		}
	}

	// Add computed entries to the config. Computed entries are those
	// that are not present in user defined configuration, but its key
	// is contained within computedKeys.
	for k, v := range resConfig {
		_, ok := usrConfig[k]
		if !ok && v != "" && ValueInSlice(k, computedKeys) {
			config[k] = v
		}
	}

	return config
}

// ToStringTypeMap convert types.Map into map[string]types.String.
func toStringTypeMap(ctx context.Context, m types.Map) (map[string]types.String, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return make(map[string]types.String), nil
	}

	config := make(map[string]types.String, len(m.Elements()))
	diags := m.ElementsAs(ctx, &config, false)
	return config, diags
}

// ToStringMap convert types.Map into map[string]string.
func ToStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	raw, diags := toStringTypeMap(ctx, m)
	if diags.HasError() {
		return nil, diags
	}

	config := make(map[string]string, len(raw))
	for k, v := range raw {
		if v.IsNull() || v.IsUnknown() {
			// config[k] = ""
		} else {
			config[k] = v.ValueString()
		}
	}

	return config, diags
}

// ToStringMapType convert map[string]string into types.Map.
func ToStringMapType(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	return types.MapValueFrom(ctx, types.StringType, m)
}
