# lxd_instance

Manages an LXD instance that can be either a container or virtual machine.

An instance can take a number of configuration and device options. A full reference can be found [here](https://documentation.ubuntu.com/lxd/en/latest/reference/instance_options/).

## Basic Example

```hcl
resource "lxd_instance" "container1" {
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
resource "lxd_storage_pool" "pool1" {
  name   = "mypool"
  driver = "zfs"
}

resource "lxd_volume" "volume1" {
  name = "myvolume"
  pool = lxd_storage_pool.pool1.name
}

resource "lxd_instance" "container1" {
  name  = "%s"
  image = "ubuntu"

  device {
    name = "volume1"
    type = "disk"
    properties = {
      path   = "/mount/point/in/instance"
      source = lxd_volume.volume1.name
      pool   = lxd_storage_pool.pool1.name
    }
  }
}
```

## Example to proxy/forward ports

```hcl
resource "lxd_instance" "container2" {
  name      = "container2"
  image     = "ubuntu"
  profiles  = ["default"]
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

* `name` - **Required** - Name of the instance.

* `image` - **Required** - Base image from which the instance will be created. Must
  specify [an image accessible from the provider remote](https://documentation.ubuntu.com/lxd/en/latest/reference/remote_image_servers/).

* `description` - *Optional* - Description of the instance.

* `type` - *Optional* -  Instance type. Can be `container`, or `virtual-machine`. Defaults to `container`.

* `ephemeral` - *Optional* - Boolean indicating if this instance is ephemeral. Defaults to `false`.

* `running` - *Optional* - Boolean indicating whether the instance should be started (running). Defaults to `true`.

* `wait_for_network` - *Optional* - Boolean indicating if the provider should wait for the instance to get an IPv4 address before considering the instance as started.
  If `running` is set to false or instance is already running (on update), this value has no effect. Defaults to `true`.

* `profiles` - *Optional* - List of LXD config profiles to apply to the new
	instance. Profile `default` will be applied if profiles are not set (are `null`).
  However, if an empty array (`[]`) is set as a value, no profiles will be applied.

* `device` - *Optional* - Device definition. See reference below.

* `file` - *Optional* - File to upload to the instance. See reference below.

* `limits` - *Optional* - Map of key/value pairs that define the
	[instance resources limits](https://documentation.ubuntu.com/lxd/en/latest/reference/instance_options/#resource-limits).

* `config` - *Optional* - Map of key/value pairs of
	[instance config settings](https://documentation.ubuntu.com/lxd/en/latest/reference/instance_options/).

* `project` - *Optional* - Name of the project where the instance will be spawned.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

The `device` block supports:

* `name` - **Required** - Name of the device.

* `type` - **Required** - Type of the device Must be one of none, disk, nic,
	unix-char, unix-block, usb, gpu, infiniband, proxy, unix-hotplug, tpm, pci.

* `properties`- **Required** - Map of key/value pairs of
	[device properties](https://documentation.ubuntu.com/lxd/en/latest/reference/devices/).

The `file` block supports:

* `content` - *__Required__ unless source_path is used* - The _contents_ of the file.
	Use the `file()` function to read in the content of a file from disk.

* `source_path` - *__Required__ unless content is used* - The source path to a file to
	copy to the instance.

* `target_path` - **Required** - The absolute path of the file on the instance,
	including the filename.

* `uid` - *Optional* - The UID of the file. Must be an unquoted integer.

* `gid` - *Optional* - The GID of the file. Must be an unquoted integer.

* `mode` - *Optional* - The octal permissions of the file, must be quoted. Defaults to `0755`.

* `create_directories` - *Optional* - Whether to create the directories leading
	to the target if they do not exist.

The `exec` block supports:

* `command` - **Required** - The command to be executed and its arguments, if any (list of strings).

* `triggers` - *Optional* - A list of arbitrary strings that, when changed, will force the command
  to be rerun.

* `environment` - *Optional* - Map of additional environment variables.
  (Variables `PATH`, `LANG`, `HOME`, and `USER` are set by default, unless passed by the user.)

* `working_dir` - *Optional* - The directory in which the command should run.

* `record_output` - *Optional* - When set to true, `stdout` and `stderr` attributes will be
  populated (exported). Defaults to `false`.

* `fail_on_error` - *Optional* - Boolean indicating whether resource provisioning should stop upon
  encountering an error during command execution. Defaults to `false`.

* `uid` - *Optional* - The user ID for running command. Defaults to `0` (root).

* `gid` - *Optional* - The group ID for running command. Defaults to `0` (root).

## Attribute Reference

The following attributes are exported:

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
  name  = "c1"
  image = "images:alpine/3.18/amd64"

  config = {
    "user.access_interface" = "eth0"
  }
}
```

## Executing Commands in Instances

The `exec` block in an LXD instance configuration is used to execute commands. You can specify
multiple exec blocks, with each block requiring a command defined as a list of strings.

### Simple Commands

For simple and short commands, you can specify the entire command as a single string in the list.
In this case, if the list contains only one string, the provider automatically splits it into
multiple arguments based on spaces.

```hcl
resource "lxd_instance" "inst" {
  name  = "c1"
  image = "images:alpine/3.18/amd64"

  exec {
    command = ["ls -lah"]
  }
}
```

### Complex Commands and Environment Access

For more complex commands or when access to the environment is required, use the `<shell> -c` syntax.

```hcl
resource "lxd_instance" "inst" {
  name  = "c1"
  image = "images:alpine/3.18/amd64"

  exec {
    command = ["sh", "-c", "echo $ENV_KEY"]

    environment = {
      "ENV_KEY" = "ENV_VALUE"
    }
  }

  exec {
    command     = ["sh", "-c", "cat os-release | tr -d '\n'"]
    working_dir = "/etc"
  }
}
```

### Capturing Command Output

Exit status of the command will be always available after command execution via `exit_code` attribute.
However, to capture and access a command's output, set `record_output` to true. The command's standard
output and standard error will then be accessible through the exported attributes `stdout` and `stderr`,
respectively.

```hcl
resource "lxd_instance" "inst" {
  name  = "c1"
  image = "images:alpine/3.18/amd64"

  exec {
    command       = ["uname"]
    record_output = true
  }
}

output "exec-output" {
  value = {
    "code" = lxd_instance.inst.exec[0].exit_code # 0
    "out"  = lxd_instance.inst.exec[0].stdout    # "Linux\n"
    "err"  = lxd_instance.inst.exec[0].stderr    # ""
  }
}
```

### Fail on Command Error

By default, command failure is ignored. If you want to stop Terraform from provisioning the resources
if command exits with a non 0 status, set `fail_on_error` attribute to true.

```hcl
resource "lxd_instance" "inst" {
  name      = "c1"
  image     = "images:alpine/3.18/amd64"

  exec {
    command       = ["invalid-cmd"]
    fail_on_error = true
  }
}
```

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>[,image=<image>]`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<name>` - **Required** - Instance name.
* `image=<image>` - *Optional* - The image used by the instance.

~> **Warning:** Importing the instance without specifying `image` will lead to its replacement
   upon the next apply, rather than an in-place update.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_instance.myinst proj/c1,image=images:alpine/3.18/amd64
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_instance" "myinst" {
  name    = "c1"
  project = "proj"
  image   = "images:alpine/3.18/amd64"
}

import {
  to = lxd_instance.myinst
  id = "proj/c1,image=images:alpine/3.18/amd64"
}
```

## Notes

* The instance resource `config` includes some keys that can be automatically generated by the LXD.
  If these keys are not explicitly defined by the user, they will be omitted from the Terraform
  state and treated as computed values.
    - `image.*`
    - `volatile.*`
