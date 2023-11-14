package lxd

import (
	"strings"

	"github.com/canonical/lxd/shared"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ConfigType determines how to handle provided configuration when
// supressing diferences or evaluating computed values.
type ConfigType uint8

// Configuration types represent different LXD resources, such as
// instance or network.
const (
	ConfigTypeInstance ConfigType = iota
	ConfigTypeNetwork
	ConfigTypeStoragePool
	ConfigTypeProject
)

// GetComputedKeys returns the list of computed keys based on the provided
// configuration type.
func GetComputedKeys(t ConfigType, d *schema.ResourceData) []string {
	if d == nil {
		return []string{}
	}

	switch t {
	case ConfigTypeNetwork:
		return []string{
			"ipv4.address",
			"ipv4.nat",
			"ipv6.address",
			"ipv6.nat",
		}
	case ConfigTypeStoragePool:
		switch d.Get("driver") {
		case "dir":
			return []string{
				"source",
			}
		case "zfs":
			return []string{
				"source",
				"size",
				"zfs.pool_name",
			}
		case "lvm":
			return []string{
				"source",
				"size",
				"lvm.vg_name",
				"lvm.thinpool_name",
			}
		case "ceph":
		case "cephfs":
		case "cephobject":
		}
	case ConfigTypeProject:
		return []string{
			"features.images",
			"features.profiles",
			"features.storage.volumes",
			"features.storage.buckets",
		}
	}

	return []string{}
}

// SuppressComputedConfigDiff supresses change for a given config entry
// an empty change is detected, if configuration length has changed, or
// if new value is empty but the key is contained within computed keys.
func SuppressComputedConfigDiff(t ConfigType) schema.SchemaDiffSuppressFunc {
	return func(key string, old string, new string, d *schema.ResourceData) bool {
		k := strings.TrimPrefix(key, "config.")

		// Supress config length change.
		if k == "%" {
			return true
		}

		// If user has set the value, never ignore changes.
		if new != "" {
			return false
		}

		// Supress empty changes.
		if old == "" {
			return true
		}

		// Ignore the change if the key is contained within
		// computed keys.
		return ValueInSlice(k, GetComputedKeys(t, d))
	}
}

// HasComputeConfigChanged returns true if resource and new (user) config differ.
func HasComputeConfigChanged(t ConfigType, d *schema.ResourceData, resConfig map[string]string, newConfig map[string]string) bool {
	computedKeys := GetComputedKeys(t, d)

	// Ensure all entries from new config are present in old config.
	for k, new := range newConfig {
		old, _ := resConfig[k]
		if new != old {
			return true
		}
	}

	// Iterate over exiting config and check whether all enteries are
	// either present in old config or computed.
	for k := range resConfig {
		_, ok := newConfig[k]
		if !ok && !shared.StringInSlice(k, computedKeys) {
			return true
		}
	}

	return false
}

// ComputeConfig merges resource (existing) configuration with new (user)
// configuration. Map entries that are contained in computed keys are left
// in the configuration file as they are treated as computed entries.
func ComputeConfig(t ConfigType, d *schema.ResourceData, resConfig map[string]string, newConfig map[string]string) map[string]string {
	config := make(map[string]string)
	computedKeys := GetComputedKeys(t, d)

	// Add new (user) entries to the config.
	for k, v := range newConfig {
		config[k] = v
	}

	// Add computed entries to the config. Computed entries are those
	// that are not contained in new (user) configuration but its key
	// is contained in computedKeys.
	for k, v := range resConfig {
		_, ok := config[k]
		if !ok && shared.StringInSlice(k, computedKeys) {
			config[k] = v
		}
	}

	return config
}
