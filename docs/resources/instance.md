# lxd_instance

Manages an LXD instance that can be either a container or virtual machine.

An instance can take a number of configuration and device options. A full reference can be found [here](https://documentation.ubuntu.com/lxd/latest/reference/instance_options/).

## Basic Example

```hcl
resource "lxd_instance" "instance1" {
  name  = "instance1"
  image = "ubuntu-daily:22.04"

  config = {
    "boot.autostart" = true
  }

  limits = {
    cpu = 2
  }
}
```

## Example to proxy/forward ports

```hcl
resource "lxd_instance" "instance2" {
  name      = "instance2"
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

* `image` - **Optional** - Base image from which the instance will be created. **For containers** it must
  specify [an image accessible from the provider remote](https://documentation.ubuntu.com/lxd/latest/reference/remote_image_servers/). If omitted, it is equivalent to the `--empty` CLI flag and creates an empty virtual machine.

* `description` - *Optional* - Description of the instance.

* `type` - *Optional* -  Instance type. Can be `container`, or `virtual-machine`. Defaults to `container`.

* `ephemeral` - *Optional* - Boolean indicating if this instance is ephemeral. Defaults to `false`.

* `running` - *Optional* - Boolean indicating whether the instance should be started (running). Defaults to `true`.

* `wait_for_network` - *Optional* - Boolean indicating if the provider should wait for the instance to get an IPv4 address before considering the instance as started.
  If `running` is set to false or instance is already running (on update), this value has no effect. Defaults to `true`.

* `allow_restart` - *Optional* - Allow instance to be stopped and restarted if required by the provider for operations like migration or renaming.

* `profiles` - *Optional* - List of LXD config profiles to apply to the new
	instance. Profile `default` will be applied if profiles are not set (are `null`).
  However, if an empty array (`[]`) is set as a value, no profiles will be applied.

* `device` - *Optional* - Device definition. See reference below.

* `file` - *Optional* - File to upload to the instance. See reference below.

* `execs` - *Optional* - Map of exec commands to run within the instance. See reference below.

* `limits` - *Optional* - Map of key/value pairs that define the
	[instance resources limits](https://documentation.ubuntu.com/lxd/latest/reference/instance_options/#resource-limits).

* `config` - *Optional* - Map of key/value pairs of
	[instance config settings](https://documentation.ubuntu.com/lxd/latest/reference/instance_options/).

* `project` - *Optional* - Name of the project where the instance will be spawned.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target cluster member or cluster member group.

The `device` block supports:

* `name` - **Required** - Name of the device.

* `type` - **Required** - Type of the device Must be one of none, disk, nic,
	unix-char, unix-block, usb, gpu, infiniband, proxy, unix-hotplug, tpm, pci.

* `properties`- **Required** - Map of key/value pairs of
	[device properties](https://documentation.ubuntu.com/lxd/latest/reference/devices/).

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

The `execs` map elements support the following attributes:

* `command` - **Required** - The command to be executed and its arguments, if any (list of strings).

* `enabled` - *Optional* - Boolean indicating whether the command should be executed.
  Defaults to `true`.

* `trigger` - *Optional* - Determines when the command should be executed.
  Available values are:
  + `on_change` (default) - Executes the command whenever the instance resource changes.
  + `on_start` - Executes the command on instance start. Note that the command will **not** be executed
    if the instance is started outside of Terraform.
  + `once` - Executes the command only once.

* `environment` - *Optional* - Map of additional environment variables.
  (Variables `PATH`, `LANG`, `HOME`, and `USER` are set by default, unless passed by the user.)

* `working_dir` - *Optional* - The directory in which the command should run.

* `record_output` - *Optional* - When set to true, `stdout` and `stderr` attributes will be
  populated (exported). Defaults to `false`.

* `fail_on_error` - *Optional* - Boolean indicating whether resource provisioning should stop upon
  encountering an error during command execution. Defaults to `false`.

* `uid` - *Optional* - The user ID for running command. Defaults to `0` (root).

* `gid` - *Optional* - The group ID for running command. Defaults to `0` (root).

-> **Note:** Command will be executed only when it is enabled, trigger condition is met,
  and instance is running (or started).

## Attribute Reference

The following attributes are exported:

* `ipv4_address` - The instance's global IPv4 address. See Instance Network
  Access for more details.

* `ipv6_address` - The instance's global IPv6 address. See Instance Network
  Access for more details.

* `mac_address` - The MAC address of the network interface. If MAC address is not available, the instance has no global IP address. See Instance Network
  Access for more details.

* `interfaces` - Map of all instance network interfaces (excluding loopback device). The map key represents the name of the network device (from LXD configuration).

* `location` - Name of the cluster member where instance is located.

* `status` - The status of the instance.

## Timeouts

Configuration options:
* `read` - Default `5m`
* `create` - Default `5m`
* `update` - Default `5m`
* `delete` - Default `5m`

If you need to set custom timeout durations for any of these operations,
you can specify them in your Terraform configuration as shown in the following example:
```hcl
resource "lxd_instance" "inst" {
  name  = "c1"
  image = "ubuntu-daily:22.04"

  timeouts = {
    read   = "10m"
    create = "10m"
    update = "10m"
    delete = "10m"
  }
}
```

## Instance Network Access

If your instance has multiple network interfaces, you can specify which one
Terraform should report the IP addresses of. If you do not specify an interface,
Terraform will use the _last_ address detected. Global IPv6 address will be favored if present.

To specify an interface, do the following:

```hcl
resource "lxd_instance" "instance1" {
  name  = "c1"
  image = "ubuntu-daily:22.04"

  config = {
    "user.access_interface" = "eth0"
  }
}
```

## Executing Commands in Instances

The `execs` map defines commands to be executed within an instance. Each element in the map
represents an exec command, uniquely identified by its map key. **The commands
are executed in alphabetical order of their map keys.**

### Commands and Environment Access

The command should be specified as a list of strings, where the first string is the path to
an executable, followed by its arguments.

```hcl
resource "lxd_instance" "inst" {
  name  = "c1"
  image = "ubuntu-daily:22.04"

  execs = {
    "00-run-first" = {
      # wait for boot to complete
      command = ["systemctl", "is-system-running", "--wait", "--quiet"]
      trigger = "on_start"
    }
    "simple-cmd" = {
      command = ["ls", "/"]
    }
    "zz-run-last" = {
      # show how long it took to boot
      command = ["systemd-analyze", "--no-pager"]
      trigger = "on_start"
    }
  }
}
```

By default, the command executes directly without a shell, meaning shell patterns like variables
and file redirects are not interpreted. To use environment variables or shell meta-characters
(e.g., pipes and file redirects), you must specify a shell executable.

```hcl
resource "lxd_instance" "inst" {
  name  = "c1"
  image = "ubuntu-daily:22.04"

  execs = {
    "shell_cmd" = {
      command = ["/bin/sh", "-c", "echo $ENV_KEY > env.txt"]

      environment = {
        "ENV_KEY" = "ENV_VALUE"
      }
    }
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
  image = "ubuntu-daily:22.04"

  execs = {
    "cmd" = {
      command       = ["hostname"]
      record_output = true
    }
  }
}

output "exec-output" {
  value = {
    "code" = lxd_instance.inst.execs["cmd"].exit_code # 0
    "out"  = lxd_instance.inst.execs["cmd"].stdout    # "c1\n"
    "err"  = lxd_instance.inst.execs["cmd"].stderr    # ""
  }
}
```

### Fail on Command Error

By default, command failure is ignored. To stop Terraform from provisioning the resources
if command exits with a non-zero exit code, set `fail_on_error` attribute to true.

```hcl
resource "lxd_instance" "inst" {
  name      = "c1"
  image     = "ubuntu-daily:22.04"

  execs = {
    "err_1" = {
      command       = ["false"]
      fail_on_error = false
    }

    "err_2" = {
      command       = ["invalid-cmd"]
      fail_on_error = true
    }
  }
}
```

In the above example, the provisioning process will continue despite the failure of the first
command (`err_1`). However, it will halt at the second command (`err_2`) because `fail_on_error`
is set to `true`.

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
$ terraform import lxd_instance.myinst proj/c1,image=ubuntu-daily:22.04
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_instance" "myinst" {
  name    = "c1"
  project = "proj"
  image   = "ubuntu-daily:22.04"
}

import {
  to = lxd_instance.myinst
  id = "proj/c1,image=ubuntu-daily:22.04"
}
```

## Notes

* The instance resource `config` includes some keys that can be automatically generated by the LXD.
  If these keys are not explicitly defined by the user, they will be omitted from the Terraform
  state and treated as computed values.
    - `image.*`
    - `volatile.*`

* Terraform LXD provider sets `user.managed-by` key to all managed instance devices.
  Removing that key from a device manually, would result in Terraform removing it on next apply.

