# incus_snapshot

Manages a snapshot of an Incus container.

## Example Usage

```hcl
resource "incus_instance" "instance" {
  name      = "my-instance"
  image     = "ubuntu"
  ephemeral = false
}

resource "incus_snapshot" "snap1" {
  name     = "my-snapshot-1"
  instance = incus_instance.instance.name
}
```

## Argument Reference

- `name` - **Required** - Name of the snapshot.

- `instance` - **Required** - The name of the instance to snapshot.

- `stateful` - _Optional_ - Set to `true` to create a stateful snapshot,
  `false` for stateless. Stateful snapshots include runtime state. Defaults to
  `false`.

- `project` - _Optional_ - Name of the project where the snapshot will be stored.

- `remote` - _Optional_ - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

- `created_at` - The time Incus reported the snapshot was successfully created,
  in UTC.
