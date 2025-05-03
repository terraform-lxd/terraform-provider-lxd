# lxd_network

Provides information about an existing LXD network.

## Example Usage

```hcl
data "lxd_network" "net" {
  name = "lxdbr0"
}

resource "lxd_instance" "inst" {
  name    = "my-instance"
  network = data.lxd_network.net.name
}
```

## Argument Reference

* `name` - **Required** - Name of the network.

* `project` - *Optional* - Name of the project where network is located.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote is used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `description` - Network description.

* `type` - Network type.

* `managed` - Whether or not the network is managed.

* `config` - Map of key/value pairs of
	[network config settings](https://documentation.ubuntu.com/lxd/latest/networks/).
