# incus_instance

Manages an Incus instance that can be either a container or virtual machine.

An instance can take a number of configuration and device options. A full reference can be found [here](https://documentation.ubuntu.com/incus/en/latest/reference/instance_options/).

## Basic Example

```hcl
resource "incus_instance" "container1" {
  name  = "container1"
  image = "images:ubuntu/22.04"

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
resource "incus_storage_pool" "pool1" {
  name   = "mypool"
  driver = "zfs"
}

resource "incus_volume" "volume1" {
  name = "myvolume"
  pool = incus_storage_pool.pool1.name
}

resource "incus_instance" "container1" {
  name  = "%s"
  image = "ubuntu"

  device {
    name = "volume1"
    type = "disk"
    properties = {
      path   = "/mount/point/in/instance"
      source = incus_volume.volume1.name
      pool   = incus_storage_pool.pool1.name
    }
  }
}
```

## Example to proxy/forward ports

```hcl
resource "incus_instance" "container2" {
  name      = "container2"
  image     = "ubuntu"
  profiles  = ["default"]
  ephemeral = false

  device {
    name = "http"
    type = "proxy"
    properties = {
      # Listen on Incus host's TCP port 80
      listen = "tcp:0.0.0.0:80"
      # And connect to the instance's TCP port 80
      connect = "tcp:127.0.0.1:80"
    }
  }
}
```

## Argument Reference

- `name` - **Required** - Name of the instance.

- `image` - **Required** - Base image from which the instance will be created. Must
  specify [an image accessible from the provider remote](https://documentation.ubuntu.com/incus/en/latest/reference/remote_image_servers/).

- `description` - _Optional_ - Description of the instance.

- `type` - _Optional_ - Instance type. Can be `container`, or `virtual-machine`. Defaults to `container`.

- `ephemeral` - _Optional_ - Boolean indicating if this instance is ephemeral. Defaults to `false`.

- `running` - _Optional_ - Boolean indicating whether the instance should be started (running). Defaults to `true`.

- `wait_for_network` - _Optional_ - Boolean indicating if the provider should wait for the instance to get an IPv4 address before considering the instance as started.
  If `running` is set to false or instance is already running (on update), this value has no effect. Defaults to `true`.

- `profiles` - _Optional_ - List of Incus config profiles to apply to the new
  instance. Profile `default` will be applied if profiles are not set (are `null`).
  However, if an empty array (`[]`) is set as a value, no profiles will be applied.

- `device` - _Optional_ - Device definition. See reference below.

- `file` - _Optional_ - File to upload to the instance. See reference below.

- `limits` - _Optional_ - Map of key/value pairs that define the
  [instance resources limits](https://documentation.ubuntu.com/incus/en/latest/reference/instance_options/#resource-limits).

- `config` - _Optional_ - Map of key/value pairs of
  [instance config settings](https://documentation.ubuntu.com/incus/en/latest/reference/instance_options/).

- `project` - _Optional_ - Name of the project where the instance will be spawned.

- `remote` - _Optional_ - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

- `target` - _Optional_ - Specify a target node in a cluster.

The `device` block supports:

- `name` - **Required** - Name of the device.

- `type` - **Required** - Type of the device Must be one of none, disk, nic,
  unix-char, unix-block, usb, gpu, infiniband, proxy, unix-hotplug, tpm, pci.

- `properties`- **Required** - Map of key/value pairs of
  [device properties](https://documentation.ubuntu.com/incus/en/latest/reference/devices/).

The `file` block supports:

- `content` - _**Required** unless source_path is used_ - The _contents_ of the file.
  Use the `file()` function to read in the content of a file from disk.

- `source_path` - _**Required** unless content is used_ - The source path to a file to
  copy to the instance.

- `target_path` - **Required** - The absolute path of the file on the instance,
  including the filename.

- `uid` - _Optional_ - The UID of the file. Must be an unquoted integer.

- `gid` - _Optional_ - The GID of the file. Must be an unquoted integer.

- `mode` - _Optional_ - The octal permissions of the file, must be quoted. Defaults to `0755`.

- `create_directories` - _Optional_ - Whether to create the directories leading
  to the target if they do not exist.

## Attribute Reference

The following attributes are exported:

- `ipv4_address` - The IPv4 Address of the instance. See Instance Network
  Access for more details.

- `ipv6_address` - The IPv6 Address of the instance. See Instance Network
  Access for more details.

- `mac_address` - The MAC address of the detected NIC. See Instance Network
  Access for more details.

- `status` - The status of the instance.

## Instance Network Access

If your instance has multiple network interfaces, you can specify which one
Terraform should report the IP addresses of. If you do not specify an interface,
Terraform will use the _last_ address detected. Global IPv6 address will be favored if present.

To specify an interface, do the following:

```hcl
resource "incus_instance" "instance1" {
  name = "instance1"
  image = "images:alpine/3.5/amd64"
  profiles = ["default"]

  config = {
    "user.access_interface" = "eth0"
  }
}
```

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>[,image=<image>]`

- `<remote>` - _Optional_ - Remote name.
- `<project>` - _Optional_ - Project name.
- `<name>` - **Required** - Instance name.
- `image=<image>` - _Optional_ - The image used by the instance.

~> **Warning:** Importing the instance without specifying `image` will lead to its replacement
upon the next apply, rather than an in-place update.

### Import example

Example using terraform import command:

```shell
$ terraform import incus_instance.myinst proj/c1,image=images:alpine/3.18/amd64
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "incus_instance" "myinst" {
  name    = "c1"
  project = "proj"
  image   = "images:alpine/3.18/amd64"
}

import {
  to = incus_instance.myinst
  id = "proj/c1,image=images:alpine/3.18/amd64"
}
```

## Notes

- The instance resource `config` includes some keys that can be automatically generated by the Incus.
  If these keys are not explicitly defined by the user, they will be omitted from the Terraform
  state and treated as computed values.
  - `image.*`
  - `volatile.*`
