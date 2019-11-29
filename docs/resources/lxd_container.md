# lxd_container

Manages an LXD container.

A container can take a number of configuration and device options. A full reference can be found [here](https://github.com/lxc/lxd/blob/master/doc/configuration.md).

## Basic Example

```hcl
resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false

  config = {
    "boot.autostart" = true
  }

  limits = {
    cpu = 2
  }
}
```

## Example to Attach a Volume

```hcl
resource "lxd_storage_pool" "pool1" {
  name = "mypool"
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
  name = "%s"
  image = "ubuntu"
  profiles = ["default"]

  device {
    name = "volume1"
    type = "disk"
    properties = {
      path = "/mount/point/in/container"
      source = "${lxd_volume.volume1.name}"
      pool = "${lxd_storage_pool.pool1.name}"
    }
  }
}
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `name` - *Required* - Name of the container.

* `image` - *Required* - Base image from which the container will be created.

* `profiles` - *Optional* - List of LXD config profiles to apply to the new
	container.

* `ephemeral` - *Optional* - Boolean indicating if this container is ephemeral.
	Valid values are `true` and `false`. Defaults to `false`.

* `config` - *Optional* - Map of key/value pairs of
	[container config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#container-configuration).

* `limits` - *Optional* - Map of key/value pairs that define the
	[container resources limits](https://github.com/lxc/lxd/blob/master/doc/containers.md).

* `device` - *Optional* - Device definition. See reference below.

* `file` - *Optional* - File to upload to the container. See reference below.

* `wait_for_network` - *Optional* - Boolean indicating if the provider should wait for the container's network address to become available during creation.
  Valid values are `true` and `false`. Defaults to `true`.

The `device` block supports:

* `name` - *Required* - Name of the device.

* `type` - *Required* - Type of the device Must be one of none, disk, nic,
	unix-char, unix-block, usb, gpu.

* `properties`- *Required* - Map of key/value pairs of
	[device properties](https://github.com/lxc/lxd/blob/master/doc/configuration.md#devices-configuration).

The `file` block supports:

* `content` - *Required unless source is used* - The _contents_ of the file.
	Use the `file()` function to read in the content of a file from disk.

* `source` - *Required unless content is used* The source path to a file to
	copy to the container.

* `target_file` - *Required* - The absolute path of the file on the container,
	including the filename.

* `uid` - *Optional* - The UID of the file. Must be an unquoted integer.

* `gid` - *Optional* - The GID of the file. Must be an unquoted integer.

* `mode` - *Optional* - The octal permissions of the file, must be quoted.

* `create_directories` - *Optional* - Whether to create the directories leading
	to the target if they do not exist.

## Attribute Reference

The following attributes are exported:

* `ip_address` - The IPv4 Address of the container. See Container Network Access
  for more details.

* `ipv4_address` - The IPv4 Address of the container. See Container Network
  Access for more details.

* `ipv6_address` - The IPv6 Address of the container. See Container Network
  Access for more details.

* `mac_address` - The MAC address of the detected NIC. See Container Network
  Access for more details.

* `status` - The status of the container.

## Container Network Access

If your container has multiple network interfaces, you can specify which one
Terraform should report the IP addresses of. If you do not specify an interface,
Terraform will use the _last_ address detected. Global IPv6 address will be favored if present.

To specify an interface, do the following:

```hcl
resource "lxd_container" "container1" {
  name = "container1"
  image = "images:alpine/3.5/amd64"
  profiles = ["default"]

  config = {
    "user.access_interface" = "eth0"
  }
}
```

## Importing

Existing containers can be imported with the following syntax:

```shell
$ terraform import lxd_container.my_container [remote:]name[/images:alpine/3.5]
```

Where:

* remote - *Optional* - The name of the remote to import the container from.
* name - *Required* - The name of the container.
* /images:alpine/3.5 - *Optional* - Translates to `image = images:alpine/3.5`
  in the resource configuration.

## Notes

* The `config` attributes cannot be changed without destroying and re-creating
	the container. However, values in `limits` can be changed on the fly.
