package auth

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// identityTypeValidator ensures exactly one of the attributes auth_method and
// type is configured. A configured auth_method also produces a warning pointing at type.
type identityTypeValidator struct{}

func (v identityTypeValidator) Description(_ context.Context) string {
	return `Exactly one of the attributes "auth_method" and "type" must be configured.`
}

func (v identityTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v identityTypeValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	resp.Diagnostics.Append(v.validate(ctx, req.Config)...)
}

func (v identityTypeValidator) ValidateDataSource(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	resp.Diagnostics.Append(v.validate(ctx, req.Config)...)
}

func (v identityTypeValidator) validate(ctx context.Context, config tfsdk.Config) diag.Diagnostics {
	var diags diag.Diagnostics
	var authMethod types.String
	var identityType types.String

	diags.Append(config.GetAttribute(ctx, path.Root("auth_method"), &authMethod)...)
	diags.Append(config.GetAttribute(ctx, path.Root("type"), &identityType)...)
	if diags.HasError() {
		return diags
	}

	switch {
	case authMethod.IsNull() && identityType.IsNull():
		diags.AddAttributeError(
			path.Root("type"),
			"Invalid attribute combination",
			`Attribute "type" must be set.`,
		)
	case !authMethod.IsNull() && !identityType.IsNull():
		diags.AddAttributeError(
			path.Root("auth_method"),
			"Invalid attribute combination",
			`Attributes "auth_method" and "type" cannot both be set. Use "type".`,
		)
	case !authMethod.IsNull():
		diags.AddAttributeWarning(
			path.Root("auth_method"),
			`Attribute "auth_method" will become read only`,
			`Use "type" instead. "auth_method" is accepted for configurations written before "type" existed, and will become a read only attribute in a future release of the provider.`,
		)
	}

	return diags
}

// identityCertificateValidator rejects a configured certificate for non-tls identities.
type identityCertificateValidator struct{}

func (v identityCertificateValidator) Description(_ context.Context) string {
	return `Attribute "tls_certificate" can only be set for identities of type "tls".`
}

func (v identityCertificateValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v identityCertificateValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var authMethod types.String
	var identityType types.String
	var certificate types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("auth_method"), &authMethod)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &identityType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("tls_certificate"), &certificate)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A certificate that is unknown until apply is checked again on create and update.
	if certificate.IsNull() || certificate.IsUnknown() {
		return
	}

	// See identityTypeModifier for why auth_method stands in for type.
	if identityType.IsNull() {
		identityType = authMethod
	}

	// An identity type that is unknown until apply is checked again on create and update.
	if identityType.IsNull() || identityType.IsUnknown() || identityType.ValueString() == "tls" {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("tls_certificate"),
		"Invalid attribute combination",
		fmt.Sprintf("Certificate must not be set for identities of type %q", identityType.ValueString()),
	)
}
