# lxd_storage_pool

Manages an LXD storage pool.

## Example Usage

```hcl
resource "lxd_storage_pool" "pool1" {
  name = "mypool"
  driver = "dir"
  config = {
    source = "/var/lib/lxd/storage-pools/mypool"
  }
}
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `name`   - *Required* - Name of the storage pool.

* `driver` - *Required* - Storage Pool driver. Must be one of `dir`, `lvm`,
	`btrfs`, or `zfs`.

* `config` - *Required* - Map of key/value pairs of
	[storage pool config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md).
	Config settings vary from driver to driver.

## Importing

Storage pools can be imported by doing:

```shell
$ terraform import lxd_storage_pool.my_pool <name of pool>
```
