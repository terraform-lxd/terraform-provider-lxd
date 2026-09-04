# lxd_auth_identity_token

The `lxd_auth_identity_token` resource allows you to issue a token for an LXD
identity of type `bearer`.

## Required API extensions

The target LXD server must support the following API extensions:

* `access_management_expiry`
* `auth_bearer`

~> **Warning:** The issued token is stored in the Terraform state in plain text.
  Anyone who can read the state can authenticate against the LXD server as the
  identity that bears the token, therefore the state must be treated as a secret
  and stored accordingly.

~> **Note:** LXD identities bear at most one token. Issuing a token revokes the
  token that the identity currently bears, therefore a single identity must not be
  referenced by multiple `lxd_auth_identity_token` resources.

## Example Usage

```hcl
resource "lxd_auth_identity" "identity" {
  type   = "bearer"
  name   = "server-admin"
  groups = ["admins"]
}

resource "lxd_auth_identity_token" "token" {
  identity = lxd_auth_identity.identity.name
  expiry   = "30d"
}

output "token" {
  value     = lxd_auth_identity_token.token.token
  sensitive = true
}
```

## Argument Reference

* `identity` - **Required** - Name of the bearer identity for which the token is issued.

* `expiry` - *Optional* - Token expiry as a space separated list of durations in
  the form `(\d)+(S|M|H|d|w|m|y)`, for example `30d` or `1H 30M`. If not provided,
  the server's default expiry is used.

* `remote` - *Optional* - The remote in which the token is issued. If not provided,
  the provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

* `token` - The issued token.

* `expires_at` - Time at which the token expires, in RFC3339 format.

## Token validity

A token that has expired is reissued.

The validity of a token that has not expired yet is determined by the expiry that
the LXD server reports for it. The token is reissued when the reported expiry
differs from `expires_at`, meaning a new token was issued for the identity, and
when no expiry is reported at all, meaning the token was revoked.

## Notes

* Changing any attribute of this resource replaces it, which issues a new token.

* Destroying this resource revokes the token that the identity currently bears.

* See the LXD [documentation](https://documentation.ubuntu.com/lxd/latest/authentication/)
  for more information on bearer identities.
