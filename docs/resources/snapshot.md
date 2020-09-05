# lxd_snapshot

Manages a snapshot of an LXD container.

## Example Usage

```hcl
resource "lxd_snapshot" "snap1" {
  container_name = "${lxd_container.container1.name}"
  name           = "snap1"
}
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `name` - *Required* - Name of the snapshot.

* `container_name` - *Required* - The name of the container to snapshot.

* `stateful` - *Optional* - Set to `true` to create a stateful snapshot,
	`false` for stateless. Stateful snapshots include runtime state. Defaults to
	`true`.

## Attribute Reference

The following attributes are exported:

* `creation_date` - **Deprecated - use `created_at` instead** - The time LXD
  reported the snapshot was successfully created, in UTC.

* `created_at` - The time LXD  reported the snapshot was successfully created,
  in UTC.
