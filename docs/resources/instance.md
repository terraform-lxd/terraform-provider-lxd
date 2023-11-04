# lxd_instance

Manages an LXD instance that can be either a container or virtual machine.

An instance can take a number of configuration and device options. A full reference can be found [here](https://documentation.ubuntu.com/lxd/en/latest/reference/instance_options/).

## Basic Example

```hcl
resource "lxd_instance" "container1" {
  name      = "container1"
  image     = "images:ubuntu/22.04"
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

resource "lxd_instance" "container1" {
  name = "%s"
  image = "ubuntu"
  profiles = ["default"]

  device {
    name = "volume1"
    type = "disk"
    properties = {
      path = "/mount/point/in/instance"
      source = "${lxd_volume.volume1.name}"
      pool = "${lxd_storage_pool.pool1.name}"
    }
  }
}
```

## Example to proxy/forward ports

```hcl
resource "lxd_instance" "container2" {
  name = "container2"
  image = "ubuntu"
  profiles = ["default"]
  ephemeral = false

  device {
    name = "http"
    type = "proxy"
    properties = {
      # Listen on LXD host's TCP port 80
      listen = "tcp:0.0.0.0:80"
      # And connect to the instance's TCP port 80
      connect = "tcp:127.0.0.1:80"
    }
  }
}
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `name` - *Required* - Name of the instance.

* `image` - *Required* - Base image from which the instance will be created. Must
*       specify [an image accessible from the provider remote](https://documentation.ubuntu.com/lxd/en/latest/reference/remote_image_servers/).

* `type` - *Optional* -  Instance type. Can be `container`, or `virtual-machine`. Defaults to `container`.

* `profiles` - *Optional* - List of LXD config profiles to apply to the new
	instance.

* `ephemeral` - *Optional* - Boolean indicating if this instance is ephemeral.
	Valid values are `true` and `false`. Defaults to `false`.

* `config` - *Optional* - Map of key/value pairs of
	[instance config settings](https://documentation.ubuntu.com/lxd/en/latest/reference/instance_options/).

* `limits` - *Optional* - Map of key/value pairs that define the
	[instance resources limits](https://documentation.ubuntu.com/lxd/en/latest/reference/instance_options/#resource-limits).

* `device` - *Optional* - Device definition. See reference below.

* `file` - *Optional* - File to upload to the instance. See reference below.

* `wait_for_network` - *Optional* - Boolean indicating if the provider should wait for the instance's network address to become available during creation.
  Valid values are `true` and `false`. Defaults to `true`.

* `start_on_create` - *Optional* - Boolean indicating if the provider should start the instance during creation. It will not re-start on update runs.
  Valid values are `true` and `false`. Defaults to `true`.

* `target` - *Optional* - Specify a target node in a cluster.

* `project` - *Optional* - Name of the project where the instance will be spawned.

The `device` block supports:

* `name` - *Required* - Name of the device.

* `type` - *Required* - Type of the device Must be one of none, disk, nic,
	unix-char, unix-block, usb, gpu, infiniband, proxy, tpm.

* `properties`- *Required* - Map of key/value pairs of
	[device properties](https://documentation.ubuntu.com/lxd/en/latest/reference/devices/).

The `file` block supports:

* `content` - *Required unless source is used* - The _contents_ of the file.
	Use the `file()` function to read in the content of a file from disk.

* `source` - *Required unless content is used* The source path to a file to
	copy to the instance.

* `target_file` - *Required* - The absolute path of the file on the instance,
	including the filename.

* `uid` - *Optional* - The UID of the file. Must be an unquoted integer.

* `gid` - *Optional* - The GID of the file. Must be an unquoted integer.

* `mode` - *Optional* - The octal permissions of the file, must be quoted.

* `create_directories` - *Optional* - Whether to create the directories leading
	to the target if they do not exist.

## Attribute Reference

The following attributes are exported:

* `ip_address` - The IPv4 Address of the instance. See Instance Network Access
  for more details.

* `ipv4_address` - The IPv4 Address of the instance. See Instance Network
  Access for more details.

* `ipv6_address` - The IPv6 Address of the instance. See Instance Network
  Access for more details.

* `mac_address` - The MAC address of the detected NIC. See Instance Network
  Access for more details.

* `status` - The status of the instance.

## Instance Network Access

If your instance has multiple network interfaces, you can specify which one
Terraform should report the IP addresses of. If you do not specify an interface,
Terraform will use the _last_ address detected. Global IPv6 address will be favored if present.

To specify an interface, do the following:

```hcl
resource "lxd_instance" "instance1" {
  name = "instance1"
  image = "images:alpine/3.5/amd64"
  profiles = ["default"]

  config = {
    "user.access_interface" = "eth0"
  }
}
```

## Importing

Existing instances can be imported with the following syntax:

```shell
$ terraform import lxd_instance.my_instance [remote:]name[/images:alpine/3.5]
```

Where:

* remote - *Optional* - The name of the remote to import the instance from.
* name - *Required* - The name of the instance.
* /images:alpine/3.5 - *Optional* - Translates to `image = images:alpine/3.5`
  in the resource configuration.

## Notes

* The `config` attributes cannot be changed without destroying and re-creating
	the instance. However, values in `limits` can be changed on the fly.
