# incus_image_publish

Create an Incus image from an instance

## Example Usage

```hcl
resource "incus_image" "xenial" {
  source_remote = "ubuntu"
  source_image  = "xenial/amd64"
}

resource "incus_instance" "test1" {
  name    = "test1"
  image   = incus_image.xenial.fingerprint
  running = false
}

resource "incus_image_publish" "test1" {
  instance = incus_instance.test1
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
	not provided, the provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

* `fingerprint` - The fingerprint of the published image.

* `architecture` - The architecture of the published image.

* `created_at` - The creation timestamp of the published image.

## Notes

* Image can only be published if the instance is stopped.
