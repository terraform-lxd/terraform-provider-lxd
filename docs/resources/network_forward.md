# incus_network_forward

Manages an Incus network forward.

See Incus network forward [configuration reference](https://linuxcontainers.org/incus/docs/main/howto/network_forwards/) for how to configure network forwards.

## Example Usage

```hcl
resource "incus_network" "my_network" {
  name = "my-network"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat"     = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat"     = "true"
  }
}

resource "incus_network_forward" "my_forward" {
  network        = incus_network.my_network.name
  listen_address = "10.150.19.10"

  config = {
    target_address = "10.150.19.111"
  }

  ports = [
    {
      description    = "SSH"
      protocol       = "tcp"
      listen_port    = "22"
      target_port    = "2022"
      target_address = "10.150.19.112"
    },
    {
      description    = "HTTP"
      protocol       = "tcp"
      listen_port    = "80"
      target_port    = "8080"
      target_address = "10.150.19.112"
    }
  ]
}
```

## Argument Reference

* `network` - **Required** - Name of the network.

* `listen_address` - **Required** - IP address to listen on.

* `description` - *Optional* - Description of the network forward.

* `config` - *Optional* - Map of key/value pairs of
  [network forward config settings](hhttps://linuxcontainers.org/incus/docs/main/howto/network_forwards/).

* `project` - *Optional* - Name of the project where the network forward will be created.

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

* `ports` - *Optional* - List of port specifications. See reference below.

The network forward port supports:

* `protocol` - **Required** - Protocol for the port(s) (`tcp` or `udp`). If not set then `tcp` will be used.

* `listen_port` - **Required** - Listen port(s) (e.g. `80,90-100`)

* `target_address` - **Required** - IP address to forward to

* `target_port` - *Optional* - T arget port(s) (e.g. `70,80-90` or `90`), same as listen_port if empty

* `description` - *Optional* - Description of port(s)

## Importing

Import ID syntax: `[<remote>:][<project>/]<network-name>/<listen-address>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<network-name>` - **Required** - Network name.
* `<listen-address>` - **Required** - IP Listen Address.

### Import example

Example using terraform import command:

```shell
$ terraform import incus_network_forward.forward1 proj/my-network/10.150.19.10
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "incus_network_forward" "forward1" {
  network_name   = "my-network"
  listen_address = "10.150.19.10"
  project        = "proj"
}

import {
  to = incus_network_forward.forward1
  id = "proj/my-network/10.150.19.10"
}
```