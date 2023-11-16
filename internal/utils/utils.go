package utils

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-version"
)

// // ToStringTypeMap convert types.Map into map[string]types.String.
// func ToStringTypeMap(ctx context.Context, m types.Map) (map[string]types.String, diag.Diagnostics) {
// 	config := make(map[string]types.String, len(m.Elements()))
// 	diags := m.ElementsAs(ctx, &config, false)
// 	return config, diags
// }

// // ToStringMap convert types.Map into map[string]string.
// func ToStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
// 	raw, diags := ToStringTypeMap(ctx, m)
// 	if diags.HasError() {
// 		return nil, diags
// 	}

// 	config := make(map[string]string, len(raw))
// 	for k, v := range raw {
// 		if v.IsNull() || v.IsUnknown() {
// 			config[k] = ""
// 		} else {
// 			config[k] = v.ValueString()
// 		}
// 	}

// 	return config, diags
// }

// // FromStringMap convert map[string]string into types.Map.
// func FromStringMap(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
// 	return types.MapValueFrom(ctx, types.StringType, m)
// }

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

// ValueInSlice checks whether a value is present in the given slice.
func ValueInSlice[T comparable](value T, slice []T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}

// ToPrettyJSON converts the given value into JSON string. If value cannot
// be marshaled into JSON, an empty string is returned.
func ToPrettyJSON(v any) string {
	bytes, _ := json.MarshalIndent(v, "", "    ")
	return string(bytes)
}
