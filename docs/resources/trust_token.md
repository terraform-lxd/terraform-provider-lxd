# lxd_trust_token

The `lxd_trust_token` resource allows you to request a new trust token.

~> **Note:** The LXD trust token resource cannot be used for the initial authentication
  with the LXD server because LXD Terraform provider needs to be authenticated in order
  to request trust tokens for other clients.

## Example Usage

```hcl
resource "lxd_trust_token" "token1" {
  name = "mytoken"
}

output "token" {
  value = lxd_trust_token.token1.token
}
```

## Argument Reference

* `name` - **Required** - Name of the token.

* `projects` - *Optional* - List of projects to restrict the token to.

* `remote` - *Optional* - The remote in which the resource will be created. If not provided,
  the provider's default remote will be used.

* `trigger` - *Optional* - When to trigger the token generation. Possible values are `once` and `always` (if missing). Defaults to `once`.

## Attribute Reference

The following attributes are exported:

* `token` - The generated token.

* `expires_at` - Time at which the trust token expires. If token expiry is configured, the value will be in format `YYYY/MM/DD hh:mm TZ`.

## Trust token expiry

~> **Warning:** The provider is unable to differentiate between an expired and a consumed token.
  If token generation is set to `once` and the token expires, it will not be regenerated.

Trust token expiry is defined in the server's configuration (`core.remote_token_expiry`).
If the setting is configured, `expires_at` attribute will be populated, otherwise, it will be empty.

For example, to set the token expiry to 20 minutes, run the following command:
```
lxc config set core.remote_token_expiry=20M
```

If trigger is set to `once` the token will not be regenerated on subsequent plan applies.
By setting the trigger to `always` ensures that the token is always present, and will be regenerated if missing.

## Notes

* Token's unique identifier is the operation ID and not the token name. Therefore, multiple tokens can exist with the same name.

* See the LXD [documentation](https://documentation.ubuntu.com/lxd/en/latest/authentication/#authentication-token) for more information on trust tokens.
