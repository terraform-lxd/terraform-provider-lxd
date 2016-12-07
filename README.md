# terraform-provider-lxd

LXD Resource provider for Terraform

[![Build Status](https://travis-ci.org/sl1pm4t/terraform-provider-lxd.svg?branch=master)](https://travis-ci.org/sl1pm4t/terraform-provider-lxd)

## Prerequisites

* [Terraform](http://terraform.io)
* [LXD](https://linuxcontainers.org/lxd)

## Usage

This provider connects to the LXD daemon over local Unix socket or HTTPS.

It makes use of the [LXD client library](http://github.com/lxc/lxd), which currently looks in `~/.config/lxc/` for `client.crt` and `client.key` files that must exist to be able to communicate with the LXD daemon.

To generate these files and store them in the LXD daemon, follow these [steps](https://linuxcontainers.org/lxd/getting-started-cli/#multiple-hosts).

### Example Configuration

**Provider (HTTPS)**

```hcl
provider "lxd" {
  scheme  = "https"
  address = "10.1.1.8"
}
```

**Resource**

```hcl
resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"     # this assumes an image has been cached locally with the alias 'ubuntu'
                           # e.g.
                           # lxc image copy images:ubuntu/xenial/amd64 local: --alias=ubuntu
  profiles  = ["default"]
  ephemeral = false

  config {
    limits.cpu = 2         # this must be a valid container config setting or LXD will throw a
                           # "Bad key: foo" error. See reference below.
  }

  device {
    name = "shared"
    type = "disk"          # Type must be one of none, disk, nic, unix-char, unix-block, usb, gpu
    properties {           # Properties are valid key/value pairs of the device.
      source = "/tmp"      # See the LXD documentation for further information.
      path = "/tmp"
    }
  }
}
```

### Reference

#### Provider

##### Parameters

  * `address`  - *Optional* - Unix socket file path or IP / FQDN where LXD daemon can be reached. Defaults to `/var/lib/lxd/unix.socket`
  * `scheme`   - *Optional* - `https` or `unix`. Defaults to `unix`.
  * `port`     - *Optional* - `https` scheme only - The port on which the LXD daemon is listening. Defaults to 8443.
  * `remote`   - *Optional* - Name of the remote LXD as it exists in the local lxc config. Defaults to `local`.

#### Resource

##### Parameters

  * `name`      - *Required* -Name of the container.
  * `image`     - *Required* -Base image from which the container will be created.
  * `profiles`  - *Optional* -Array of LXD config profiles to apply to the new container.
  * `ephemeral` - *Optional* -Boolean indicating if this container is ephemeral. Default = false.
  * `privileged`- *Optional* -Boolean indicating if this container will run in privileged mode. Default = false.
  * `config`    - *Optional* -Map of key/value pairs of [container config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#container-configuration).
  * `device`    - *Optional* -Device definition. See reference below.

##### Device Block

  * `name`      - *Required* -Name of the device.
  * `type`      - *Required* -Type of the device Must be one of none, disk, nic, unix-char, unix-block, usb, gpu.
  * `properties`- *Required* -Map of key/value pairs of [https://github.com/lxc/lxd/blob/master/doc/configuration.md#devices-configuration](device properties).

## Known Limitations

All the base LXD images do not include an SSH server, therefore terraform will be unable to execute any `provisioners`.
A basic base image must be prepared in advance, that includes the SSH server.

## To Do

- [ ] Support for using client cert / key from other paths
- [ ] Ability to update container config
- [ ] Ability to exec commands via LXD WebSocket channel
- [ ] Ability to upload files via LXD WebSocket channel
- [ ] Volumes support
- [ ] Add LXD `profile` resource
- [ ] Add LXD `image` resource
