package auth

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// authMethodModifier derives the planned auth_method from the configured type.
type authMethodModifier struct{}

func (m authMethodModifier) Description(_ context.Context) string {
	return "Attribute auth_method is derived from attribute type when it is not configured."
}

func (m authMethodModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m authMethodModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	// The identity type is read from the configuration and not from the plan.
	// Plan modifiers of distinct attributes run in Go map order, so the planned
	// identity type is not settled at this point.
	var identityType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &identityType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Neither attribute is configured, which the configuration validator rejects.
	if identityType.IsNull() {
		return
	}

	// Every identity type is also an authentication method, so the value is
	// taken as it is. An unknown identity type makes the authentication method
	// unknown.
	resp.PlanValue = identityType
}

// identityTypeModifier derives the planned type from the configured auth_method.
type identityTypeModifier struct{}

func (m identityTypeModifier) Description(_ context.Context) string {
	return "Attribute type is derived from attribute auth_method when it is not configured."
}

func (m identityTypeModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m identityTypeModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	// See authMethodModifier for why this reads the configuration.
	var authMethod types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("auth_method"), &authMethod)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Neither attribute is configured, which the configuration validator rejects.
	if authMethod.IsNull() {
		return
	}

	// Every authentication method is also an identity type, so the value is
	// taken as it is. An unknown authentication method makes the identity type
	// unknown.
	resp.PlanValue = authMethod
}

// requiresReplaceIdentityTypeDescription describes requiresReplaceIdentityType
// in the schema.
const requiresReplaceIdentityTypeDescription = "If the identity type changes, Terraform will destroy and recreate the identity."

// requiresReplaceIdentityType requires replacement when the identity type
// changes. State written before the type attribute existed holds a null
// identity type and records it in auth_method, which the planned identity type
// is compared against instead.
func requiresReplaceIdentityType(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if !req.StateValue.IsNull() {
		resp.RequiresReplace = true
		return
	}

	var authMethod types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("auth_method"), &authMethod)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.RequiresReplace = !req.PlanValue.Equal(authMethod)
}
