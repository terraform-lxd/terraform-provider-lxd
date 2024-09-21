# incus_network

Manages an Incus network.

See Incus networking [configuration reference](https://linuxcontainers.org/incus/docs/main/explanation/networks/)
for all network details.

## Example Usage

```hcl
resource "incus_network" "new_default" {
  name = "new_default"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat"     = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat"     = "true"
  }
}

resource "incus_profile" "profile1" {
  name = "profile1"

  device {
    name = "eth0"
    type = "nic"

    properties = {
      nictype = "bridged"
      parent  = "${incus_network.new_default.name}"
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

resource "incus_instance" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false
  profiles  = ["${incus_profile.profile1.name}"]
}
```

## Multiple Network Example

This example uses the "default" Incus network on `eth0` (unspecified) and a
custom network on `eth1`

```hcl
resource "incus_network" "internal" {
  name = "internal"

  config = {
    "ipv4.address" = "192.168.255.1/24"
  }
}

resource "incus_profile" "profile1" {
  name = "profile1"

  device {
    name = "eth1"
    type = "nic"

    properties = {
      nictype = "bridged"
      parent  = "${incus_network.internal.name}"
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

resource "incus_instance" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false
  profiles  = ["default", "${incus_profile.profile1.name}"]

  provisioner "local-exec" {
    command = "lxc exec local:${self.name} dhclient eth1"
  }
}
```

## Tunnel Example

Tunnel "server":

```hcl
resource "incus_network" "vxtun" {
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
resource "incus_network" "vxtun" {
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
resource "incus_network" "my_network_node1" {
  name   = "my_network"
  target = "node1"
}

resource "incus_network" "my_network_node2" {
  name   = "my_network"
  target = "node2"
}

resource "incus_network" "my_network" {
  depends_on = [
    "incus_network.my_network_node1",
    "incus_network.my_network_node2",
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

Please see the [Incus Clustering documentation](https://linuxcontainers.org/incus/docs/main/howto/cluster_config_networks/)
for more details on how to create a network in clustered mode.


## Argument Reference

* `name` - **Required** - Name of the network. This is usually the device the
	network will appear as to instances.

* `description` - *Optional* - Description of the network.

* `type` - *Optional* - The type of network to create. Can be one of: bridge,
  macvlan, sriov, ovn, or physical. If no type is specified, a bridge network
  is created.

* `config` - *Optional* - Map of key/value pairs of
	[network config settings](https://linuxcontainers.org/incus/docs/main/networks/).

* `project` - *Optional* - Name of the project where the network will be created.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.

## Attribute Reference

The following attributes are exported:

* `type` - The type of network. Can be one of: bridge, macvlan, sriov, ovn or
  physical.

* `managed` - Whether or not the network is managed.

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<name>` - **Required** - Network name.

-> Clustered networks cannot be imported.

### Import example

Example using terraform import command:

```shell
terraform import incus_network.mynet proj/net1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "incus_network" "mynet" {
  name    = "net1"
  project = "proj"
}

import {
  to = incus_network.mynet
  id = "proj/net1"
}
```

## Notes

* The network resource `config` includes some keys that can be automatically generated by the Incus.
  If these keys are not explicitly defined by the user, they will be omitted from the Terraform
  state and treated as computed values.
    - `bridge.mtu`
    - `ipv4.nat`
    - `ipv4.address`
    - `ipv6.nat`
    - `ipv6.address`
    - `volatile.*`
