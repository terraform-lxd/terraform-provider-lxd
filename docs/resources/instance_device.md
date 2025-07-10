# lxd_instance_device

Manages a single device in an LXD instance. Provides an alternative way to attach
and manage a device.

This resource is useful for managing devices on a LXD instance in a cluster.
For example, it allows attaching volumes to an instance with initially unknown
location in the cluster.

~> **Warning:** This is an experimental feature of Terraform LXD Provider and it may
   change in the future.

## Example

```hcl
resource "lxd_instance" "instance1" {
  name  = "instance1"
  image = "ubuntu-daily:22.04"

  # Device attached during instance creation.
  device {
    name = "eth0"
    type = "nic"

    properties = {
      network = "lxdbr0"
    }
  }
}

resource "lxd_volume" "vol1" {
  name   = "vol1"
  pool   = "default"
  type   = "disk"

  # In case we want to create the volume on the same cluster member
  # where the instance has been created, the "target" needs to match.
  #
  # However, this prevents us from using device block in instance
  # resource, as it would create a dependency loop. Therefore, we
  # will use "lxd_instance" resource, which allows device attachment
  # after instance creation.
  target = lxd_instance.instance1.location

  config {
    size = "10GB"
  }
}

# Device attachment after instance creation.
resource "lxd_device" "vol1" {
  instance     = lxd_instance.instance1.name # Target instance.
  device_name  = lxd_volume.vol1.name        # Target volume, which is created after instance creation.
  type         = "disk"

  properties = {
    path   = "/mnt/vol1"
    pool   = "default"
    source = lxd_volume.vol1.name
  }
}
```

## Argument Reference

* `name` - **Required** - Name of the device.

* `instance_name` - **Required** - Name of the instance.

* `project` - *Optional* - Name of the project where the instance to which this device will be attached exists

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target cluster member or cluster member group of the instance.

* `type` - **Required** - Type of the device Must be one of none, disk, nic,
	unix-char, unix-block, usb, gpu, infiniband, proxy, unix-hotplug, tpm, pci.

* `properties`- **Required** - Map of key/value pairs of
	[device properties](https://documentation.ubuntu.com/lxd/latest/reference/devices/).

## Notes

* Terraform LXD provider sets user.managed-by key to all managed instance devices.
  Removing that key from a device manually, would result in Terraform removing it on next apply.

