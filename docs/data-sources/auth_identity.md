# lxd_auth_identity

Provides information about an existing LXD identity.

## Example Usage

```hcl
data "lxd_auth_identity" "id" {
  auth_method = "bearer"
  name        = "my-identity"
}
```

## Argument Reference

* `name` - **Required** - Name of the identity.

* `auth_method` - **Required** - Authentication method, can be either `tls`, `bearer`, or `oidc`.

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
