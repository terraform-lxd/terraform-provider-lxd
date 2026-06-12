# lxd_image

Manages a locally-stored LXD image.

## Example Usage

```hcl
resource "lxd_image" "xenial" {
  source_image = {
    image = "ubuntu:xenial/amd64"
  }
}

resource "lxd_instance" "test1" {
  name      = "test1"
  image     = lxd_image.xenial.fingerprint
  ephemeral = false
}
```

## Argument Reference

* `source_image` - *Optional* - The source image from which the image will be copied. See reference below.

* `source_instance` - *Optional* - The source instance from which the image will be created. See reference below.

* `aliases` - *Optional* - A list of aliases to assign to the image after
	pulling.

* `project` - *Optional* - Name of the project where the image will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

The `source_image` block supports:

* `image` - **Required** - Name of the source image in the format `[<remote>:]<image>`.
  If the remote is omitted, the provider's default remote is used.

* `type` - *Optional* - Type of image to cache. Must be one of `container` or
  `virtual-machine`. Defaults to `container`.

* `architecture` - *Optional* - Architecture of the image to pull (e.g.
  `amd64`, `arm64`). If not provided, the default architecture of
  `source_image` is used.

* `copy_aliases` - *Optional* - Whether to copy the aliases of the image from
  the remote. Valid values are `true` and `false`. Defaults to `false`.

The `source_instance` block supports:

* `name` - **Required** - Name of the source instance.

* `snapshot` - *Optional* - Name of the snapshot of the source instance.

## Attribute Reference

The following attributes are exported:

* `created_at` - The datetime of image creation, in Unix time.

* `fingerprint` - The unique hash fingperint of the image.

* `copied_aliases` - The list of aliases that were copied from the
  `source_image`.

## Notes

* See the LXD [documentation](https://documentation.ubuntu.com/lxd/latest/howto/images_remote) for more info on default image remotes.
