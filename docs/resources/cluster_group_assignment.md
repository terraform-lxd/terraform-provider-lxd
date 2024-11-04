# incus_cluster_group_assignment

Manages an Incus cluster group assignment.

## Example Usage

```hcl
resource "incus_cluster_group" "amd64" {
  name        = "amd64"
  description = "x86-64 nodes"
}

resource "incus_cluster_group_assignment" "node_11" {
  cluster_group = incus_cluster_group.amd64.name
  member        = "node_1"
}
```

## Argument Reference

* `cluster_group` - **Required** - Name of the cluster group.

* `member` - **Required** - Name of the cluster group member.

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

No attributes are exported.

## Importing

Cluster groups can be imported with the following command:

```shell
terraform import incus_cluster_group_assignment.member [<remote>:]/<cluster_group>/<member>
```

## Importing

Import ID syntax: `[<remote>:]/<cluster_group>/<member>`

* `<remote>` - *Optional* - Remote name.
* `<cluster_group>` - **Required** - Cluster group name.
* `<member>` - **Required** - Cluster group member name.

### Import example

Example using terraform import command:

```shell
terraform import incus_cluster_group_assignment.node_1 /my-cluster/node-1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "incus_cluster_group_assignment" "node_1" {
  cluster_group = "my-cluster"
  member        = "node-1"
}

import {
  to = incus_cluster_group.mygroup
  id = "/my-cluster/node-1"
}
```