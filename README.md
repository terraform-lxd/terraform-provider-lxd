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

## Known Limitations

All the base LXD images do not include an SSH server, therefore terraform will be unable to execute any `provisioners`.
A basic base image must be prepared in advance, that includes the SSH server.

## To Do

- [ ] Support for container config map
- [ ] Support for using client cert / key from other paths
- [ ] Ability to update container profile/config
- [ ] Ability to exec commands via LXD WebSocket channel
- [ ] Ability to upload files via LXD WebSocket channel
- [ ] Ability to specify CPU / Memory limits
- [ ] Volumes support
- [ ] Add LXD `profile` resource
- [ ] Add LXD `image` resource
