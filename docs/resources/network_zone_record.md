# lxd_network_zone_record

Manages an LXD network zone record.

You must be using LXD 4.20 or later. See
[this](https://discuss.linuxcontainers.org/t/lxd-built-in-dns-server/12033)
forum post for details about LXD network zones and the
[configuration reference](https://documentation.ubuntu.com/lxd/en/latest/howto/network_zones/)
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
  zone = lxd_network_zone.zone.id

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

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `name` - *Required* - Name of the network zone record.

* `zone` - *Required* - Name of the zone to add the entries of this record.

* `entry` - *Optional* - Entry in network zone record - see below.

* `config` - *Optional* - Map of key/value pairs of
	[network zone_config settings](https://documentation.ubuntu.com/lxd/en/latest/howto/network_zones/#configuration-options).

The `entry` block supports:

* `type` - *Required* - The entry type. Valid values are DNS record type, e.g. `A`, `AAAA`, `CNAME`, `TXT`, etc.

* `value` - *Required* - The entry value.

## Attribute Reference

No attributes are exported.

## Importing

Network zone records can be imported with the following command:

```shell
$ terraform import lxd_network_zone_record.my_record [<remote>:][<project>]/<zone_name>/<record_name>
```

