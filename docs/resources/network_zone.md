# lxd_network_zone

Manages an LXD network zone.

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
      type = "A"
      value = "<lxd.host.ip>"
  }
}
```

See the `lxd_network_zone_record` resource for information on how to configure network zone records.


## Argument Reference

* `name` - **Required** - Name of the network zone.

* `description` - *Optional* - Description of the network zone.

* `config` - *Optional* - Map of key/value pairs of
	[network zone_config settings](https://documentation.ubuntu.com/lxd/en/latest/howto/network_zones/#configuration-options).

* `project` - *Optional* - Name of the project where the network zone will be created.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

## Attribute Reference

No attributes are exported.

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<name>` - **Required** - Network zone name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_network_zone.myzone proj/zone1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_network_zone" "myzone" {
  name    = "zone1"
  project = "proj"
}

import {
  to = lxd_network_zone.myzone
  id = "proj/zone1"
}
```

