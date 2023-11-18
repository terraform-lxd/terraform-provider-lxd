package utils

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
)

// CheckVersion checks whether the version satisfies the provided version constraints.
func CheckVersion(versionString string, versionConstraint string) (bool, error) {
	ver, err := version.NewVersion(versionString)
	if err != nil {
		return false, fmt.Errorf("Unable to parse version %q: %v", versionString, err)
	}

	constraint, err := version.NewConstraint(versionConstraint)
	if err != nil {
		return false, fmt.Errorf("Unable to parse version constraint %q: %v", versionConstraint, err)
	}

	return constraint.Check(ver), nil
}

// HasAnyPrefix checks whether a value has any of the prefixes.
func HasAnyPrefix(value string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(value, p) {
			return true
		}
	}

	return false
}

// ValueInSlice checks whether a value is present in the given slice.
func ValueInSlice[T comparable](value T, slice []T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}

// SortMapKeys returns map keys sorted alphabetically.
func SortMapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

// ToPrettyJSON converts the given value into JSON string. If value cannot
// be marshaled into JSON, an empty string is returned.
func ToPrettyJSON(v any) string {
	bytes, _ := json.MarshalIndent(v, "", "    ")
	return string(bytes)
}
