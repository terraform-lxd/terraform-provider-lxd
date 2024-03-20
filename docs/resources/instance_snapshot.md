# incus_instance_snapshot

Manages a snapshot of an Incus instance.

## Example Usage

```hcl
resource "incus_instance" "instance" {
  name      = "my-instance"
  image     = "ubuntu"
  ephemeral = false
}

resource "incus_instance_snapshot" "snap1" {
  name     = "my-snapshot-1"
  instance = incus_instance.instance.name
}
```

## Argument Reference

* `name` - **Required** - Name of the snapshot.

* `instance` - **Required** - The name of the instance to snapshot.

* `stateful` - *Optional* - Set to `true` to create a stateful snapshot,
	`false` for stateless. Stateful snapshots include runtime state. Defaults to
	`false`.

* `project` - *Optional* - Name of the project where the snapshot will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

* `created_at` - The time Incus  reported the snapshot was successfully created,
  in UTC.
