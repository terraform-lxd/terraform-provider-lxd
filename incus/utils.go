package incus

import (
	"fmt"

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

// ValueInSlice checks whether a value is present in the given slice.
func ValueInSlice[T comparable](value T, slice []T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}
