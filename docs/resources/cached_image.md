# lxd_cached_image

Manages a locally-stored LXD image.

## Example Usage

```hcl
resource "lxd_cached_image" "xenial" {
  image = "images:ubuntu/xenial/amd64"
}

resource "lxd_instance" "test1" {
  name      = "test1"
  image     = lxd_cached_image.xenial.fingerprint
  ephemeral = false
}
```

## Argument Reference

* `image` - **Required** - Image to pull, specified as `[<remote>:]<image>`.
  The remote is the name of a remote image server (e.g. `images`). If no remote
  is specified, the image is looked up on the provider's server. The image part
  can be an alias or fingerprint.

* `type` - *Optional* - Type of image to cache. Must be one of `container` or
  `virtual-machine`. Defaults to `container`.

* `aliases` - *Optional* - A list of aliases to assign to the image after
	pulling.

* `copy_aliases` - *Optional* - Whether to copy the aliases of the image from
	the remote. Valid values are `true` and `false`. Defaults to `false`.

* `project` - *Optional* - Name of the project where the image will be stored.

## Attribute Reference

The following attributes are exported:

* `architecture` - The image architecture (e.g. amd64, i386).

* `created_at` - The datetime of image creation, in Unix time.

* `fingerprint` - The unique hash fingperint of the image.

* `copied_aliases` - The list of aliases that were copied from the source image.

## Notes

* See the LXD [documentation](https://documentation.ubuntu.com/lxd/latest/howto/images_remote) for more info on default image remotes.
