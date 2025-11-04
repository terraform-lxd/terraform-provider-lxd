package common

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

// ImportMetadata defines import ID properties that are used when parsing
// values from an ID.
type ImportMetadata struct {
	ResourceName   string
	RequiredFields []string
	AllowedOptions []string
}

// ParseImportID parses remote name, project name, required fields, and other
// allowed options from an import ID.
//
// Remote is separated using colun "[remote:]". Project and other required
// fields are separated using slash "[project/]rf". If there are multiple
// required fields, first slash becomes mandatory "[project]/rf1/rf2".
// Options are separated using comma "rf[,opt1=value][,opt2=value]".
//
// Expected format:
//
//	[remote:][project/]name[,optKey1=optVal1][,optKeyN=optValN]
//
// Result is a map of key=value pairs:
//
//	map[string]string{
//	  "remote":  "local"
//	  "project": "test"
//	  "name":    "myres"
//	  "image":   "jammy" // option
//	  "optKey2": "value"  // option
//	}
func (m ImportMetadata) ParseImportID(importID string) (map[string]string, diag.Diagnostic) {
	if strings.TrimSpace(importID) == "" {
		return nil, newImportIDError(m, importID, fmt.Errorf("Import ID cannot be empty"))
	}

	// First split by comma to determine mandatory and optional parts.
	parts := strings.Split(importID, ",")

	// Extract fields (including project and remote) from first part.
	result, err := processFields(parts[0], m.RequiredFields)
	if err != nil {
		return nil, newImportIDError(m, importID, err)
	}

	// Extract options.
	if len(parts) > 1 {
		options, err := processOptions(parts[1:], m.AllowedOptions)
		if err != nil {
			return nil, newImportIDError(m, importID, err)
		}

		for k, v := range options {
			result[k] = v
		}
	}

	return result, nil
}

// processFields convert the mandatory part of the import ID into remote,
// project, and any number of provided required fields.
func processFields(id string, requiredFields []string) (map[string]string, error) {
	result := make(map[string]string)

	// Check for remote if import ID contains colon.
	// If colon appears after first slash, it is not a remote (most likely part of the IPv6).
	before, _, _ := strings.Cut(id, "/")
	remote, _, found := strings.Cut(before, ":")
	if found {
		if remote != "" {
			result["remote"] = remote
		}

		id = strings.TrimPrefix(id, remote+":")
	}

	// Split the remaining id into project and required fields.
	parts := strings.Split(id, "/")
	if len(parts) > 1 {
		project := parts[0]
		if project != "" {
			result["project"] = project
		}

		parts = parts[1:]
	}

	// Ensure the length of remaining fields equals the length of
	// required fields.
	if len(parts) != len(requiredFields) {
		return nil, fmt.Errorf("Import ID does not contain all required fields: [%v]", strings.Join(requiredFields, ", "))
	}

	// Extract values for fields into result.
	for i := range parts {
		key := requiredFields[i]
		val := parts[i]

		if val == "" {
			return nil, fmt.Errorf("Import ID requires non-empty value for %q", key)
		}

		result[key] = val
	}

	return result, nil
}

// processOptions convert optional part of the import id and returns it as
// a map. If non-allowed or empty option is extracted, an error is returned.
func processOptions(options []string, allowedOptions []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, o := range options {
		if o == "" {
			continue
		}

		parts := strings.Split(o, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Import ID contains invalid option %q. Options must be in key=value format", o)
		}

		key := parts[0]
		val := parts[1]

		if !utils.ValueInSlice(key, allowedOptions) {
			return nil, fmt.Errorf("Import ID contains unexpected option %q", key)
		}

		result[key] = val
	}

	return result, nil
}

// newImportIDError converts an error into terraform diagnostic.
func newImportIDError(m ImportMetadata, importID string, err error) diag.Diagnostic {
	remote := "[<remote>:]"
	project := "[<project>/]"
	if len(m.RequiredFields) > 1 {
		project = "[<project>]/"
	}

	options := ""
	for _, o := range m.AllowedOptions {
		options += fmt.Sprintf("[,%s=<value>]", o)
	}

	fields := fmt.Sprintf("<%s>", strings.Join(m.RequiredFields, ">/<"))

	return diag.NewErrorDiagnostic(
		fmt.Sprintf("Invalid import ID: %q", importID),
		fmt.Sprintf(
			"%v.\n\nValid import format:\nimport lxd_%s.<resource> %s%s%s%s",
			err, m.ResourceName, remote, project, fields, options,
		),
	)
}
