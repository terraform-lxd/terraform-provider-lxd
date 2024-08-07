# lxd_trust_token

The `lxd_trust_token` resource allows you to create new tokens in the LXD trust store.

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

## Attribute Reference

The following attributes are exported:

* `token` - The generated token.

* `expires_at` - Time when trust token expires. If token expiry is configured, the value will be in format `YYYY/MM/DD hh:mm TZ`.

## Trust token expiry

Trust token expiry is defined by the system configuration `core.remote_token_expiry`.
If the setting is configured, `expires_at` attribute will be populated, otherwise, it will be empty.

For example, to set the token expiry to 20 minutes, run the following command:
```
lxc config set core.remote_token_expiry=20M
```

If token expires or manually is removed, a new one will be created. Otherwise, the existing one is returned.

## Notes

* Token's unique identifier is the operation ID and not the token name. Therefore, multiple tokens can exist with the same name.

* See the LXD [documentation](https://documentation.ubuntu.com/lxd/en/latest/authentication/#authentication-token) for more information on trust tokens.
