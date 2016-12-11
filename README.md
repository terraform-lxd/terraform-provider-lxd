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
resource "lxd_network" "eth1" {
  name = "eth1"

  config {
    ipv4.address = "10.150.19.1/24"
    ipv4.nat = "true"
    ipv6.address = "fd42:474b:622d:259d::1/64"
    ipv6.nat = "true"
  }
}

resource "lxd_profile" "profile1" {
  name = "profile1"

  config {
    limits.cpu = 2
  }

  device {
    name = "eth1"
    type = "nic"
    properties {
      nictype = "bridged"
      parent = "${lxd_network.eth1.name}"
    }
  }
}

resource "lxd_container" "test1" {
  name      = "test1"

  # this assumes an image has been cached locally with the alias 'ubuntu'
  # e.g.
  # lxc image copy images:ubuntu/xenial/amd64 local: --alias=ubuntu
  image     = "ubuntu"

  profiles  = ["default", "${lxd_profile.profile1.name}"]
  ephemeral = false

  device {
    name = "shared"
    type = "disk"
    properties {
      source = "/tmp"
      path = "/tmp"
    }
  }
}

```

## Reference

### Provider

#### lxd

##### Parameters

  * `address`  - *Optional* - Unix socket file path or IP / FQDN where LXD daemon can be reached. Defaults to `/var/lib/lxd/unix.socket`
  * `scheme`   - *Optional* - `https` or `unix`. Defaults to `unix`.
  * `port`     - *Optional* - `https` scheme only - The port on which the LXD daemon is listening. Defaults to 8443.
  * `remote`   - *Optional* - Name of the remote LXD as it exists in the local lxc config. Defaults to `local`.

### Resources

The following resources currently exist:

  * `lxd_profile` - Creates and manages a Profile
  * `lxd_network` - Creates and manages a Network
  * `lxd_container` - Creates and manages a Container

#### lxd_profile

##### Parameters

  * `name`      - *Required* -Name of the container.
  * `config`    - *Optional* -Map of key/value pairs of [container config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#container-configuration).
  * `device`    - *Optional* -Device definition. See reference below.

##### Device Block

  * `name`      - *Required* -Name of the device.
  * `type`      - *Required* -Type of the device Must be one of none, disk, nic, unix-char, unix-block, usb, gpu.
  * `properties`- *Required* -Map of key/value pairs of [device properties](https://github.com/lxc/lxd/blob/master/doc/configuration.md#devices-configuration).

#### lxd_network

##### Parameters

  * `name`      - *Required* -Name of the network. This is usually the device the network will appear as to containers.
  * `config`    - *Optional* -Map of key/value pairs of [network config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md#network-configuration).

##### Exported Attributes

  * `type`      - The type of network. This will be either bridged or physical.
  * `managed`   - Whether or not the network is managed.

#### lxd_container

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
  * `properties`- *Required* -Map of key/value pairs of [device properties](https://github.com/lxc/lxd/blob/master/doc/configuration.md#devices-configuration).

## Known Limitations

All the base LXD images do not include an SSH server, therefore terraform will be unable to execute any `provisioners`.
A basic base image must be prepared in advance, that includes the SSH server.

## To Do

- [ ] Support for using client cert / key from other paths
- [ ] Ability to update container config
- [ ] Ability to exec commands via LXD WebSocket channel
- [ ] Ability to upload files via LXD WebSocket channel
- [ ] Volumes support
- [ ] Add LXD `image` resource
