# incus_volume_copy

Copies an existing Incus volume.

## Example Usage

```hcl
resource "incus_storage_pool" "pool1" {
  name   = "mypool"
  driver = "dir"
}

resource "incus_volume" "volume1" {
  name = "myvolume"
  pool = incus_storage_pool.pool1.name
}

resource "incus_volume_copy" "volume1_copy" {
  name        = "myvolume_copy"
  pool        = incus_storage_pool.pool1.name
  source_pool = incus_storage_pool.pool1.name
  source_name = incus_volume.volume1.name
}
```

## Argument Reference

* `name` - **Required** - Name of the storage volume.

* `pool` - **Required** - The storage pool that will receive the copy of the volume copy.

* `source_pool` - **Required** - The storage pool that hosts the existing volume to use as the source.

* `source_name` - **Required** - Name of the existing storage volume that is to be copied.

* `source_remote` - *Optional* - The remote from which the source volume is to be copied. If
	it is not provided, the default provider remote is used.

* `project` - *Optional* - Name of the target project where the volume will be copied to.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

No attributes are exported.

## Notes

* [Incus move/copy documentation](https://linuxcontainers.org/incus/docs/main/howto/storage_move_volume/).
