# lxd_auth_identity_provider_group

Manages a LXD identity provider group, which maps a group defined by the identity provider to LXD authorization groups.

## Example Usage

```hcl
resource "lxd_auth_group" "viewers" {
  name = "viewers"
  permissions = [
    {
      entitlement = "viewer"
      entity_type = "server"
    }
  ]
}

resource "lxd_auth_identity_provider_group" "viewers" {
  name   = "viewers"
  groups = [lxd_auth_group.viewers.name]
}
```

## Argument Reference

* `name` - **Required** - Name of the identity provider group, as it appears in the OIDC token claim.

* `groups` - *Optional* - List of authorization group names the identity provider group is mapped to. If not specified, the identity provider group is mapped to no authorization groups.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

## Importing

This resource is imported using its resource identity.

* `name` - **Required** - Identity provider group name.
* `remote` - *Optional* - Remote name. If not provided, the provider's default remote is used.

### Import example

```hcl
resource "lxd_auth_identity_provider_group" "viewers" {
  name = "viewers"
}

import {
  to = lxd_auth_identity_provider_group.viewers
  identity = {
    name = "viewers"
  }
}
```
