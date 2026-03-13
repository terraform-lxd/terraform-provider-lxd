# lxd_auth_group

Manages a LXD authorization group.

## Example Usage

```hcl
resource "lxd_instance" "inst" {
  name    = "c1"
  image   = "ubuntu:24.04"
  project = "default"
}

resource "lxd_auth_group" "group" {
  name = "c1-viewer"
  permissions = [
    {
      entitlement = "can_view"
      entity_type = "project"
      entity_args = {
        name = lxd_instance.inst.project
      }
    },
    {
      entitlement = "can_view"
      entity_type = "instance"
      entity_args = {
        name    = lxd_instance.inst.name
        project = lxd_instance.inst.project
      }
    }
  ]
}
```

## Argument Reference

* `name` - **Required** - Name of the group.

* `description` - *Optional* - Description of the group.

* `permissions` - *Optional* - List of group permissions. If not specified, the group has no permissions. Please, refer to the [official LXD documentation](https://documentation.ubuntu.com/lxd/latest/reference/permissions/) for available entity types and entitlements that can be granted against each entity type.

The `permissions` list element supports:

* `entity_type` - **Required** - Entity type represents LXD API resource. Examples: `server`, `project`, `instance`.

* `entitlement` - **Required** - Entitlement granted against the specified entity type. Available values depend on the `entity_type`. Examples: `can_edit`, `can_view`, `can_delete`.

* `entity_args` - **Optional** - Map of key-value pairs used to identify a specific entity. Available keys depend on the `entity_type`, and are not required for certain entity types, such as `server`.

## Importing

Import ID syntax: `[<remote>:]<group>`

* `<remote>` - *Optional* - Remote name.
* `<name>` - **Required** - Authorization group name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_auth_group.mygroup group1
```

Example using the import block:

```hcl
resource "lxd_auth_group" "mygroup" {
  name        = "group1"
}

import {
  to = lxd_auth_group.mygroup
  id = "group1"
}
```
