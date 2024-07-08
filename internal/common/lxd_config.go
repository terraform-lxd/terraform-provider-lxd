package common

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ToConfigMap converts config of type types.Map into map[string]string.
func ToConfigMap(ctx context.Context, configMap types.Map) (map[string]string, diag.Diagnostics) {
	if configMap.IsNull() || configMap.IsUnknown() {
		return make(map[string]string), nil
	}

	// Convert to an intermediate nullable type.
	tfConfig := make(map[string]*string, len(configMap.Elements()))
	diags := configMap.ElementsAs(ctx, &tfConfig, false)
	if diags != nil {
		return nil, diags
	}

	// Then convert to our native type.
	config := make(map[string]string, len(tfConfig))
	for k, v := range tfConfig {
		if v == nil {
			continue
		}

		config[k] = *v
	}

	return config, nil
}

// ToConfigMapType converts map[string]string into config of type types.Map.
func ToConfigMapType(ctx context.Context, config map[string]*string, modelConfig types.Map) (types.Map, diag.Diagnostics) {
	// Add any missing nil values.
	nullConfig := map[string]*string{}
	if !modelConfig.IsNull() && !modelConfig.IsUnknown() {
		_ = modelConfig.ElementsAs(context.Background(), &nullConfig, false)
	}

	for k, v := range nullConfig {
		if v != nil {
			continue
		}

		_, ok := config[k]
		if !ok {
			config[k] = nil
		}
	}

	return types.MapValueFrom(ctx, types.StringType, config)
}

// ToNullableConfig converts map[string]string to map[string]*string.
func ToNullableConfig(config map[string]string) map[string]*string {
	nullConfig := make(map[string]*string, len(config))

	for k := range config {
		// Copy the value.
		v := string(config[k])
		nullConfig[k] = &v
	}

	return nullConfig
}

// MergeConfig merges resource (existing) configuration with user defined
// configuration. Non-empty resource config entries that are contained in
// the provided computed keys are inserted in the user config.
func MergeConfig(resConfig map[string]string, usrConfig map[string]string, computedKeys []string) map[string]string {
	config := make(map[string]string)

	// Add user defined non-empty entries to the config. Empty values
	// in LXD configuration are considered null (unset).
	for k, v := range usrConfig {
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

// StripConfig removes any computed keys from the user-defined configuration
// file in order to be able to produce a consistent Terraform plan. If there
// is a non-computed-key entry, it will be retained in the configuration and
// will trigger an error.
func StripConfig(resConfig map[string]string, modelConfig types.Map, computedKeys []string) map[string]*string {
	// Handle nulls in modelConfig.
	usrConfig := map[string]*string{}
	if !modelConfig.IsNull() && !modelConfig.IsUnknown() {
		_ = modelConfig.ElementsAs(context.Background(), &usrConfig, false)
	}

	// Populate empty values from user config, so they do not "disappear"
	// from the state.
	config := make(map[string]*string)

	for k, v := range usrConfig {
		if v == nil {
			config[k] = nil
		}
	}

	// Apply entries to the config that are not empty (unset), are not
	// computed, or are present in the user configuration file. The last
	// one ensures that the correct change is shown in the terraform plan.
	for k, v := range resConfig {
		if v == "" {
			continue
		}

		_, ok := usrConfig[k]
		if ok || !isComputedKey(k, computedKeys) {
			if usrConfig[k] == nil && isComputedKey(k, computedKeys) {
				// Keep as null.
				config[k] = nil
			} else {
				// Copy the value.
				v := string(resConfig[k])
				config[k] = &v
			}
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
