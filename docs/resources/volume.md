# lxd_volume

Manages an LXD volume.

## Example Usage

```hcl
resource "lxd_storage_pool" "pool1" {
  name   = "mypool"
  driver = "zfs"
}

resource "lxd_volume" "volume1" {
  name = "myvolume"
  pool = lxd_storage_pool.pool1.name
}
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `name` - *Required* - Name of the storage volume.

* `target` - *Optional* - Specify a target node in a cluster.

* `pool` - *Required* - The Storage Pool to host the volume.

* `type` - *Optional* - The "type" of volume. The default value is `custom`,
	which is the type to use for storage volumes attached to containers.

* `config` - *Optional* - Map of key/value pairs of
	[volume config settings](https://documentation.ubuntu.com/lxd/en/latest/reference/storage_drivers/).
	Config settings vary depending on the Storage Pool used.

* `project` - *Optional* - Name of the project where the volume will be stored.

* `content_type` - *Optional* - Volume content type (filesystem or block)

## Attribute Reference

The following attributes are exported:

* `location` - Name of the node where volume was created. It could be useful with LXD in cluster mode.

## Importing

Storage pool volumes can be imported with the following command:

```shell
$ terraform import lxd_volume.my_vol [<remote>:][<project>]/<pool_name>/<volume_name>
```

## Notes

* Technically, an LXD volume is simply a container or profile device of
  type `disk`.
