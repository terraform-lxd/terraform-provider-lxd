# incus_storage_bucket

Manages an Incus storage bucket.

## Example Usage

```hcl
resource "incus_storage_pool" "pool1" {
  name   = "mypool"
  driver = "zfs"
}

resource "incus_storage_bucket" "bucket1" {
  name = "mybucket"
  pool = incus_storage_pool.pool1.name
}
```

## Argument Reference

* `name` - **Required** - Name of the storage bucket.

* `pool` - **Required** - Name of storage pool to host the storage bucket.

* `description` - *Optional* - Description of the storage bucket.

* `config` - *Optional* - Map of key/value pairs of
  [storage bucket config settings](https://linuxcontainers.org/incus/docs/main/howto/storage_buckets/#configure-storage-bucket-settings).
  Config settings vary depending on the Storage Pool used.

* `project` - *Optional* - Name of the project where the storage bucket will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.


## Attribute Reference

The following attributes are exported:

* `location` - Name of the node where storage bucket was created. It could be useful with Incus in cluster mode.

## Importing

Import ID syntax: `[<remote>:][<project>]/<pool>/<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<pool>` - **Required** - Storage pool name.
* `<name>` - **Required** - Storage bucket name.

### Import example

Example using terraform import command:

```shell
terraform import incus_storage_bucket.bucket1 proj/pool1/bucket1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "incus_storage_bucket" "mybucket" {
  name    = "bucket1"
  pool    = "pool1"
  project = "proj"
}

import {
  to = incus_storage_bucket.mybucket
  id = "proj/pool1/mybucket"
}
```

## Notes

* Incus creates by default for each storage bucket an admin access key 
	and a secret key. This key can be imported using the `incus_storage_bucket_key` resource.

