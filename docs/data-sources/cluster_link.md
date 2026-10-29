# lxd_cluster_link

Provides information about an existing LXD cluster link.

## Example Usage

```hcl
data "lxd_cluster_link" "link" {
  name = "cluster-b"
}
```

## Argument Reference

* `name` - **Required** - Name of the cluster link.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote is used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `description` - Description of the cluster link.

* `type` - Type of the cluster link (`bidirectional`, `unidirectional` or `public`).

* `config` - Map of key/value pairs of cluster link config settings. Only `user.*` keys are
  included, the `volatile.*` keys are managed by LXD.

* `addresses` - Addresses of the linked cluster's members. Empty while a `bidirectional` link is
  pending.
