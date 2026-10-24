# lxd_cluster_link

Manages an LXD cluster link.

Cluster links let an LXD cluster (or standalone server) call the API of another cluster over
mutual TLS.

## LXD documentation

* [How to create cluster links](https://documentation.ubuntu.com/lxd/latest/howto/cluster_links_create/)
* [How to manage cluster links](https://documentation.ubuntu.com/lxd/latest/howto/cluster_links_manage/)

## Required API extensions

The target LXD server must support the following API extensions:

* `cluster_links` - For all link types.
* `cluster_links_unidirectional` - For links of type `unidirectional`.
* `cluster_links_public` - For links of type `public`.

## Example Usage

### Bidirectional link

A bidirectional link is created on both clusters. The first cluster creates the link without a
trust token, which issues one. The second cluster creates its link with that token, which
activates the link on both clusters.

```hcl
# On cluster A: issue a trust token for cluster B.
resource "lxd_cluster_link" "cluster_b" {
  name        = "cluster-b"
  description = "Backup cluster"
  auth_groups = ["link"]
}

output "trust_token" {
  value     = lxd_cluster_link.cluster_b.trust_token
  sensitive = true
}
```

```hcl
# On cluster B: consume the trust token issued by cluster A.
resource "lxd_cluster_link" "cluster_a" {
  name        = "cluster-a"
  trust_token = var.trust_token_from_cluster_a
  auth_groups = ["link"]
}
```

When the provider is configured with both clusters as remotes, both halves can live in a single
configuration:

```hcl
resource "lxd_cluster_link" "cluster_b" {
  remote      = "cluster-a"
  name        = "cluster-b"
  auth_groups = ["link"]
}

resource "lxd_cluster_link" "cluster_a" {
  remote      = "cluster-b"
  name        = "cluster-a"
  trust_token = lxd_cluster_link.cluster_b.trust_token
  auth_groups = ["link"]
}
```

### Unidirectional link

A unidirectional link lets this cluster call the linked cluster, but not the other way around.
The linked cluster issues the trust token by creating a pending cluster link identity:

```shell
lxc auth identity create cluster-link/cluster-a --group link
```

```hcl
resource "lxd_cluster_link" "cluster_b" {
  name        = "cluster-b"
  type        = "unidirectional"
  trust_token = var.trust_token_from_cluster_b
}
```

### Public link

A public link lets this cluster call the linked cluster anonymously. No trust token is exchanged
and no identity is created. LXD fetches the linked cluster's certificate and pins it once its
fingerprint is confirmed.

```hcl
resource "lxd_cluster_link" "cluster_b" {
  name           = "cluster-b"
  type           = "public"
  remote_address = "10.0.0.2:8443"
  fingerprint    = "a1b2c3d4..."
}
```

The link is created only if the certificate served at `remote_address` has the configured
`fingerprint`, otherwise the apply fails. The fingerprint can be read from the linked cluster
with `lxc info`, in the `certificate_fingerprint` field.

## Argument Reference

* `name` - **Required** - Name of the cluster link. For a `bidirectional` link, LXD also uses it
  as the name of the identity it creates for the linked cluster.

* `type` - *Optional* - Type of the cluster link. Can be `bidirectional`, `unidirectional` or
  `public`. Defaults to `bidirectional`.

* `description` - *Optional* - Description of the cluster link.

* `config` - *Optional* - Map of key/value pairs of
  [cluster link config settings](https://documentation.ubuntu.com/lxd/latest/reference/cluster_link_config/).
  Only `user.*` keys can be set. The `volatile.*` keys are managed by LXD and are not part of the
  configuration in state.

* `auth_groups` - *Optional* - Set of auth groups to add the linked cluster's identity to.
  Applicable only to `bidirectional` links, because the other types create no identity on this
  cluster. Changing the groups updates the identity in place.

* `trust_token` - *Optional* - Trust token issued by the linked cluster. If omitted on a
  `bidirectional` link, the link is created in a pending state and a trust token for the linked
  cluster is issued and exported in this attribute instead. Required to create a `unidirectional`
  link. Must not be set for `public` links. Changing a configured token replaces the link.

* `remote_address` - *Optional* - Address of the linked cluster. Required to create a `public`
  link and must not be set for other types.

* `fingerprint` - *Optional* - SHA-256 fingerprint of the linked cluster's certificate. Required
  to create a `public` link and must not be set for other types.

* `remote` - *Optional* - The remote in which the resource will be created. If not provided, the
  provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

* `addresses` - Addresses of the linked cluster's members. Empty while a `bidirectional` link is
  pending, that is until the linked cluster has redeemed the issued trust token.

* `expires_at` - Time at which the issued trust token expires, in RFC 3339 format and UTC. It is
  null when no token was issued or the server config `core.remote_token_expiry` is not set, which
  means the token does not expire.

## Importing

Import ID syntax: `[<remote>:]<name>`

* `<remote>` - *Optional* - Remote name.
* `<name>` - **Required** - Cluster link name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_cluster_link.mylink cluster-b
```

Example using the import block:

```hcl
resource "lxd_cluster_link" "mylink" {
  name = "cluster-b"
}

import {
  to = lxd_cluster_link.mylink
  id = "cluster-b"
}
```

## Notes

* Deleting a `bidirectional` link removes the trust and identity only on this cluster. The linked
  cluster keeps its link until it is deleted there as well.
