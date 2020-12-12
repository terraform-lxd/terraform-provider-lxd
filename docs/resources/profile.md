# lxd_profile

Manages an LXD profile.

## Example Usage

```hcl
resource "lxd_profile" "profile1" {
  name = "profile1"

  config = {
    "limits.cpu" = 2
  }

  device {
    name = "shared"
    type = "disk"

    properties = {
      source = "/tmp"
      path   = "/tmp"
    }
  }

  device {
    type = "disk"
    name = "root"

    properties = {
      pool = "default"
      path = "/"
    }
  }
}

resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false
  profiles  = ["default", "${lxd_profile.profile1.name}"]
}
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `name` - *Required* - Name of the container.

* `config` - *Optional* - Map of key/value pairs of
	[container config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md).

* `device` - *Optional* - Device definition. See reference below.

The `device` block supports:

* `name` - *Required* - Name of the device.

* `type` - *Required* - Type of the device Must be one of none, disk, nic,
	unix-char, unix-block, usb, gpu.

* `properties`- *Required* - Map of key/value pairs of
	[device properties](https://github.com/lxc/lxd/blob/master/doc/configuration.md).

## Importing

Profiles can be imported by doing:

```shell
$ terraform import lxd_profile.my_profile <name of profile>
```

## Notes

* The order in which profiles are specified is important. LXD applies profiles
	from left to right. Profile options may be overridden by other profiles.
