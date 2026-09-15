# lxd_auth_identity_provider_group

Provides information about an existing LXD identity provider group.

## Example Usage

```hcl
data "lxd_auth_identity_provider_group" "viewers" {
  name = "viewers"
}
```

## Argument Reference

* `name` - **Required** - Name of the identity provider group.

* `remote` - *Optional* - The remote from which the identity provider group is retrieved. If
  not provided, the provider's default remote is used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `groups` - List of authorization group names the identity provider group is mapped to.
