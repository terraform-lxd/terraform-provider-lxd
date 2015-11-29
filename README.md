# terraform-provider-lxd

LXD Resource provider for Terraform

## Prerequisites

* [Terraform](http://terraform.io)
* [LXD](https://linuxcontainers.org/lxd)

### How to use

This provider connects to the LXD daemon over local Unix socket or HTTPS.

It makes use of the LXD client library [github.com/lxc/lxd], which currently looks in `~/.config/lxc/` for `client.crt`
and `client.key` files that must exist to be able to communicate with the LXD daemon over HTTPS.
To generate these files and store them in the LXD daemon, follow these [steps](https://linuxcontainers.org/lxd/getting-started-cli/#multiple-hosts).

### Example Configuration

Provider (HTTPS)

```
provider "lxd" {
  scheme  = "https"
  address = "192.168.1.8"
}
```

Resource

```
resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  profiles  = ["default"]
  ephemeral = false
}
```

### Misc

This has only been tested on Ubuntu 14.10 against LXD version 0.21

## To Do

- [ ] Add support for container config map
- [ ] Add support for using client cert / key from other paths
- [ ] Add ability to update container profile/config
- [ ] Add ability to exec commands via LXD WebSocket channel
- [ ] Add ability to upload files via LXD WebSocket channel
