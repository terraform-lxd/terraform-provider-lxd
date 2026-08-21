package auth

import (
	"github.com/canonical/lxd/shared/api"
)

// toAuthMethod returns the authentication method for the given identity type.
// The authentication method is the path segment of every identity endpoint,
// and a devlxd identity is reached through the bearer method. Every other
// identity type the provider accepts is also an authentication method.
func toAuthMethod(identityType string) string {
	if identityType == "devlxd" {
		return api.AuthenticationMethodBearer
	}

	return identityType
}

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
	case api.IdentityTypeBearerTokenDevLXD:
		return "devlxd"
	case api.IdentityTypeOIDCClient:
		// Only the data source accepts oidc.
		return "oidc"
	}

	return ""
}
