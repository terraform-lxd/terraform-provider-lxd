# lxd_storage_bucket_key

Manages an LXD storage bucket key.

~> **Warning:** The exported attributes `access_key` and `secret_key` are stored in the Terraform state as plain-text.
  Read more about [sensitive data in state](https://www.terraform.io/language/state/sensitive-data).

## Example Usage

```hcl
resource "lxd_storage_pool" "pool" {
  name   = "mypool"
  driver = "zfs"
}

resource "lxd_storage_bucket" "bucket" {
  name = "mybucket"
  pool = lxd_storage_pool.pool1.name
}

resource "lxd_storage_bucket_key" "key" {
  name   = "mykey"
  pool   = lxd_storage_bucket.bucket.pool
  bucket = lxd_storage_bucket.bucket.name
}
```

## Argument Reference

* `name` - **Required** - Name of the storage bucket key.

* `pool` - **Required** - Name of storage pool to host the storage bucket key.

* `bucket` - **Required** - Name of the storage bucket.

* `description` - *Optional* - Description of the storage bucket key.

* `role` - *Optional* - Name of the role that controls the access rights for the key.
   If not specified, the default role is used, as described in the [official documentation](https://documentation.ubuntu.com/lxd/en/latest/howto/storage_buckets/#manage-storage-bucket-keys).

* `project` - *Optional* - Name of the project where the storage bucket key will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If not provided,
  the provider's default remote will be used.


## Attribute Reference

The following attributes are exported:

* `access_key` - Access key of the storage bucket key.

* `secret_key` - Secret key of the storage bucket key.

## Importing

Import ID syntax: `[<remote>:][<project>]/<pool>/<bucket>/<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<pool>` - **Required** - Storage pool name.
* `<bucket>` - **Required** - Storage bucket name.
* `<name>` - **Required** - Storage bucket key name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_storage_bucket_key.key proj/mypool/mybucket/mykey
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_storage_bucket_key" "key" {
  name    = "mykey"
  project = "proj"
  pool    = "mypool"
  bucket  = "mybucket"
}

import {
  to = lxd_storage_bucket_key.key
  id = "proj/mypool/mybucket/mykey"
}
```

## Notes

* By default, LXD creates each storage bucket with an admin access key and a secret key.
  Those keys can be imported if needed.
