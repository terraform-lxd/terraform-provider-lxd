# lxd_instance

Provides information about an existing LXD instance.

## Example Usage

```hcl
data "lxd_instance" "inst" {
  name = "my-instance"
}
```

## Argument Reference

* `name` - **Required** - Name of the instance.

* `project` - *Optional* - Name of the project where instance is located.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote is used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `description` - Description of the instance.

* `type` - Instance type.

* `ephemeral` - Boolean indicating if this instance is ephemeral.

* `running` - Boolean indicating whether the instance is currently running.

* `profiles` - List of applied instance profiles.

* `devices` - Map of instance devices. The map key represents a device name.

* `limits` - Map of key/value pairs that define the
	[instance resources limits](https://documentation.ubuntu.com/lxd/latest/reference/instance_options/#resource-limits).

* `config` - Map of key/value pairs of
	[instance config settings](https://documentation.ubuntu.com/lxd/latest/reference/instance_options/).

* `interfaces` - Map of all instance network interfaces (excluding loopback device). The map key represents the name of the network device (from LXD configuration).

* `ipv4_address` - The instance's IPv4 address.

* `ipv6_address` - The instance's IPv6 address.

* `mac_address` - The instance's MAC address.

* `location` - Name of the cluster member where instance is located.

* `status` - The status of the instance.
