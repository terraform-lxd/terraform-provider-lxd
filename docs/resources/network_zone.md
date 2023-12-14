# incus_network_zone

Manages an Incus network zone.

You must be using Incus 4.20 or later. See
[this](https://discuss.linuxcontainers.org/t/incus-built-in-dns-server/12033)
forum post for details about Incus network zones and the
[configuration reference](https://documentation.ubuntu.com/incus/en/latest/howto/network_zones/)
for all network zone details.

## Example Usage

```hcl
resource "incus_network_zone" "zone" {
  name = "custom.example.org"

  config = {
    "dns.nameservers"  = "ns.custom.example.org"
    "peers.ns.address" = "127.0.0.1"
  }
}

resource "incus_network_zone_record" "record" {
  name = "ns"
  zone = incus_network_zone.zone.id

  entry {
      type  = "CNAME"
      value = "<incus.host.name>."
  }

  entry {
      type = "A"
      value = "<incus.host.ip>"
  }
}
```

See the `incus_network_zone_record` resource for information on how to configure network zone records.

## Argument Reference

- `name` - **Required** - Name of the network zone.

- `description` - _Optional_ - Description of the network zone.

- `config` - _Optional_ - Map of key/value pairs of
  [network zone_config settings](https://documentation.ubuntu.com/incus/en/latest/howto/network_zones/#configuration-options).

- `project` - _Optional_ - Name of the project where the network zone will be created.

- `remote` - _Optional_ - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

No attributes are exported.

## Importing

Import ID syntax: `[<remote>:][<project>/]<name>`

- `<remote>` - _Optional_ - Remote name.
- `<project>` - _Optional_ - Project name.
- `<name>` - **Required** - Network zone name.

### Import example

Example using terraform import command:

```shell
$ terraform import incus_network_zone.myzone proj/zone1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "incus_network_zone" "myzone" {
  name    = "zone1"
  project = "proj"
}

import {
  to = incus_network_zone.myzone
  id = "proj/zone1"
}
```
