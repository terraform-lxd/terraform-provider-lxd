package common

import (
	"context"
	"testing"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memberOverridesValue builds a member_overrides map value from the given per-member configs.
func memberOverridesValue(t *testing.T, overrides map[string]map[string]attr.Value) types.Map {
	t.Helper()

	elements := make(map[string]attr.Value, len(overrides))
	for memberName, config := range overrides {
		configValue, diags := types.MapValue(types.StringType, config)
		require.False(t, diags.HasError(), diags)

		objectValue, diags := types.ObjectValue(memberObjectType.AttrTypes, map[string]attr.Value{
			"config": configValue,
		})
		require.False(t, diags.HasError(), diags)

		elements[memberName] = objectValue
	}

	value, diags := types.MapValue(memberObjectType, elements)
	require.False(t, diags.HasError(), diags)

	return value
}

func TestMemberOverridesHaveUnknownConfig(t *testing.T) {
	ctx := context.Background()

	assert.False(t, MemberOverridesHaveUnknownConfig(ctx, types.MapNull(memberObjectType)))
	assert.False(t, MemberOverridesHaveUnknownConfig(ctx, types.MapUnknown(memberObjectType)))

	known := memberOverridesValue(t, map[string]map[string]attr.Value{
		"member-1": {"size": types.StringValue("10GiB")},
	})
	assert.False(t, MemberOverridesHaveUnknownConfig(ctx, known))

	unknown := memberOverridesValue(t, map[string]map[string]attr.Value{
		"member-1": {"size": types.StringUnknown()},
	})
	assert.True(t, MemberOverridesHaveUnknownConfig(ctx, unknown))
}

func TestToMembersMapType(t *testing.T) {
	ctx := context.Background()

	// A resource on a non-clustered server has no members, which must produce
	// an empty map instead of a null one, as the attribute is computed.
	empty, diags := ToMembersMapType(ctx, nil)
	require.False(t, diags.HasError(), diags)
	assert.False(t, empty.IsNull())
	assert.Empty(t, empty.Elements())

	members, diags := ToMembersMapType(ctx, map[string]map[string]string{
		"member-1": {"source": "/dev/sdb"},
		"member-2": {},
	})
	require.False(t, diags.HasError(), diags)

	expected := memberOverridesValue(t, map[string]map[string]attr.Value{
		"member-1": {"source": types.StringValue("/dev/sdb")},
		"member-2": {},
	})
	assert.Equal(t, expected, members)
}

func TestResolveMemberConfigs(t *testing.T) {
	ctx := context.Background()
	memberNames := []string{"member-1", "member-2"}
	configKeys := NewLocalMetadataConfigKeys([]string{"size", "source"})

	config := map[string]string{
		"source":         "/dev/sda",
		"size":           "10GiB",
		"zfs.clone_copy": "true",
	}

	overrides := memberOverridesValue(t, map[string]map[string]attr.Value{
		"member-1": {"source": types.StringValue("/dev/sdb")},
	})

	globalConfig, memberConfigs, err := ResolveMemberConfigs(ctx, `Storage pool "p1" (zfs)`, config, overrides, memberNames, configKeys)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"zfs.clone_copy": "true"}, globalConfig)
	assert.Equal(t, map[string]map[string]string{
		"member-1": {"source": "/dev/sdb", "size": "10GiB"},
		"member-2": {"source": "/dev/sda", "size": "10GiB"},
	}, memberConfigs)
}

func TestResolveMemberConfigs_MetadataPattern(t *testing.T) {
	keys := api.MetadataConfigurationConfigKeys{
		Keys: []map[string]api.MetadataConfigurationConfigKey{
			{"storage.project.{name}.backups_volume": {Scope: "local"}},
		},
	}
	configKeys := NewMetadataConfigKeys(MetadataConfigKeySource{Keys: keys})
	config := map[string]string{
		"storage.project.default.backups_volume": "pool/backups",
		"core.https_address":                     "127.0.0.1:8443",
	}

	globalConfig, memberConfigs, err := ResolveMemberConfigs(
		context.Background(),
		"LXD server",
		config,
		types.MapNull(memberObjectType),
		[]string{"member-1"},
		configKeys,
	)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"core.https_address": "127.0.0.1:8443"}, globalConfig)
	assert.Equal(t, map[string]map[string]string{
		"member-1": {"storage.project.default.backups_volume": "pool/backups"},
	}, memberConfigs)
}

func TestResolveMemberConfigs_MetadataPatternOverride(t *testing.T) {
	keys := api.MetadataConfigurationConfigKeys{
		Keys: []map[string]api.MetadataConfigurationConfigKey{
			{"storage.project.{name}.backups_volume": {Scope: "local"}},
		},
	}
	configKeys := NewMetadataConfigKeys(MetadataConfigKeySource{Keys: keys})
	overrides := memberOverridesValue(t, map[string]map[string]attr.Value{
		"member-1": {"storage.project.default.backups_volume": types.StringValue("pool/member-1")},
	})

	globalConfig, memberConfigs, err := ResolveMemberConfigs(
		context.Background(),
		"LXD server",
		map[string]string{"storage.project.default.backups_volume": "pool/backups"},
		overrides,
		[]string{"member-1", "member-2"},
		configKeys,
	)
	require.NoError(t, err)

	assert.Empty(t, globalConfig)
	assert.Equal(t, map[string]map[string]string{
		"member-1": {"storage.project.default.backups_volume": "pool/member-1"},
		"member-2": {"storage.project.default.backups_volume": "pool/backups"},
	}, memberConfigs)
}

func TestResolveMemberConfigs_UnknownMember(t *testing.T) {
	overrides := memberOverridesValue(t, map[string]map[string]attr.Value{
		"member-9": {"source": types.StringValue("/dev/sdb")},
	})

	configKeys := NewLocalMetadataConfigKeys([]string{"source"})
	_, _, err := ResolveMemberConfigs(context.Background(), `Storage pool "p1" (zfs)`, nil, overrides, []string{"member-1"}, configKeys)
	assert.ErrorContains(t, err, `contains member-specific config override for a non-existent cluster member "member-9"`)
}

func TestResolveMemberConfigs_GlobalKeyInOverride(t *testing.T) {
	overrides := memberOverridesValue(t, map[string]map[string]attr.Value{
		"member-1": {"zfs.clone_copy": types.StringValue("true")},
	})

	configKeys := NewLocalMetadataConfigKeys([]string{"source"})
	_, _, err := ResolveMemberConfigs(context.Background(), `Storage pool "p1" (zfs)`, nil, overrides, []string{"member-1"}, configKeys)
	assert.ErrorContains(t, err, `Invalid config key "zfs.clone_copy" for member "member-1"`)
}
