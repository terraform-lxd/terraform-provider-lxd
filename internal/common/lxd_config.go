package common

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ModifyConfigStatePlan is used to determines which configuration changes
// are computed by LXD and left untouched, and which should be considered
// modified externally and therefore removed. User defined configuration
// has precedence and is always retained.
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
	newConfigState := MergeConfig(configState, config, computedKeys)
	resp.Plan.SetAttribute(ctx, path.Root("config_state"), newConfigState)
}

// MergeConfig merges resource (existing) configuration with user defined
// configuration. Non-empty resource config entries that are contained in
// the provided computed keys are inserted in the user config.
func MergeConfig(resConfig map[string]string, usrConfig map[string]string, computedKeys []string) map[string]string {
	config := make(map[string]string)

	// Add user defined entries to the config.
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
		if !ok && v != "" && isComputedKey(k, computedKeys) {
			config[k] = v
		}
	}

	return config
}

// isComputedKey determines if a given key is considered "computed".
// A key is considered computed in two scenarios:
//  1. It exactly matches one of the computed keys.
//  2. It starts with any of the computed keys that end with a dot.
//
// For example, if "volatile." is a computed key, then "volatile.demo"
// is considered computed. However, "volatile" without a trailing dot
// will not make "volatile.demo" computed.
func isComputedKey(key string, computedKeys []string) bool {
	for _, ck := range computedKeys {
		if key == ck || strings.HasSuffix(ck, ".") && strings.HasPrefix(key, ck) {
			return true
		}
	}

	return false
}

// ToConfigMap converts config of type types.Map into map[string]string.
func ToConfigMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return make(map[string]string), nil
	}

	config := make(map[string]string, len(m.Elements()))
	diags := m.ElementsAs(ctx, &config, false)
	return config, diags
}

// ToConfigMapType converts map[string]string into config of type types.Map.
func ToConfigMapType(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	return types.MapValueFrom(ctx, types.StringType, m)
}
