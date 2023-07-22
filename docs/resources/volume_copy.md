# lxd_volume_copy

Copies an existing LXD volume.

## Example Usage

```hcl
resource "lxd_storage_pool" "pool1" {
  name   = "mypool"
  driver = "dir"
  config = {
    source = "/var/lib/lxd/storage-pools/mypool"
  }
}

resource "lxd_volume" "volume1" {
  name = "myvolume"
  pool = lxd_storage_pool.pool1.name
}

resource "lxd_volume_copy" "volume1_copy" {
  name        = "myvolume_copy"
  pool        = lxd_storage_pool.pool1.name
  source_pool = lxd_storage_pool.pool1.name
  source_name = lxd_volume.volume1.name
}
```

## Argument Reference

* `name` - *Required* - Name of the storage volume.

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `target` - *Optional* - Specify a target node in a cluster.

* `pool` - *Required* - The Storage Pool to host the volume copy.

* `source_remote` - *Optional* - The remote from which the source volume is to be copied. If
	it is not provided, the default provider remote is used.

* `source_pool` - *Required* - The Storage Pool that hosts the existing volume that is to be copied.

* `source_name` - *Required* - Name of the existing storage volume that is to be copied.


## Notes

[LXD move/copy documentation](https://documentation.ubuntu.com/lxd/en/latest/howto/storage_move_volume/).
