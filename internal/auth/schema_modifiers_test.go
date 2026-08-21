package auth

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// identitySchema holds the two attributes that the derivation modifiers and the
// replacement condition read.
var identitySchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"auth_method": schema.StringAttribute{Optional: true, Computed: true},
		"type":        schema.StringAttribute{Optional: true, Computed: true},
	},
}

// identityObject builds an object holding the given auth_method and type. A nil
// value is null and tftypes.UnknownValue is unknown.
func identityObject(authMethod any, identityType any) tftypes.Value {
	return tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"auth_method": tftypes.String,
				"type":        tftypes.String,
			},
		},
		map[string]tftypes.Value{
			"auth_method": tftypes.NewValue(tftypes.String, authMethod),
			"type":        tftypes.NewValue(tftypes.String, identityType),
		},
	)
}

// stringValue converts a value of a test table into the framework type. A nil
// value is null and tftypes.UnknownValue is unknown.
func stringValue(value any) types.String {
	switch v := value.(type) {
	case nil:
		return types.StringNull()
	case string:
		return types.StringValue(v)
	}

	return types.StringUnknown()
}

func TestAuthMethodModifier(t *testing.T) {
	var tests = []struct {
		Name             string
		ConfigAuthMethod any
		ConfigType       any
		PlanAuthMethod   any
		PlanType         any
		Expect           types.String
	}{
		{
			Name:             "Configured authentication method is kept",
			ConfigAuthMethod: "tls",
			PlanAuthMethod:   "tls",
			PlanType:         "tls",
			Expect:           types.StringValue("tls"),
		},
		{
			Name:           "Derived from the configured identity type",
			ConfigType:     "bearer",
			PlanAuthMethod: tftypes.UnknownValue,
			PlanType:       "bearer",
			Expect:         types.StringValue("bearer"),
		},
		{
			// A prior state holding tls carries that value into the plan, which
			// a derivation reading the plan would return instead of bearer.
			Name:           "Derived from the configuration and not from the plan",
			ConfigType:     "bearer",
			PlanAuthMethod: "tls",
			PlanType:       "tls",
			Expect:         types.StringValue("bearer"),
		},
		{
			Name:           "Derived from the configured devlxd identity type",
			ConfigType:     "devlxd",
			PlanAuthMethod: tftypes.UnknownValue,
			PlanType:       "devlxd",
			Expect:         types.StringValue("bearer"),
		},
		{
			Name:           "Unknown identity type",
			ConfigType:     tftypes.UnknownValue,
			PlanAuthMethod: tftypes.UnknownValue,
			PlanType:       tftypes.UnknownValue,
			Expect:         types.StringUnknown(),
		},
		{
			// The configuration validator rejects this, the modifier keeps the
			// value it was given.
			Name:           "Neither attribute configured",
			PlanAuthMethod: "bearer",
			PlanType:       "bearer",
			Expect:         types.StringValue("bearer"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				Config:      tfsdk.Config{Raw: identityObject(test.ConfigAuthMethod, test.ConfigType), Schema: identitySchema},
				ConfigValue: stringValue(test.ConfigAuthMethod),
				Plan:        tfsdk.Plan{Raw: identityObject(test.PlanAuthMethod, test.PlanType), Schema: identitySchema},
				PlanValue:   stringValue(test.PlanAuthMethod),
			}

			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			authMethodModifier{}.PlanModifyString(t.Context(), req, resp)

			require.False(t, resp.Diagnostics.HasError())
			assert.Equal(t, test.Expect, resp.PlanValue)
		})
	}
}

func TestIdentityTypeModifier(t *testing.T) {
	var tests = []struct {
		Name             string
		ConfigAuthMethod any
		ConfigType       any
		PlanAuthMethod   any
		PlanType         any
		Expect           types.String
	}{
		{
			Name:       "Configured identity type is kept",
			ConfigType: "tls",
			PlanType:   "tls",
			Expect:     types.StringValue("tls"),
		},
		{
			Name:             "Derived from the configured authentication method",
			ConfigAuthMethod: "bearer",
			PlanAuthMethod:   "bearer",
			PlanType:         tftypes.UnknownValue,
			Expect:           types.StringValue("bearer"),
		},
		{
			// A prior state holding tls carries that value into the plan,
			// which is the drift the derivation exists to report.
			Name:             "Derived from the configuration and not from the plan",
			ConfigAuthMethod: "bearer",
			PlanAuthMethod:   "bearer",
			PlanType:         "tls",
			Expect:           types.StringValue("bearer"),
		},
		{
			// A prior state holding devlxd carries that value into the plan,
			// which is the drift the derivation exists to report.
			Name:             "Derived from the authentication method of a devlxd identity",
			ConfigAuthMethod: "bearer",
			PlanAuthMethod:   "bearer",
			PlanType:         "devlxd",
			Expect:           types.StringValue("bearer"),
		},
		{
			Name:             "Unknown authentication method",
			ConfigAuthMethod: tftypes.UnknownValue,
			PlanAuthMethod:   tftypes.UnknownValue,
			PlanType:         tftypes.UnknownValue,
			Expect:           types.StringUnknown(),
		},
		{
			// The configuration validator rejects this, the modifier keeps the
			// value it was given.
			Name:           "Neither attribute configured",
			PlanAuthMethod: "bearer",
			PlanType:       "bearer",
			Expect:         types.StringValue("bearer"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				Config:      tfsdk.Config{Raw: identityObject(test.ConfigAuthMethod, test.ConfigType), Schema: identitySchema},
				ConfigValue: stringValue(test.ConfigType),
				Plan:        tfsdk.Plan{Raw: identityObject(test.PlanAuthMethod, test.PlanType), Schema: identitySchema},
				PlanValue:   stringValue(test.PlanType),
			}

			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			identityTypeModifier{}.PlanModifyString(t.Context(), req, resp)

			require.False(t, resp.Diagnostics.HasError())
			assert.Equal(t, test.Expect, resp.PlanValue)
		})
	}
}

// TestRequiresReplaceIdentityType covers the cases the framework passes on. It
// calls the condition only when the planned identity type differs from the one
// in the prior state, and never on create or destroy.
func TestRequiresReplaceIdentityType(t *testing.T) {
	var tests = []struct {
		Name            string
		StateAuthMethod any
		StateType       any
		PlanType        any
		Expect          bool
	}{
		{
			Name:            "Identity type changed",
			StateAuthMethod: "tls",
			StateType:       "tls",
			PlanType:        "bearer",
			Expect:          true,
		},
		{
			// State written before the type attribute existed. The identity is
			// a client bearer one and stays that way, so the null identity type
			// is filled in place.
			Name:            "Identity type filled from the authentication method",
			StateAuthMethod: "bearer",
			PlanType:        "bearer",
			Expect:          false,
		},
		{
			Name:            "Identity type filled from the tls authentication method",
			StateAuthMethod: "tls",
			PlanType:        "tls",
			Expect:          false,
		},
		{
			Name:            "Identity type changed to devlxd",
			StateAuthMethod: "bearer",
			StateType:       "bearer",
			PlanType:        "devlxd",
			Expect:          true,
		},
		{
			// Same state, but the configuration asks for another identity
			// type, which the recorded authentication method contradicts.
			Name:            "Identity type changed against the authentication method",
			StateAuthMethod: "bearer",
			PlanType:        "tls",
			Expect:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				State:      tfsdk.State{Raw: identityObject(test.StateAuthMethod, test.StateType), Schema: identitySchema},
				StateValue: stringValue(test.StateType),
				PlanValue:  stringValue(test.PlanType),
			}

			resp := &stringplanmodifier.RequiresReplaceIfFuncResponse{}
			requiresReplaceIdentityType(t.Context(), req, resp)

			require.False(t, resp.Diagnostics.HasError())
			assert.Equal(t, test.Expect, resp.RequiresReplace)
		})
	}
}
