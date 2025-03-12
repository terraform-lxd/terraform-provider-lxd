# lxd_info

Provides general information about LXD remote.

## Example Usage

```hcl
data "lxd_info" "local" {
  remote = "local"
}
```

## Argument Reference

* `remote` - *Optional* - The remote to inspect. If not provided, the provider's default remote is used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `api_extensions` - List of API extensions supported by the LXD server.

* `cluster_members` - Map of cluster members, which is empty if LXD is not clustered. The map key represents a cluster member name.

* `instance_types` - List of supported instance types (e.g. `virtual-machine`, `container`).
