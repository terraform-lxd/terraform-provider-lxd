# lxd_auth_identity

Manages LXD identities.

## Example Usage

```hcl
resource "lxd_auth_identity" "bearer-identity" {
  auth_method = "bearer"
  name        = "bearer-server-admin"
  groups      = ["admins"]
}
```

```hcl
resource "lxd_auth_identity" "tls-identity" {
  auth_method     = "tls"
  name            = "tls-server-admin"
  groups          = ["admins"]
  tls_certificate = file("client.cert")
}
```

### Pending TLS identity

If `tls_certificate` is omitted for a `tls` identity, LXD creates the identity in a
pending state and issues a trust token. The token is handed to an untrusted client,
which redeems it to enroll its own certificate:

```hcl
resource "lxd_auth_identity" "pending-identity" {
  auth_method = "tls"
  name        = "jane"
  groups      = ["admins"]
}

output "trust_token" {
  value     = lxd_auth_identity.pending-identity.trust_token
  sensitive = true
}
```

The client then runs `lxc remote add <remote> <token>`.

Requires the `access_management_tls` API extension.

## Argument Reference

* `name` - **Required** - Name of the identity.

* `auth_method` - **Required** - Authentication method, can be either `tls` or `bearer`.

* `groups` - *Optional* - List of group names to add this identity to.

* `tls_certificate` - *Optional* - PEM encoded x509 certificate. Applicable only when the
  authentication method is `tls`. If omitted, a pending TLS identity is created and a trust
  token is issued instead. Once a TLS identity is created, removing this attribute from the
  configuration leaves the certificate intact.

* `remote` - *Optional* - The remote in which the resource will be created. If not provided,
  the provider's default remote will be used.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `trust_token` - Trust token that an untrusted client uses to enroll its certificate. It is
  set only for a pending TLS identity, and is cleared once the token has been redeemed.

* `expires_at` - Time at which the trust token expires, in RFC 3339 format and UTC. It is null when
  the server config `core.remote_token_expiry` is not set, which means the token does not expire.

## Pending TLS trust token lifecycle

A trust token is returned only when a pending TLS identity is created, which means no
`tls_certificate` was set, and can never be retrieved from the server afterwards. LXD
provides no way to reissue a token for an existing identity, so if the token expires before
it is redeemed, the identity is unusable and Terraform plans a replacement of the resource.

Setting `tls_certificate` on a TLS identity whose token is still outstanding also plans a
replacement. LXD does not allow a certificate to be set on a pending identity, and the
outstanding token is invalidated along with the identity it was issued for.

Once the token is redeemed, the identity holds a certificate and is never replaced on that
account. Subsequent changes, such as group membership or a new `tls_certificate`, are
applied in place.

An imported pending TLS identity has no `trust_token`, because the token cannot be read back
from the server.

## Importing

Import ID syntax: `[<remote>:]/<auth_method>/<name>`

* `<remote>` - *Optional* - Remote name.
* `<auth_method>` - **Required** - Authentication method.
* `<name>` - **Required** - Identity name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_auth_identity.myidentity /bearer/identity1
```

Example using the import block:

```hcl
resource "lxd_auth_identity" "myidentity" {
  name        = "identity1"
  auth_method = "bearer"
}

import {
  to = lxd_auth_identity.myidentity
  id = "/bearer/identity1"
}
```
