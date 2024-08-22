# incus_network_load_balancer

Incus load balancer resource forwards ports from external IPs to internal ones within its network,
distributing traffic among multiple backends.

-> The load balancer resource is exclusively compatible with OVN (Open Virtual Network).

For more information, please refer to [How to configuration network load balancers](https://linuxcontainers.org/incus/docs/main/howto/network_load_balancers/)
in the official Incus documentation.

## Example Usage

```hcl
resource "incus_network" "network" {
  name = "ovn"
  type = "ovn"

  config = {
    # ...
  }
}

resource "incus_network_lb" "load_balancer" {
  network        = incus_network.network.name
  description    = "My Load Balancer"
  listen_address = "10.10.10.200"

  config = {
    "key" = "value"
  }

  backend {
    name           = "instance-1"
    description    = "Load Balancer Backend"
    target_address = "10.0.0.10"
    target_port    = "80"
  }

  backend {
    name           = "instance-2"
    description    = "Load Balancer Backend"
    target_address = "10.0.0.11"
    target_port    = "80"
  }

  port {
    description = "Port 8080/tcp"
    protocol    = "tcp"
    listen_port = "8080"
    target_backend = [
      "instance-1",
      "instance-2",
    ]
  }
}
```

## Argument Reference

* `network` - **Required** - Name of the uplink network.

* `listen_address` - **Required** - IP address to listen on. Also, see the [Requirements for listen address](https://linuxcontainers.org/incus/docs/main/howto/network_load_balancers/#requirements-for-listen-addresses) in the official Incus documentation.

* `description` - *Optional* - Description of the network load balancer.

* `backend` - *Optional* - Load balancer's backend definition. See reference below.

* `port` - *Optional* - Load balancer's port definition. See reference below.

* `config` - *Optional* - Map of key/value pairs (load balancer's currently support only `user.*` keys).

* `project` - *Optional* - Name of the project where the load balancer will be spawned.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

The `backend` block supports:

* `name` - **Required** - Name of the load balancer's backend.

* `target_address` - **Required** - IP address to forward to.

* `target_port` - *Optional* - Target port(s) (e.g. `80`, `80,32000-32080`). Default: *`listen_port` of the corresponding `port` block*

* `description` - *Optional* - Description of the load balancer's backend.

The `port` block supports:

* `listen_port` - **Required** - Listen port(s) (e.g. `80`, `80,32000-32080`).

* `target_backend` - **Required** - Backend name(s) to forward to.

* `protocol` - *Optional* - Protocol of the port(s). Can be either `tcp` or `udp`. Default: `tcp`

* `description` - *Optional* - Description of the load balancer's port.

## Attribute Reference

No attributes are exported.


