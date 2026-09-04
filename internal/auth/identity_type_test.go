package auth

import (
	"testing"

	"github.com/canonical/lxd/shared/api"
	"github.com/stretchr/testify/assert"
)

func TestToType(t *testing.T) {
	var tests = []struct {
		LXDIdentityType string
		Expect          string
	}{
		{LXDIdentityType: api.IdentityTypeCertificateClient, Expect: "tls"},
		{LXDIdentityType: api.IdentityTypeCertificateClientRestricted, Expect: "tls"},
		{LXDIdentityType: api.IdentityTypeCertificateClientUnrestricted, Expect: "tls"},
		{LXDIdentityType: api.IdentityTypeCertificateClientPending, Expect: "tls"},
		{LXDIdentityType: api.IdentityTypeBearerTokenClient, Expect: "bearer"},
		{LXDIdentityType: api.IdentityTypeBearerTokenDevLXD, Expect: ""},
		{LXDIdentityType: api.IdentityTypeOIDCClient, Expect: "oidc"},
		{LXDIdentityType: api.IdentityTypeCertificateServer, Expect: ""},
		{LXDIdentityType: api.IdentityTypeCertificateMetricsRestricted, Expect: ""},
		{LXDIdentityType: api.IdentityTypeCertificateMetricsUnrestricted, Expect: ""},
		{LXDIdentityType: api.IdentityTypeCertificateClusterLink, Expect: ""},
		{LXDIdentityType: api.IdentityTypeCertificateClusterLinkPending, Expect: ""},
		{LXDIdentityType: api.IdentityTypeBearerTokenInitialUI, Expect: ""},
		{LXDIdentityType: "", Expect: ""},
	}

	for _, test := range tests {
		t.Run(test.LXDIdentityType, func(t *testing.T) {
			assert.Equal(t, test.Expect, toType(test.LXDIdentityType))
		})
	}
}
