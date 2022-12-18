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

## Cluster Example

In order to create a network in a cluster, you first have to
define the network on each node in the cluster. Then you can create
the actual network:

```hcl
resource "lxd_network" "my_network_node1" {
  name = "my_network"
  target = "node1"
}

resource "lxd_network" "my_network_node2" {
  name = "my_network"
  target = "node2"
}

resource "lxd_network" "my_network" {
  depends_on = [
    "lxd_network.my_network_node1",
    "lxd_network.my_network_node2",
  ]

  name = "my_network"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat"     = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat"     = "true"
  }
}
```

Please see the [LXD Clustering documentation](https://lxd.readthedocs.io/en/latest/clustering/)
for more details on how to create a network in clustered mode.


## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `target` - *Optional* - Specify a target node in a cluster.

* `name` - *Required* - Name of the network. This is usually the device the
	network will appear as to containers.

* `type` - *Optional* - The type of network to create. Can be one of: bridge,
  macvlan, sriov, ovn, or physical. If no type is specified, usually a bridge
  is created.

* `config` - *Optional* - Map of key/value pairs of
	[network config settings](https://github.com/lxc/lxd/blob/master/doc/configuration.md).

* `project` - *Optional* - Name of the project where the network will be created.

## Attribute Reference

The following attributes are exported:

* `type` - The type of network. Can be one of: bridge, macvlan, sriov, ovn or
  physical.

* `managed` - Whether or not the network is managed.

## Importing

Networks can be imported by doing:

```shell
$ terraform import lxd_network.my_network <name of network>
```

> NOTE: Importing of clustered networks is not supported.

