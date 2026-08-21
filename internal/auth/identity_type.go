package auth

import (
	"github.com/canonical/lxd/shared/api"
)

// toType returns the identity type for the given LXD identity type. LXD
// identity types that the provider cannot name return an empty string.
func toType(lxdIdentityType string) string {
	switch lxdIdentityType {
	case api.IdentityTypeCertificateClient,
		api.IdentityTypeCertificateClientRestricted,
		api.IdentityTypeCertificateClientUnrestricted,
		api.IdentityTypeCertificateClientPending:
		return "tls"
	case api.IdentityTypeBearerTokenClient:
		return "bearer"
	case api.IdentityTypeOIDCClient:
		// Only the data source accepts oidc.
		return "oidc"
	}

	return ""
}
