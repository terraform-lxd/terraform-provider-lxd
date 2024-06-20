# lxd_network_acl

Manages an LXD network ACL.

See LXD network ACL [configuration reference](https://documentation.ubuntu.com/lxd/en/latest/howto/network_acls/) for how to configure network ACLs.

## Example Usage

```hcl
resource "lxd_network_acl" "acl1" {
  name    = "my-acl"

  egress = [
    {
      action           = "allow"
      destination      = "1.1.1.1,1.0.0.1"
      destination_port = "53"
      protocol         = "udp"
      description      = "DNS to cloudflare public resolvers (UDP)"
      state            = "enabled"
    },
    {
      action           = "allow"
      destination      = "1.1.1.1,1.0.0.1"
      destination_port = "53"
      protocol         = "tcp"
      description      = "DNS to cloudflare public resolvers (TCP)"
      state            = "enabled"
    },
  ]

  ingress = [
    {
      action           = "allow"
      source           = "@external"
      destination_port = "22"
      protocol         = "tcp"
      description      = "Incoming SSH connections"
      state            = "logged"
    }
  ]
}
```

## Argument Reference

* `name` - **Required** - Name of the network ACL.

* `description` - *Optional* - Description of the network ACL.

* `config` - *Optional* - Map of key/value pairs of
  [network ACL config settings](https://documentation.ubuntu.com/lxd/en/latest/howto/network_acls/).

* `project` - *Optional* - Name of the project where the network ACL will be created.

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

* `egress` - *Optional* - List of network ACL rules for egress traffic. See reference below.

* `ingress` - *Optional* - List of network ACL rules for ingress traffic. See reference below.

The network ACL rule supports:

* `action` - **Required** - Action to take for matching traffic , must be one of allow, allow-stateless, drop, reject

* `description` - *Optional* - Description of the network ACL rule.

* `destination_port` - *Optional* - If protocol is `udp` or tcp, then a comma-separated list of ports or port ranges (start-end inclusive), or empty for any

* `destination` - *Optional* - Comma-separated list of CIDR or IP ranges, destination subject name selectors (for egress rules), or empty for any

* `icmp_code` - *Optional* - If protocol is `icmp4` or `icmp6`, then ICMP code number, or empty for any

* `icmp_type` - *Optional* - If protocol is `icmp4` or `icmp6`, then ICMP type number, or empty for any

* `protocol` - *Optional* - If protocol is `udp` or `tcp`, then a comma-separated list of ports or port ranges (start-end inclusive), or empty for any

* `source` - *Optional* - Comma-separated list of CIDR or IP ranges, source subject name selectors (for ingress rules), or empty for any

* `state` - *Optional* - State of the rule (enabled, disabled or logged), defaulting to enabled if not specified

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<name>` - **Required** - Network name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_network_acl.acl1 proj/acl1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_network_acl" "acl1" {
  name    = "acl1"
  project = "proj"
}

import {
  to = lxd_network_acl.acl1
  id = "proj/acl1"
}
```
