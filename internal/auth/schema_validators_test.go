package auth

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
			identityTypeValidator{}.ValidateResource(ctx, resource.ValidateConfigRequest{
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
			identityTypeValidator{}.ValidateDataSource(ctx, datasource.ValidateConfigRequest{
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

			if test.ExpectWarn == "" {
				assert.Empty(t, resourceResp.Diagnostics.Warnings())
				assert.Empty(t, dataSourceResp.Diagnostics.Warnings())
				return
			}

			require.Len(t, resourceResp.Diagnostics.Warnings(), 1)
			require.Len(t, dataSourceResp.Diagnostics.Warnings(), 1)
			assert.Contains(t, resourceResp.Diagnostics.Warnings()[0].Detail(), test.ExpectWarn)
			assert.Contains(t, dataSourceResp.Diagnostics.Warnings()[0].Detail(), test.ExpectWarn)
		})
	}
}

func TestIdentityCertificateValidator(t *testing.T) {
	var tests = []struct {
		Name         string
		AuthMethod   any
		IdentityType any
		Certificate  any
		ExpectError  string
	}{
		{
			Name:         "No certificate",
			IdentityType: "bearer",
		},
		{
			Name:         "Certificate on tls type",
			IdentityType: "tls",
			Certificate:  "cert",
		},
		{
			Name:        "Certificate on tls auth method",
			AuthMethod:  "tls",
			Certificate: "cert",
		},
		{
			Name:         "Certificate on unknown type",
			IdentityType: tftypes.UnknownValue,
			Certificate:  "cert",
		},
		{
			Name:         "Unknown certificate on bearer type",
			IdentityType: "bearer",
			Certificate:  tftypes.UnknownValue,
		},
		{
			Name:         "Certificate on devlxd type",
			IdentityType: "devlxd",
			Certificate:  "cert",
			ExpectError:  `Certificate must not be set for identities of type "devlxd"`,
		},
		{
			Name:        "Certificate on bearer auth method",
			AuthMethod:  "bearer",
			Certificate: "cert",
			ExpectError: `Certificate must not be set for identities of type "bearer"`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			raw := tftypes.NewValue(
				tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"auth_method":     tftypes.String,
						"type":            tftypes.String,
						"tls_certificate": tftypes.String,
					},
				},
				map[string]tftypes.Value{
					"auth_method":     tftypes.NewValue(tftypes.String, test.AuthMethod),
					"type":            tftypes.NewValue(tftypes.String, test.IdentityType),
					"tls_certificate": tftypes.NewValue(tftypes.String, test.Certificate),
				},
			)

			resp := &resource.ValidateConfigResponse{}
			identityCertificateValidator{}.ValidateResource(t.Context(), resource.ValidateConfigRequest{
				Config: tfsdk.Config{
					Raw: raw,
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"auth_method":     schema.StringAttribute{Optional: true, Computed: true},
							"type":            schema.StringAttribute{Optional: true, Computed: true},
							"tls_certificate": schema.StringAttribute{Optional: true, Computed: true},
						},
					},
				},
			}, resp)

			if test.ExpectError == "" {
				require.False(t, resp.Diagnostics.HasError())
				return
			}

			require.True(t, resp.Diagnostics.HasError())
			assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), test.ExpectError)
		})
	}
}
