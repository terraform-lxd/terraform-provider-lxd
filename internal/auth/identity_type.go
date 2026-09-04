package auth

import (
	"github.com/canonical/lxd/shared/api"
)

// Pending bearer identity types, reported by LXD with the
// access_management_bearer_pending API extension. The LXD api package used by
// the provider does not export them yet.
const (
	identityTypeBearerTokenClientPending = "Client token bearer (pending)"
	identityTypeBearerTokenDevLXDPending = "DevLXD token bearer (pending)"
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
	case api.IdentityTypeBearerTokenClient,
		identityTypeBearerTokenClientPending:
		return "bearer"
	case api.IdentityTypeBearerTokenDevLXD,
		identityTypeBearerTokenDevLXDPending:
		return "devlxd"
	case api.IdentityTypeOIDCClient:
		// Only the data source accepts oidc.
		return "oidc"
	}

	return ""
}

// isPending reports whether the given LXD identity type is a pending one. A
// pending identity has no credential, because none was issued yet or the
// issued one was revoked.
func isPending(lxdIdentityType string) bool {
	switch lxdIdentityType {
	case api.IdentityTypeCertificateClientPending,
		api.IdentityTypeCertificateClusterLinkPending,
		identityTypeBearerTokenClientPending,
		identityTypeBearerTokenDevLXDPending:
		return true
	}

	return false
}
