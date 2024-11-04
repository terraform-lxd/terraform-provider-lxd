# incus_cluster_group

Manages an Incus cluster group.

## Example Usage

```hcl
resource "incus_cluster_group" "amd64" {
  name        = "amd64"
  description = "x86-64 nodes"
}
```

## Argument Reference

* `name` - **Required** - Name of the cluster group.

* `config` - *Optional* - Map of key/value pairs of
  [cluster group config settings](https://linuxcontainers.org/incus/docs/main/howto/cluster_groups/#configuration-options).

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

No attributes are exported.

## Importing

Cluster groups can be imported with the following command:

```shell
terraform import incus_cluster_group.my_group [<remote>:]<name>
```

## Importing

Import ID syntax: `[<remote>:]<name>`

* `<remote>` - *Optional* - Remote name.
* `<name>` - **Required** - Cluster group name.

### Import example

Example using terraform import command:

```shell
terraform import incus_cluster_group.my_group my_group
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "incus_cluster_group" "my_group" {
  name = "my_group"
}

import {
  to = incus_cluster_group.my_group
  id = "my_group"
}
```