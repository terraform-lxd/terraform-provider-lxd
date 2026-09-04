package auth

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// identityTypeValidator ensures exactly one of the attributes auth_method and
// type is configured. When warnOnAuthMethod is set, a configured auth_method
// also produces a warning pointing at type.
type identityTypeValidator struct {
	warnOnAuthMethod bool
}

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
	case !authMethod.IsNull() && v.warnOnAuthMethod:
		diags.AddAttributeWarning(
			path.Root("auth_method"),
			`Attribute "auth_method" will become read only`,
			`Use "type" instead. "auth_method" is accepted for configurations written before "type" existed, and will become a read only attribute in a future release of the provider.`,
		)
	}

	return diags
}
