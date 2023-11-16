package errors

import (
	"fmt"
	"net/http"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// IsNotFoundError checks whether the given error is of type NotFound.
func IsNotFoundError(err error) bool {
	return api.StatusErrorCheck(err, http.StatusNotFound)
}

// NewInstanceServerError converts an error into diagnostic indicating
// that provider failed to retrieve LXD instance server client.
func NewInstanceServerError(err error) diag.Diagnostic {
	return diag.NewErrorDiagnostic("Failed to retrieve LXD InstanceServer", err.Error())
}

// NewProviderDataTypeError returns a diagnostic error indicating that
// a resource has received provider data of unexpected type.
func NewProviderDataTypeError(value any) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Unexpected ProviderData type",
		fmt.Sprintf(
			"Expected *provider_config.LxdProviderConfig, got %T. "+
				"Please report this issue to the provider maintainers.",
			value,
		),
	)
}
