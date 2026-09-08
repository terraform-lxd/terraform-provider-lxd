package common

import (
	"context"
	"fmt"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
)

// MemberModel represents a per-member configuration of a clustered resource.
type MemberModel struct {
	Config types.Map `tfsdk:"config"`
}

// memberObjectType is the type of a single entry in the "member_overrides"
// and "members" attributes.
var memberObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"config": types.MapType{ElemType: types.StringType},
	},
}

// MemberOverridesHaveUnknownConfig reports whether any entry of an otherwise
// known member_overrides map contains a config value that is only known
// after apply. The outer map can be fully known (all member keys present)
// while a nested override's "config" attribute, or a value within it,
// remains unknown, which ToConfigMap cannot convert.
func MemberOverridesHaveUnknownConfig(ctx context.Context, memberOverrides types.Map) bool {
	if memberOverrides.IsNull() || memberOverrides.IsUnknown() {
		return false
	}

	overrides := map[string]MemberModel{}
	diags := memberOverrides.ElementsAs(ctx, &overrides, true)
	if diags.HasError() {
		return false
	}

	for _, override := range overrides {
		if ConfigHasUnknownValue(override.Config) {
			return true
		}
	}

	return false
}

// ToMembersMapType converts per-member configuration into the value of the
// computed "members" attribute.
func ToMembersMapType(ctx context.Context, memberConfigs map[string]map[string]string) (types.Map, diag.Diagnostics) {
	members := make(map[string]MemberModel, len(memberConfigs))

	for memberName, memberConfig := range memberConfigs {
		configValue, diags := types.MapValueFrom(ctx, types.StringType, ToNullableConfig(memberConfig))
		if diags.HasError() {
			return types.MapNull(memberObjectType), diags
		}

		members[memberName] = MemberModel{Config: configValue}
	}

	return types.MapValueFrom(ctx, memberObjectType, members)
}

// ResolveMemberConfigs splits config into cluster-wide and member-specific configuration.
// Local keys found in config become the default for every cluster member, and each member's
// override is merged on top of that default. The returned global config contains only the
// keys that do not have local scope. Entity describes the resource in error messages,
// for example `Storage pool "p1" (zfs)`.
func ResolveMemberConfigs(ctx context.Context, entity string, config map[string]string, memberOverrides types.Map, memberNames []string, configKeys MetadataConfigKeys) (globalConfig map[string]string, memberConfigs map[string]map[string]string, err error) {
	// Separate global and member-specific configuration.
	globalConfig = make(map[string]string, len(config))
	defaultMemberConfig := make(map[string]string)

	for k, v := range config {
		if configKeys.IsLocal(k) {
			defaultMemberConfig[k] = v
			continue
		}

		globalConfig[k] = v
	}

	// Set member-specific config from global config to all members by default.
	memberConfigs = make(map[string]map[string]string, len(memberNames))
	for _, memberName := range memberNames {
		memberConfigs[memberName] = maps.Clone(defaultMemberConfig)
	}

	// Extract member-specific config overrides.
	overrides := map[string]MemberModel{}
	err = errors.FromDiagnostics(memberOverrides.ElementsAs(ctx, &overrides, true))
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to extract member-specific config overrides: %v", err)
	}

	for memberName, override := range overrides {
		memberConfig, ok := memberConfigs[memberName]
		if !ok {
			return nil, nil, fmt.Errorf("%s contains member-specific config override for a non-existent cluster member %q!", entity, memberName)
		}

		// Parse and apply member-specific override.
		configMap, diags := ToConfigMap(ctx, override.Config)
		err := errors.FromDiagnostics(diags)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable to convert member-specific config override to map: %v", err)
		}

		maps.Copy(memberConfig, configMap)

		// Ensure member-specific config does not contain global keys.
		for k := range memberConfig {
			if !configKeys.IsLocal(k) {
				return nil, nil, fmt.Errorf("%s: Invalid config key %q for member %q: Only member-specific keys are allowed in per-member configuration", entity, k, memberName)
			}
		}

		// Store resolved config.
		memberConfigs[memberName] = memberConfig
	}

	return globalConfig, memberConfigs, nil
}
