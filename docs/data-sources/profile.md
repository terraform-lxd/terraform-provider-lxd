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

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

* `device` - *Optional* - Device definition. See reference below.

* `config` - *Optional* - Map of key/value pairs of
  [instance config settings](https://linuxcontainers.org/incus/docs/main/reference/instance_options/).

The `device` block supports:

* `name` - **Required** - Name of the device.

* `type` - **Required** - Type of the device Must be one of none, disk, nic,
  unix-char, unix-block, usb, gpu, infiniband, proxy, unix-hotplug, tpm, pci.

* `properties`- **Required** - Map of key/value pairs of
  [device properties](https://linuxcontainers.org/incus/docs/main/reference/devices/).
