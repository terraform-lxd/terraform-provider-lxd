package common

import (
	"testing"

	"github.com/canonical/lxd/shared/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileMetadataKeyPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     string
		match   bool
	}{
		{
			name:    "Uppercase",
			pattern: "bgp.peers.NAME.address",
			key:     "bgp.peers.router1.address",
			match:   true,
		},
		{
			name:    "UppercaseExtraSegment",
			pattern: "bgp.peers.NAME.address",
			key:     "bgp.peers.site.router1.address",
			match:   false,
		},
		{
			name:    "UppercaseUnderscore",
			pattern: "limits.networks.uplink_ips.ipv4.NETWORK_NAME",
			key:     "limits.networks.uplink_ips.ipv4.uplink1",
			match:   true,
		},
		{
			name:    "Angle",
			pattern: "volatile.<name>.hwaddr",
			key:     "volatile.eth0.hwaddr",
			match:   true,
		},
		{
			name:    "AngleDottedSegment",
			pattern: "volatile.<name>.hwaddr",
			key:     "volatile.eth0.1.hwaddr",
			match:   true,
		},
		{
			name:    "Brace",
			pattern: "storage.project.{name}.backups_volume",
			key:     "storage.project.default.backups_volume",
			match:   true,
		},
		{
			name:    "BraceDottedSegment",
			pattern: "storage.project.{name}.backups_volume",
			key:     "storage.project.my.proj.backups_volume",
			match:   true,
		},
		{
			name:    "EmptyPlaceholder",
			pattern: "storage.project.{name}.backups_volume",
			key:     "storage.project..backups_volume",
			match:   false,
		},
		{
			name:    "Wildcard",
			pattern: "user.*",
			key:     "user.owner",
			match:   true,
		},
		{
			name:    "WildcardDottedSuffix",
			pattern: "linux.sysctl.*",
			key:     "linux.sysctl.net.ipv4.ip_forward",
			match:   true,
		},
		{
			name:    "EmptyWildcard",
			pattern: "user.*",
			key:     "user.",
			match:   false,
		},
		{
			name:    "MissingSegment",
			pattern: "tunnel.NAME.local",
			key:     "tunnel.local",
			match:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := compileMetadataKeyPattern(tt.pattern)
			require.NotNil(t, pattern)
			assert.Equal(t, tt.match, pattern.MatchString(tt.key))
		})
	}
}

func TestCompileMetadataKeyPattern_NumberIsLiteral(t *testing.T) {
	assert.Nil(t, compileMetadataKeyPattern("limits.example.123NAME"))
	assert.Nil(t, compileMetadataKeyPattern("limits.example.NAME123"))
}

func TestMetadataConfigKeys(t *testing.T) {
	keys := api.MetadataConfigurationConfigKeys{
		Keys: []map[string]api.MetadataConfigurationConfigKey{
			{"core.https_address": {Scope: "global"}},
			{"limits.hugepages.1GB": {Scope: "global"}},
			{"storage.project.{name}.backups_volume": {Scope: "local"}},
			{"user.*": {Scope: "global"}},
		},
	}

	configKeys := NewMetadataConfigKeys(MetadataConfigKeySource{Keys: keys})

	config, ok := configKeys.Lookup("core.https_address")
	require.True(t, ok)
	assert.Equal(t, "global", config.Scope)

	_, ok = configKeys.Lookup("limits.hugepages.1GB")
	assert.True(t, ok)

	_, ok = configKeys.Lookup("limits.hugepages.10GB")
	assert.False(t, ok)

	assert.True(t, configKeys.IsLocal("storage.project.default.backups_volume"))
	assert.True(t, configKeys.IsLocal("storage.project.my.proj.backups_volume"))

	_, ok = configKeys.Lookup("user.owner.team")
	assert.True(t, ok)

	_, ok = configKeys.Lookup("unknown.key")
	assert.False(t, ok)
}

func TestMetadataConfigKeysPrefix(t *testing.T) {
	keys := api.MetadataConfigurationConfigKeys{
		Keys: []map[string]api.MetadataConfigurationConfigKey{
			{"size": {Scope: "local"}},
		},
	}

	configKeys := NewMetadataConfigKeys(MetadataConfigKeySource{Keys: keys, Prefix: "volume."})

	_, ok := configKeys.Lookup("volume.size")
	assert.True(t, ok)
	assert.True(t, configKeys.IsLocal("volume.size"))
	_, ok = configKeys.Lookup("size")
	assert.False(t, ok)
}

func TestMetadataConfigKeys_ExactKeyWins(t *testing.T) {
	keys := api.MetadataConfigurationConfigKeys{
		Keys: []map[string]api.MetadataConfigurationConfigKey{
			{"foo.NAME.bar": {Scope: "global"}},
			{"foo.x.bar": {Scope: "local"}},
		},
	}

	configKeys := NewMetadataConfigKeys(MetadataConfigKeySource{Keys: keys})

	assert.True(t, configKeys.IsLocal("foo.x.bar"))
	assert.False(t, configKeys.IsLocal("foo.y.bar"))
}

func TestMetadataConfigKeys_EmptyScope(t *testing.T) {
	keys := api.MetadataConfigurationConfigKeys{
		Keys: []map[string]api.MetadataConfigurationConfigKey{
			{"ipv4.address": {}},
			{"tunnel.NAME.protocol": {}},
		},
	}

	configKeys := NewMetadataConfigKeys(MetadataConfigKeySource{Keys: keys})

	_, ok := configKeys.Lookup("ipv4.address")
	assert.True(t, ok)
	assert.False(t, configKeys.IsLocal("ipv4.address"))

	_, ok = configKeys.Lookup("tunnel.gre1.protocol")
	assert.True(t, ok)
	assert.False(t, configKeys.IsLocal("tunnel.gre1.protocol"))
}

func TestLocalMetadataConfigKeys(t *testing.T) {
	configKeys := NewLocalMetadataConfigKeys([]string{"source"})

	assert.True(t, configKeys.IsLocal("source"))
	assert.False(t, configKeys.IsLocal("unknown.key"))
}
