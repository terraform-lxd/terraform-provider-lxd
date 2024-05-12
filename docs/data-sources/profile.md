# lxd_profile

Provides information about an existing LXD profile.

## Example Usage

```hcl
data "lxd_profile" "default" {
  name = "default"
}

resource "lxd_instance" "inst" {
  name     = "my-instance"
  image    = "ubuntu:24.04"
  profiles = [data.lxd_profile.default.name]
}
```

## Argument Reference

* `name` - **Required** - Name of the profile.

* `project` - *Optional* - Name of the project where the profile is create.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote is used.

## Attribute Reference

* `device` - Device definition. See reference below.

* `config` - Map of key/value pairs of
	[instance config settings](https://documentation.ubuntu.com/lxd/en/latest/reference/instance_options/).

The `device` block supports:

* `name` - Name of the device.

* `type` - Type of the device.

* `properties`- Map of key/value pairs of
	[device properties](https://documentation.ubuntu.com/lxd/en/latest/reference/devices/).
