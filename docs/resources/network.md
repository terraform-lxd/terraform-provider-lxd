# lxd_network

Manages an LXD network.

You must be using LXD 2.3 or later. See
[this](https://www.stgraber.org/2016/10/27/network-management-with-lxd-2-3/)
blog post for details about LXD networking and the
[configuration reference](https://github.com/lxc/lxd/blob/master/doc/configuration.md)
for all network details.

## Example Usage

```hcl
resource "lxd_network" "new_default" {
  name = "new_default"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat"     = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat"     = "true"
  }
}

resource "lxd_profile" "profile1" {
  name = "profile1"

  device {
    name = "eth0"
    type = "nic"

    properties = {
      nictype = "bridged"
      parent  = "${lxd_network.new_default.name}"
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
  profiles  = ["${lxd_profile.profile1.name}"]
}
```

## Multiple Network Example

This example uses the "default" LXD nework on `eth0` (unspecified) and a
custom network on `eth1`

```hcl
resource "lxd_network" "internal" {
  name = "internal"

  config = {
    "ipv4.address" = "192.168.255.1/24"
  }
}

resource "lxd_profile" "profile1" {
  name = "profile1"

  device {
    name = "eth1"
    type = "nic"

    properties = {
      nictype = "bridged"
      parent  = "${lxd_network.internal.name}"
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

  provisioner "local-exec" {
    command = "lxc exec local:${self.name} dhclient eth1"
  }
}
```

## Tunnel Example

Tunnel "server":

```hcl
resource "lxd_network" "vxtun" {
  name = "vxtun"

  config = {
    "tunnel.vxtun.protocol" = "vxlan"
    "tunnel.vxtun.id"       = 9999
    "tunnel.vxtun.local"    = "10.1.1.1"
    "tunnel.vxtun.remote"   = "10.255.1.1"
    "ipv4.address"          = "192.168.255.1/24"
    "ipv6.address"          = "none"
  }
}
```

Tunnel "client":

```hcl
resource "lxd_network" "vxtun" {
  name = "vxtun"

  config = {
    "tunnel.vxtun.protocol" = "vxlan"
    "tunnel.vxtun.id"       = 9999
    "tunnel.vxtun.local"    = "10.255.1.1"
    "tunnel.vxtun.remote"   = "10.1.1.1"
    "ipv4.address"          = "none"
    "ipv6.address"          = "none"
  }
}
```

Note how the `local` and `remote` addresses are swapped between the two.
Also note how the client does not provide an IP address range.

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `name` - *Required* - Name of the network. This is usually the device the
	network will appear as to containers.

* `config` - *Optional* - Map of key/value pairs of
	[network config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md).

## Attribute Reference

The following attributes are exported:

* `type` - The type of network. This will be either bridged or physical.

* `managed` - Whether or not the network is managed.
