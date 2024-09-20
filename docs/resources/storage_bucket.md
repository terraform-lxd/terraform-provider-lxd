# lxd_storage_bucket

Manages an LXD storage bucket.

## Example Usage

```hcl
resource "lxd_storage_pool" "pool" {
  name   = "mypool"
  driver = "zfs"
}

resource "lxd_storage_bucket" "bucket" {
  name = "mybucket"
  pool = lxd_storage_pool.pool.name
}
```

## Argument Reference

* `name` - **Required** - Name of the storage bucket.

* `pool` - **Required** - Name of storage pool to host the storage bucket.

* `description` - *Optional* - Description of the storage bucket.

* `config` - *Optional* - Map of key/value pairs of
  [storage bucket config settings](https://documentation.ubuntu.com/lxd/en/latest/howto/storage_buckets/#configure-storage-bucket-settings).
  Note that config settings vary depending on the used storage pool.

* `project` - *Optional* - Name of the project where the storage bucket will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

* `target` - *Optional* - Specify a target node in a cluster.


## Attribute Reference

The following attributes are exported:

* `location` - Name of the node where storage bucket was created.

## Importing

Import ID syntax: `[<remote>:][<project>]/<pool>/<name>`

* `<remote>` - *Optional* - Remote name.
* `<project>` - *Optional* - Project name.
* `<pool>` - **Required** - Storage pool name.
* `<name>` - **Required** - Storage bucket name.

### Import example

Example using terraform import command:

```shell
$ terraform import lxd_storage_bucket.bucket proj/mypool/mybucket
```

Example using the import block (only available in Terraform v1.5.0 and later):

```hcl
resource "lxd_storage_bucket" "bucket" {
  name    = "mybucket"
  pool    = "mypool"
  project = "proj"
}

import {
  to = lxd_storage_bucket.bucket
  id = "proj/mypool/mybucket"
}
```

## Notes

* By default, LXD creates each storage bucket with an admin access key and a secret key.
	Those keys can be imported using the `lxd_storage_bucket_key` resource.

