package errors

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/lxc/incus/shared/api"
)

// IsNotFoundError checks whether the given error is of type NotFound.
func IsNotFoundError(err error) bool {
	return api.StatusErrorCheck(err, http.StatusNotFound)
}

// NewInstanceServerError converts an error into diagnostic indicating
// that provider failed to retrieve Incus instance server client.
func NewInstanceServerError(err error) diag.Diagnostic {
	return diag.NewErrorDiagnostic("Failed to retrieve Incus InstanceServer", err.Error())
}

// NewImageServerError converts an error into diagnostic indicating
// that provider failed to retrieve Incus image server client.
func NewImageServerError(err error) diag.Diagnostic {
	return diag.NewErrorDiagnostic("Failed to retrieve Incus ImageServer", err.Error())
}

// NewProviderDataTypeError returns a diagnostic error indicating that
// a resource has received provider data of unexpected type.
func NewProviderDataTypeError(value any) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Unexpected ProviderData type",
		fmt.Sprintf(
			"Expected *provider_config.IncusProviderConfig, got %T. "+
				"Please report this issue to the provider maintainers.",
			value,
		),
	)
}
