# incus_storage_bucket_key

Manages an Incus storage bucket key.

~> **Note:** The exported attributes `access_key` and `secret_key` will be stored in the raw state as plain-text. [Read more about sensitive data in state](https://www.terraform.io/language/state/sensitive-data).

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

resource "incus_storage_bucket_key" "key1" {
  name           = "mykey"
  pool           = incus_storage_bucket.bucket1.pool
  storage_bucket = incus_storage_bucket.bucket1.name
}
```

## Argument Reference

* `name` - **Required** - Name of the storage bucket key.

* `pool` - **Required** - Name of storage pool to host the storage bucket key.

* `storage_bucket` - **Required** - Name of the storage bucket.

* `description` - *Optional* - Description of the storage bucket key.

* `role` - *Optional* - Name of the role that controls the access rights for the
  key. If not specified, the default role is used, as described in the [official documentation](https://linuxcontainers.org/incus/docs/main/howto/storage_buckets/#manage-storage-bucket-keys).

* `project` - *Optional* - Name of the project where the storage bucket key will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.


## Attribute Reference

The following attributes are exported:

* `access_key` - Access key of the storage bucket key.

* `secret_key` - Secret key of the storage bucket key.

## Importing

Import ID syntax: `[<remote>:][<project>]/<pool>/<storage_bucket>/<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<pool>` - **Required** - Storage pool name.
* `<name>` - **Required** - Storage bucket name.

### Import example

Example using terraform import command:

```shell
$ terraform import incus_storage_bucket_key.key1 proj/pool1/bucket1/key1
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "incus_storage_bucket_key" "mykey" {
  name           = "mykey"
  project        = "proj"
  pool           = "pool1"
  storage_bucket = "bucket1"
}

import {
  to = incus_storage_bucket.mykey
  id = "proj/pool1/bucket1/mykey"
}
```

## Notes

* Incus creates by default for each storage bucket an admin access key
  and a secret key. This key can be imported using the resource.
