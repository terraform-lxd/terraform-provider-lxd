# lxd_volume_container_attach

Manages an attachment between an LXD volume and container.

This resource has been deprecated. You can attach volumes to
containers and profiles by creating the appropriate `device`
configuration.

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
  pool = "${lxd_storage_pool.pool1.name}"
}

resource "lxd_container" "container1" {
  name     = "%s"
  image    = "ubuntu"
  profiles = ["default"]
}

resource "lxd_volume_container_attach" "attach1" {
  pool           = "${lxd_storage_pool.pool1.name}"
  volume_name    = "${lxd_volume.volume1.name}"
  container_name = "${lxd_container.container1.name}"
  path           = "/tmp"
}
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `pool` - *Required* - Name of the volume's storage pool.

* `volume_name` - *Required* - Name of the volume to attach.

* `container_name` - *Required* - Name of the container to attach the volume to.

* `path` - *Required* - Mountpoint of the volume in the container.

* `device_name` - *Optional* - The volume device name as seen by the container.
	By default, this will be the volume name.
