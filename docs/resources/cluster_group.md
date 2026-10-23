# lxd_cluster_group

Manages an LXD cluster group.

Cluster groups are named sets of cluster members. They can be used as placement targets for
instances (`--target @<group>`) and to restrict projects to a subset of the cluster members.

## LXD documentation

* [How to set up cluster groups](https://documentation.ubuntu.com/lxd/latest/howto/cluster_groups/)

## Required API extensions

The target LXD server must support the `clustering_groups` API extension.

## Example Usage

```hcl
resource "lxd_cluster_group" "amd64" {
  name        = "amd64"
  description = "x86-64 members"
  members     = ["node-1", "node-2"]
}
```

## Argument Reference

* `name` - **Required** - Name of the cluster group.

* `description` - *Optional* - Description of the cluster group.

* `members` - *Optional* - Set of cluster member names in the group. A member can belong to
  several groups, but must belong to at least one, so LXD rejects removing a member from its
  only group.

* `remote` - *Optional* - The remote in which the resource will be created. If not provided, the
  provider's default remote will be used.

## Attribute Reference

No attributes are exported.

## Importing

Import ID syntax: `[<remote>:]<name>`

* `<remote>` - *Optional* - Remote name.
* `<name>` - **Required** - Cluster group name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_cluster_group.amd64 amd64
```

Example using the import block:

```hcl
resource "lxd_cluster_group" "amd64" {
  name = "amd64"
}

import {
  to = lxd_cluster_group.amd64
  id = "amd64"
}
```

## Notes

* Every cluster has a `default` group that contains all members. It cannot be deleted, and
  adding a member to another group does not remove it from `default`.

* LXD deletes only empty cluster groups, so destroying a group that has members first removes
  them from the group. A group that is in use, for example by a project's
  `restricted.cluster.groups` setting, cannot be deleted until the reference is removed.
