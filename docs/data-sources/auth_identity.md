# lxd_auth_identity

Provides information about an existing LXD identity.

## Example Usage

```hcl
data "lxd_auth_identity" "id" {
  type = "bearer"
  name = "my-identity"
}
```

## Argument Reference

* `name` - **Required** - Name of the identity.

* `type` - **Required** - Identity type, can be `tls`, `bearer`, or `oidc`.
  See [Note on `auth_method`](#note-on-auth_method).

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `auth_method` - Authentication method of the identity, can be `tls`, `bearer`, or `oidc`.

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

## Note on `auth_method`

The `auth_method` represents the authentication method and `type` represents the type of identity.
Before argument `type` was introduced, the `auth_method` was a required field.
A single `auth_method`, such as `bearer`, represents multiple identity types.

To prevent breaking changes, the shift from `auth_method` to `type` was handled gracefully allowing
any of the two values to be provided. However, in the future releases, the `auth_method` will become
a computed field and `type` required.
