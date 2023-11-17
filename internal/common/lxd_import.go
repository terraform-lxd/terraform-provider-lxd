package common

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// SplitImportID splits import ID into remote, project and name.
func SplitImportID(id string, resType string) (remote string, project string, name string, err diag.Diagnostic) {
	errSummary := fmt.Sprintf("Invalid import format: %q", id)
	errDetails := fmt.Sprintf("Valid import:\nimport lxd_%[1]s.<resource_name> [<remote>:][<project>/]<%[1]s_name>", resType)

	// Split id into [remote:]<id>
	split := strings.Split(id, ":")
	if len(split) > 2 {
		err = diag.NewErrorDiagnostic(errSummary, errDetails)
	} else if len(split) == 2 {
		remote = split[0]
		id = split[1]
	}

	// Split id into [project/]<name>
	split = strings.Split(id, "/")
	if len(split) > 2 {
		err = diag.NewErrorDiagnostic(errSummary, errDetails)
	} else if len(split) == 2 {
		project = split[0]
		name = split[1]
	} else {
		name = id
	}

	// Verify name of the LXD resource is not empty.
	if name == "" {
		err = diag.NewErrorDiagnostic(errSummary, errDetails)
	}

	return
}
