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

## Argument Reference

* `name` - **Required** - Name of the identity.

* `auth_method` - **Required** - Authentication method, can be either `tls` or `bearer`.

* `groups` - *Optional* - List of group names to add this identity to.

* `tls_certificate` - *Optional* - PEM encoded x509 certificate. Must be set when authentication method is `tls`.

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
