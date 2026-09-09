package common

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"golang.org/x/sync/singleflight"
)

// MetadataConfigKeySource describes a metadata configuration group and an optional prefix to
// apply to every key in that group.
type MetadataConfigKeySource struct {
	Keys   api.MetadataConfigurationConfigKeys
	Prefix string
}

// MetadataConfigKeys matches concrete configuration keys to their metadata.
type MetadataConfigKeys struct {
	// Map of exact keys.
	exact map[string]api.MetadataConfigurationConfigKey

	// List of keys with dynamic segments, such as wildcards or named placeholders.
	patterns []metadataConfigKeyPattern
}

type metadataConfigKeyPattern struct {
	regexp *regexp.Regexp
	config api.MetadataConfigurationConfigKey
}

var (
	metadataCache     = make(map[string]*api.MetadataConfiguration)
	metadataCacheLock sync.RWMutex
	metadataGroup     singleflight.Group
)

// NewMetadataConfigKeys builds a complete configuration key set from metadata groups.
func NewMetadataConfigKeys(sources ...MetadataConfigKeySource) MetadataConfigKeys {
	configKeys := MetadataConfigKeys{
		exact: make(map[string]api.MetadataConfigurationConfigKey),
	}

	for _, source := range sources {
		for _, keys := range source.Keys.Keys {
			for key, config := range keys {
				key = source.Prefix + key
				pattern := compileMetadataKeyPattern(key)
				if pattern != nil {
					configKeys.patterns = append(configKeys.patterns, metadataConfigKeyPattern{regexp: pattern, config: config})
					continue
				}

				configKeys.exact[key] = config
			}
		}
	}

	return configKeys
}

// NewLocalMetadataConfigKeys builds local-scope metadata for servers that do not expose metadata.
func NewLocalMetadataConfigKeys(localKeys []string) MetadataConfigKeys {
	keys := make(map[string]api.MetadataConfigurationConfigKey, len(localKeys))
	for _, key := range localKeys {
		keys[key] = api.MetadataConfigurationConfigKey{Scope: "local"}
	}

	return NewMetadataConfigKeys(MetadataConfigKeySource{
		Keys: api.MetadataConfigurationConfigKeys{Keys: []map[string]api.MetadataConfigurationConfigKey{keys}},
	})
}

// Lookup returns metadata for a concrete configuration key.
func (k MetadataConfigKeys) Lookup(key string) (api.MetadataConfigurationConfigKey, bool) {
	config, ok := k.exact[key]
	if ok {
		return config, true
	}

	for _, pattern := range k.patterns {
		if pattern.regexp.MatchString(key) {
			return pattern.config, true
		}
	}

	return api.MetadataConfigurationConfigKey{}, false
}

// IsLocal reports whether a concrete key has local scope.
func (k MetadataConfigKeys) IsLocal(key string) bool {
	config, ok := k.Lookup(key)
	return ok && config.Scope == "local"
}

// compileMetadataKeyPattern compiles a metadata key that contains placeholders into a regular
// expression and returns nil for a literal key. The placeholder spellings are a convention of
// the LXD metadata rather than a rule it defines. A trailing "*" matches the rest of the key,
// a bracketed name such as "{name}" or "<name>" may contain dots, and an uppercase name such
// as "NAME" matches a single segment.
func compileMetadataKeyPattern(key string) *regexp.Regexp {
	parts := strings.Split(key, ".")
	dynamic := false

	for i, part := range parts {
		switch {
		case part == "*" && i == len(parts)-1:
			// A trailing wildcard matches the remaining suffix.
			parts[i] = `.+`
			dynamic = true
		case isMetadataBracketedPlaceholder(part):
			// Bracketed placeholders may contain dots, for example a project name "my.proj".
			parts[i] = `.+`
			dynamic = true
		case isMetadataUppercasePlaceholder(part):
			// Uppercase placeholders match one segment.
			parts[i] = `[^.]+`
			dynamic = true
		default:
			// Literal segments are quoted to match exactly.
			parts[i] = regexp.QuoteMeta(part)
		}
	}

	if !dynamic {
		return nil
	}

	return regexp.MustCompile(`^` + strings.Join(parts, `\.`) + `$`)
}

// isMetadataBracketedPlaceholder reports whether part is spelled like {name} or <name>.
func isMetadataBracketedPlaceholder(part string) bool {
	if len(part) < 3 {
		return false
	}

	isAngle := part[0] == '<' && part[len(part)-1] == '>'
	isBrace := part[0] == '{' && part[len(part)-1] == '}'

	return isAngle || isBrace
}

// isMetadataUppercasePlaceholder reports whether part is an uppercase identifier such as NAME
// or NETWORK_NAME. A segment that starts with a digit, such as 1GB, is a literal.
func isMetadataUppercasePlaceholder(part string) bool {
	if part == "" || part[0] < 'A' || part[0] > 'Z' {
		return false
	}

	for _, char := range part {
		if char != '_' && (char < 'A' || char > 'Z') {
			return false
		}
	}

	return true
}

func ServerMetadataConfiguration(name string, server lxd.InstanceServer) (*api.MetadataConfiguration, error) {
	metadataCacheLock.RLock()
	meta, ok := metadataCache[name]
	metadataCacheLock.RUnlock()
	if ok {
		return meta, nil
	}

	value, err, _ := metadataGroup.Do(name, func() (any, error) {
		metadataCacheLock.RLock()
		meta, ok := metadataCache[name]
		metadataCacheLock.RUnlock()
		if ok {
			return meta, nil
		}

		meta, err := server.GetMetadataConfiguration()
		if err != nil {
			return nil, err
		}

		metadataCacheLock.Lock()
		metadataCache[name] = meta
		metadataCacheLock.Unlock()

		return meta, nil
	})
	if err != nil {
		return nil, err
	}

	meta, ok = value.(*api.MetadataConfiguration)
	if !ok {
		return nil, fmt.Errorf("Unexpected metadata type %T", value)
	}

	return meta, nil
}
