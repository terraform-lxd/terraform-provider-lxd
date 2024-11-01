# incus_profile

Provides information about an Incus profile.

## Example Usage

```hcl
data "incus_profile" "default" {
  name = "default"
}

resource "incus_instance" "d1" {
  profiles = [data.incus_profile.default.name]
  image    = "images:debian/12"
  name     = "d1"
}
```

## Argument Reference

* `name` - **Required** - Name of the profile.

* `project` - *Optional* - Name of the project where the profile will be stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

* `device` - Device definition. See reference below.

* `config` - Map of key/value pairs of
  [instance config settings](https://linuxcontainers.org/incus/docs/main/reference/instance_options/).

The `device` block supports:

* `name` - Name of the device.

* `type` - Type of the device Must be one of none, disk, nic,
  unix-char, unix-block, usb, gpu, infiniband, proxy, unix-hotplug, tpm, pci.

* `properties` - Map of key/value pairs of
  [device properties](https://linuxcontainers.org/incus/docs/main/reference/devices/).
