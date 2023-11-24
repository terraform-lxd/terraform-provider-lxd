# lxd_storage_pool

Manages an LXD storage pool.

## Example Usage

### Basic Example

```hcl
resource "lxd_storage_pool" "pool1" {
  name   = "mypool"
  driver = "dir"
  config = {
    source = "/var/lib/lxd/storage-pools/mypool"
  }
}
```

### Cluster Example

In order to create a storage pool in a cluster, you first have to define
the storage pool on each node in the cluster. Then you can create the
actual pool

```hcl
resource "lxd_storage_pool" "mypool_node1" {
  name   = "mypool"
  driver = "zfs"
  target = "node1"
}

resource "lxd_storage_pool" "mypool_node2" {
  name   = "mypool"
  driver = "zfs"
  target = "node2"
}

resource "lxd_storage_pool" "mypool" {
  depends_on = [
    lxd_storage_pool.mypool_node1,
    lxd_storage_pool.mypool_node2,
  ]

  name   = "mypool"
  driver = "zfs"
}
```

Please see the [LXD Clustering documentation](https://documentation.ubuntu.com/lxd/en/latest/howto/cluster_config_storage/)
for more details on how to create a storage pool in clustered mode.

## Argument Reference

* `name`   - **Required** - Name of the storage pool.

* `driver` - **Required** - Storage Pool driver. Must be one of `dir`, `zfs`, `lvm`, `btrfs`,`ceph`, `cephfs`, or `cephobject`.

* `description` - *Optional* - Description of the storage pool.

* `config` - *Optional* - Map of key/value pairs of
	[storage pool config settings](https://documentation.ubuntu.com/lxd/en/latest/reference/storage_drivers/).
	Config settings vary from driver to driver.

* `project` - *Optional* - Name of the project where the storage pool will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `target` - *Optional* - Specify a target node in a cluster.

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<name>` - **Required** - Storage pool name.

-> Clustered networks cannot be imported.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_storage_pool.mypool proj/pool1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_storage_pool" "mypool" {
  name    = "pool1"
  project = "proj"
  driver  = "zfs"
}

import {
    to = lxd_storage_pool.mypool
    id = "proj/pool1"
}
```
