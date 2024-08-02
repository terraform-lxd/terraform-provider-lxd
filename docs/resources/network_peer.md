# lxd_network_peer

Manages an LXD network peer routing. Currently, network peers can be created only for between two OVN networks.

## Example Usage

```hcl
resource "lxd_network_peer" "peer-1-2" {
  name           = "peer-${lxd_network.network_2.name}"
  description    = "Peer with network ${lxd_network.network_2.name}"
  source_network = lxd_network.network_1.name
  target_network = lxd_network.network_2.name
}

resource "lxd_network_peer" "peer-2-1" {
  name           = "peer-${lxd_network.network_1.name}"
  description    = "Peer with network ${lxd_network.network_1.name}"
  source_network = lxd_network.network_2.name
  target_network = lxd_network.network_1.name
}
```

## Argument Reference

* `name` - **Required** - Name of the network peer.

* `source_network` - **Required** - Name of the source network.

* `target_network` - **Required** - Name of the target network.

* `source_project` - *Optional* - Name of the source network project. Defaults to `default`.

* `target_project` - *Optional* - Name of the target network project. Defaults to value of the *source_project* field.

* `description` - *Optional* - Description of the network peer.

* `config` - *Optional* - Map of key/value pairs of [network peer config settings](https://documentation.ubuntu.com/lxd/en/latest/howto/network_ovn_peers/#peering-properties).

* `remote` - *Optional* - The remote in which the resource will be created. If not provided, the provider's default remote will be used.

## Attribute Reference

No attributes are exported.

## Importing

Import ID syntax: `[<remote>:]/<name>/<sourceProject>/<sourceNetwork>/<targetProject>/<targetNetwork>`

* `<remote>` - *Optional* - Remote name.
* `<name>` - **Required** - Network peer name.
* `<sourceProject>` - **Required** - Source project name.
* `<sourceNetwork>` - **Required** - Source network name.
* `<targetProject>` - **Required** - Target project name.
* `<targetNetwork>` - **Required** - Target network name.

-> **Note:** The import ID must include a forward slash (`/`) before the network peer name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_network_peer.mypeer /peer1/srcProj/srcNet/dstProj/dstNet
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_network_peer" "mypeer" {
  name           = "peer1"
  source_project = "srcProj"
  source_network = "srcNet"
  target_project = "dstProj"
  target_network = "dstNet"
}

import {
  to = lxd_network_peer.mypeer
  id = "/peer1/srcProj/srcNet/dstProj/dstNet"
}
```

## Notes

* See the LXD [documentation](https://documentation.ubuntu.com/lxd/en/latest/howto/network_ovn_peers/) for more information on network peer routing.
