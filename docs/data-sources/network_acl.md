# lxd_network_acl

Provides information about an existing LXD network ACL.

## Example Usage

```hcl
data "lxd_network_acl" "acl" {
  name = "my-acl"
}
```

## Argument Reference

* `name` - **Required** - Name of the network ACL.

* `project` - *Optional* - Name of the project where the ACL is located.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote is used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `description` - Description of the network ACL.

* `config` - Map of key/value pairs of network ACL config settings.

* `egress` - Set of egress rules. Each rule exports:
  * `action` - Rule action (`allow`, `allow-stateless`, `drop`, `reject`).
  * `state` - Rule state (`enabled`, `disabled`, `logged`).
  * `description` - Rule description.
  * `source` - Source address/CIDR.
  * `destination` - Destination address/CIDR.
  * `destination_port` - Destination port(s).
  * `protocol` - Protocol (`tcp`, `udp`, `icmp4`, `icmp6`).
  * `icmp_type` - ICMP type (only relevant for icmp4/icmp6).
  * `icmp_code` - ICMP code (only relevant for icmp4/icmp6).

* `ingress` - Set of ingress rules. Same attributes as `egress`.
