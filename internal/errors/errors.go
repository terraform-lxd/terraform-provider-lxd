package errors

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// IsNotFoundError checks whether the given error is of type NotFound.
func IsNotFoundError(err error) bool {
	return api.StatusErrorCheck(err, http.StatusNotFound)
}

// IsConflictError checks whether the given error is of type Conflict.
func IsConflictError(err error) bool {
	return api.StatusErrorCheck(err, http.StatusConflict)
}

// NewInstanceServerError converts an error into diagnostic indicating
// that provider failed to retrieve LXD instance server client.
func NewInstanceServerError(err error) diag.Diagnostic {
	return diag.NewErrorDiagnostic("Failed to retrieve LXD InstanceServer", err.Error())
}

// NewImageServerError converts an error into diagnostic indicating
// that provider failed to retrieve LXD image server client.
func NewImageServerError(err error) diag.Diagnostic {
	return diag.NewErrorDiagnostic("Failed to retrieve LXD ImageServer", err.Error())
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

// FromDiagnostics converts a [diag.Diagnostics] object into an [error].
// If the Diagnostics object does not contain any errors, nil is returned.
func FromDiagnostics(diags diag.Diagnostics) error {
	if diags == nil || !diags.HasError() {
		return nil
	}

	toErrorMsg := func(diag diag.Diagnostic) string {
		msg := diag.Summary()

		detail := diag.Detail()
		if detail != "" {
			msg += ": " + detail
		}

		return msg
	}

	errDiags := diags.Errors()

	if len(errDiags) == 1 {
		msg := toErrorMsg(errDiags[0])
		return errors.New(msg)
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "%d errors occurred:", len(errDiags))
	for _, e := range errDiags {
		msg.WriteString("\n- ")
		msg.WriteString(toErrorMsg(e))
	}

	return errors.New(msg.String())
}
