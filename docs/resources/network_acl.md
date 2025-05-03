# lxd_network_acl

Manages an LXD network ACL.

See LXD network ACL [configuration reference](https://documentation.ubuntu.com/lxd/latest/howto/network_acls/) for how to configure network ACLs.

## Example Usage

```hcl
resource "lxd_network_acl" "acl1" {
  name = "my-acl"

  egress = [
    {
      description      = "DNS to cloudflare public resolvers (UDP)"
      action           = "allow"
      destination      = "1.1.1.1,1.0.0.1"
      destination_port = "53"
      protocol         = "udp"
      state            = "enabled"
    },
    {
      description      = "DNS to cloudflare public resolvers (TCP)"
      action           = "allow"
      destination      = "1.1.1.1,1.0.0.1"
      destination_port = "53"
      protocol         = "tcp"
      state            = "enabled"
    },
  ]

  ingress = [
    {
      description      = "Incoming SSH connections"
      action           = "allow"
      source           = "@external"
      destination_port = "22"
      protocol         = "tcp"
      state            = "logged"
    }
  ]
}
```

## Argument Reference

* `name` - **Required** - Name of the network ACL.

* `description` - *Optional* - Description of the network ACL.

* `ingress` - *Optional* - List of network ACL rules for ingress traffic. See reference below.

* `egress` - *Optional* - List of network ACL rules for egress traffic. See reference below.

* `config` - *Optional* - Map of key/value pairs of
  [network ACL config settings](https://documentation.ubuntu.com/lxd/latest/howto/network_acls/).

* `project` - *Optional* - Name of the project where the network ACL will be created.

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

The network ACL rule supports:

* `action` - **Required** - Action to take for the matching traffic. Possible values are `allow`, `allow-stateless`, `drop`, or `reject`.

* `description` - *Optional* - Description of the network ACL rule.

* `destination` - *Optional* - Comma-separated list of CIDR or IP ranges, destination subject name selectors (for egress rules), or leave the value empty for any.

* `destination_port` - *Optional* - If the protocol is `udp` or `tcp` you can specify a comma-separated list of ports or port ranges (start-end), or leave the value empty for any.

* `icmp_code` - *Optional* - If the protocol is `icmp4` or `icmp6` you can specify the ICMP code number, or leave the value empty for any.

* `icmp_type` - *Optional* - If the protocol is `icmp4` or `icmp6` you can specify the ICMP type number, or leave the value empty for any.

* `protocol` - *Optional* - Protocol to match. Possible values are `icmp4`, `icmp6`, `tcp`, or `udp`. Leave the value empty for any protocol.

* `source` - *Optional* - Comma-separated list of CIDR or IP ranges, source subject name selectors (for ingress rules), or leave the value empty for any.

* `state` - *Optional* - State of the rule. Possible values are `enabled`, `disabled`, and `logged`. Defaults to `enabled`.

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<name>` - **Required** - Network name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_network_acl.acl1 proj/my-acl
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_network_acl" "acl1" {
  name    = "my-acl"
  project = "proj"
}

import {
  to = lxd_network_acl.acl1
  id = "proj/my-acl"
}
```
