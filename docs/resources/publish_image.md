# lxd_publish_image

Create a LXD image from a container

## Example Usage

```hcl
resource "lxd_cached_image" "xenial" {
  source_remote = "ubuntu"
  source_image  = "xenial/amd64"
}

resource "lxd_instance" "test1" {
  name  = "test1"
  image = lxd_cached_image.xenial.fingerprint

  start_on_create = false
}

resource "lxd_publish_image" "test1" {
  instance = lxd_instance.test1
  aliases  = ["test1_img"]
}
```

## Argument Reference

* `instance` - **Required** - The name of the instance.

* `aliases` - *Optional* - A list of aliases to assign to the image.

* `properties` - *Optional* - A map of properties to assign to the image.

* `public` - *Optional* - Whether the image can be downloaded by untrusted users.
	Valid values are `true` and `false`. Defaults to `false`.

* `filename` - *Optional* - Used for export.

* `compression_algorithm` - *Optional* - Override the compression algorithm for the image.
    Valid values are (`bzip2`, `gzip`, `lzma`, `xz` or `none`). Defaults to `gzip`.

* `triggers` - *Optional* - A list of arbitrary strings that, when changed, will force the resource to be replaced.

* `project` - *Optional* - Name of the project where the published image will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider's remote is used.

## Attribute Reference

The following attributes are exported:

* `fingerprint` - The fingerprint of the published image.

* `architecture` - The architecture of the published image.

* `created_at` - The creation timestamp of the published image.

## Notes

* Image can be published only if the container is stopped.
