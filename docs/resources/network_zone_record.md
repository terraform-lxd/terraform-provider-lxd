# lxd_network_zone_record

Manages an LXD network zone record.

You must be using LXD 4.20 or later. See
[this](https://discuss.linuxcontainers.org/t/lxd-built-in-dns-server/12033)
forum post for details about LXD network zones and the
[configuration reference](https://documentation.ubuntu.com/lxd/latest/howto/network_zones/)
for all network zone details.

## Example Usage

```hcl
resource "lxd_network_zone" "zone" {
  name = "custom.example.org"

  config = {
    "dns.nameservers"  = "ns.custom.example.org"
    "peers.ns.address" = "127.0.0.1"
  }
}

resource "lxd_network_zone_record" "record" {
  name = "ns"
  zone = lxd_network_zone.zone.name

  entry {
      type  = "CNAME"
      value = "<lxd.host.name>."
  }

  entry {
      type  = "A"
      value = "<lxd.host.ip>"
  }
}
```

See the `lxd_network_zone` resource for information on how to configure network zones.

## Argument Reference

* `name` - **Required** - Name of the network zone record.

* `zone` - **Required** - Name of the zone to add the entries of this record.

* `description` - *Optional* - Description of the network zone.

* `entry` - *Optional* - Entry in network zone record - see below.

* `config` - *Optional* - Map of key/value pairs of
	[network zone_config settings](https://documentation.ubuntu.com/lxd/latest/howto/network_zones/#configuration-options).

* `project` - *Optional* - Name of the project where the network zone record will be created.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

The `entry` block supports:

* `type` - **Required** - Entry type. Valid values are DNS record type, e.g. `A`, `AAAA`, `CNAME`, `TXT`, etc.

* `value` - **Required** - Entry value.

* `ttl` - *Optional* - Entry time to live (TTL).

## Attribute Reference

No attributes are exported.

## Importing

Import ID syntax: `[<remote>:][<project>]/<zone>/<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<zone>` - **Required** - Network zone name.
* `<name>` - **Required** - Network zone record name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_network_zone_record.myrecord proj/zone1/record1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_network_zone_record" "myrecord" {
  name    = "record1"
  zone    = "zone1"
  project = "proj"
}

import {
  to = lxd_network_zone_record.myrecord
  id = "proj/zone1/record1"
}
```
