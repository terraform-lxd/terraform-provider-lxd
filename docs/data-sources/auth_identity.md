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

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `groups` - List of group names the identity is part of.

* `tls_certificate` - PEM encoded x509 certificate. Populated only for TLS identities.

* `identifier` - Identity ID.
