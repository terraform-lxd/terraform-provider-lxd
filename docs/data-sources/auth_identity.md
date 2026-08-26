# lxd_auth_identity

Provides information about an existing LXD identity.

## Example Usage

```hcl
data "lxd_auth_identity" "id" {
  type = "devlxd"
  name = "my-identity"
}
```

## Argument Reference

* `name` - **Required** - Name of the identity.

* `type` - *Optional* - Identity type, can be `tls`, `bearer`, `devlxd`, or `oidc`. The lookup fails
  if the identity that is found is of another type. Exactly one of `type` and `auth_method` must be
  set.

* `auth_method` - *Optional* - Authentication method, can be `tls`, `bearer`, or `oidc`. Exactly one
  of `type` and `auth_method` must be set.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `groups` - List of group names the identity is part of.

* `tls_certificate` - PEM encoded x509 certificate. Populated only for TLS identities.
	It is empty for a pending TLS identity that has not yet redeemed its trust token.

* `identifier` - Identity ID. For a pending TLS identity this is a UUID, which LXD replaces
	with the certificate fingerprint once the trust token is redeemed. Use `name` instead if
	you need a value that is stable across that transition.

* `expires_at` - Expiry of the identity's credential, in RFC3339 format. For bearer
	identities this is the expiry of the token that the identity currently bears, and
	it is empty once the identity bears no token. Requires the `access_management_expiry`
	API extension.

* `pending` - Whether the identity has no credential, because none was issued yet or the
	issued one was revoked. A TLS identity is pending until its token is used. A bearer or
	`devlxd` identity is pending until a token is issued for it, and again once that token is
	revoked. Bearer identities require the `access_management_bearer_pending` API extension to
	report pending, and read as not pending without it.

Both `type` and `auth_method` are populated on every read.
