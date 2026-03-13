# lxd_info

Provides general information about the LXD server.

## Example Usage

```hcl
data "lxd_info" "self" {}
```

## Argument Reference

No arguments are supported.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `api_extensions` - List of API extensions supported by the LXD server.

* `cluster_members` - Map of cluster members, which is empty if LXD is not clustered. The map key represents a cluster member name.

* `instance_types` - List of supported instance types (e.g. `virtual-machine`, `container`).
