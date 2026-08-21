package auth

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityTypeValidator(t *testing.T) {
	var tests = []struct {
		Name         string
		AuthMethod   any
		IdentityType any
		ExpectError  string
		ExpectWarn   string
	}{
		{
			Name:         "Neither set",
			AuthMethod:   nil,
			IdentityType: nil,
			ExpectError:  `Attribute "type" must be set.`,
		},
		{
			Name:         "Both set",
			AuthMethod:   "bearer",
			IdentityType: "bearer",
			ExpectError:  `Attributes "auth_method" and "type" cannot both be set. Use "type".`,
		},
		{
			Name:         "Only type set",
			AuthMethod:   nil,
			IdentityType: "bearer",
		},
		{
			Name:         "Only auth method set",
			AuthMethod:   "bearer",
			IdentityType: nil,
			ExpectWarn:   `Use "type" instead.`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ctx := t.Context()
			raw := identityObject(test.AuthMethod, test.IdentityType)

			resourceResp := &resource.ValidateConfigResponse{}
			identityTypeValidator{warnOnAuthMethod: true}.ValidateResource(ctx, resource.ValidateConfigRequest{
				Config: tfsdk.Config{
					Raw: raw,
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"auth_method": schema.StringAttribute{Optional: true, Computed: true},
							"type":        schema.StringAttribute{Optional: true, Computed: true},
						},
					},
				},
			}, resourceResp)

			dataSourceResp := &datasource.ValidateConfigResponse{}
			identityTypeValidator{warnOnAuthMethod: false}.ValidateDataSource(ctx, datasource.ValidateConfigRequest{
				Config: tfsdk.Config{
					Raw: raw,
					Schema: dsschema.Schema{
						Attributes: map[string]dsschema.Attribute{
							"auth_method": dsschema.StringAttribute{Optional: true, Computed: true},
							"type":        dsschema.StringAttribute{Optional: true, Computed: true},
						},
					},
				},
			}, dataSourceResp)

			if test.ExpectError != "" {
				require.True(t, resourceResp.Diagnostics.HasError())
				require.True(t, dataSourceResp.Diagnostics.HasError())
				assert.Contains(t, resourceResp.Diagnostics.Errors()[0].Detail(), test.ExpectError)
				assert.Contains(t, dataSourceResp.Diagnostics.Errors()[0].Detail(), test.ExpectError)
				return
			}

			require.False(t, resourceResp.Diagnostics.HasError())
			require.False(t, dataSourceResp.Diagnostics.HasError())

			// The warning is emitted by the resource only.
			assert.Empty(t, dataSourceResp.Diagnostics.Warnings())

			if test.ExpectWarn == "" {
				assert.Empty(t, resourceResp.Diagnostics.Warnings())
				return
			}

			require.Len(t, resourceResp.Diagnostics.Warnings(), 1)
			assert.Contains(t, resourceResp.Diagnostics.Warnings()[0].Detail(), test.ExpectWarn)
		})
	}
}
