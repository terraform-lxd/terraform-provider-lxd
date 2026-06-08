# lxd_network

Manages an LXD network.

Refer to the [configuration reference](https://documentation.ubuntu.com/lxd/latest/explanation/networks/) for all network details.

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

resource "lxd_instance" "test1" {
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

resource "lxd_instance" "test1" {
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

A single `lxd_network` resource is enough to create a network across all cluster members.

```hcl
resource "lxd_network" "my_network" {
  name = "my_network"
  type = "bridge"
}
```

For clustered networks, per-member local config keys (such as `bridge.external_interfaces` or
`parent`) are extracted from `config` and applied across all cluster members. However, custom
per-member configuration can be set using `member_overrides`.

```hcl
resource "lxd_network" "my_network" {
  name = "my_network"
  type = "bridge"

  config = {
    "ipv4.address"               = "10.150.19.1/24"
    "ipv4.nat"                   = "true"
    "bridge.external_interfaces" = "eth0"
  }

  member_overrides = {
    "member-1" = {
      config = {
        "bridge.external_interfaces" = "eth1"
      }
    }

    "member-2" = {
      config = {
        "bridge.external_interfaces" = "eth2"
      }
    }
  }
}
```

Please see the [LXD Clustering documentation](https://documentation.ubuntu.com/lxd/latest/howto/cluster_config_networks/)
for more details on how to create a network in clustered mode.


## Argument Reference

* `name` - **Required** - Name of the network. This is usually the device the
	network will appear as to instances.

* `description` - *Optional* - Description of the network.

* `type` - *Optional* - The type of network to create. Can be one of: bridge,
  macvlan, sriov, ovn, or physical. If no type is specified, a bridge network
  is created.

* `config` - *Optional* - Map of key/value pairs of
	[network config settings](https://documentation.ubuntu.com/lxd/latest/networks/).

* `member_overrides` - *Optional* - Map of per-member local config overrides for clustered networks.
  Each key is a cluster member name.
  Each value is an object with a config map of local-scoped keys to apply for that member.
  Values in `member_overrides` take precedence over values from `config`.

* `members` - *Computed* - Map of resolved local config for every cluster member, populated
  after apply. Used by the provider to detect out-of-band changes (drift) on individual cluster members.

* `project` - *Optional* - Name of the project where the network will be created.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

* `type` - The type of network. Can be one of: bridge, macvlan, sriov, ovn or
  physical.

* `managed` - Whether or not the network is managed.

* `ipv4_address` - The network's global IPv4 address in CIDR notation. For example `10.0.190.1/24`. When no such address exists, an empty string is set.

* `ipv6_address` - The network's global IPv6 address in CIDR notation. For example `fd42:b40e:534a:b208::1/64`. When no such address exists, an empty string is set.

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<name>` - **Required** - Network name.

-> Clustered networks cannot be imported.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_network.mynet proj/net1
```

Example using the import block:

```hcl
resource "lxd_network" "mynet" {
  name    = "net1"
  project = "proj"
}

import {
  to = lxd_network.mynet
  id = "proj/net1"
}
```

## Notes

* The network resource `config` includes some keys that can be automatically generated by the LXD.
  If these keys are not explicitly defined by the user, they will be omitted from the Terraform
  state and treated as computed values.
    - `bridge.mtu`
    - `ipv4.nat`
    - `ipv4.address`
    - `ipv6.nat`
    - `ipv6.address`
    - `volatile.*`
