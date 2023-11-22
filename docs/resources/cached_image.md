# lxd_cached_image

Manages a locally-stored LXD image.

## Example Usage

```hcl
resource "lxd_cached_image" "xenial" {
  source_remote = "ubuntu"
  source_image  = "xenial/amd64"
}

resource "lxd_instance" "test1" {
  name      = "test1"
  image     = lxd_cached_image.xenial.fingerprint
  ephemeral = false
}
```

## Argument Reference

* `source_image` - **Required** - Fingerprint or alias of image to pull.

* `source_remote` - **Required** - Name of the LXD remote from where image will
	be pulled.

* `type` - *Optional* - Type of image to cache. Must be one of `container` or
  `virtual-machine`. Defaults to `container`.

* `aliases` - *Optional* - A list of aliases to assign to the image after
	pulling.

* `copy_aliases` - *Optional* - Whether to copy the aliases of the image from
	the remote. Valid values are `true` and `false`. Defaults to `true`.

* `project` - *Optional* - Name of the project where the image will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If it
	is not provided, the default provider remove will be used.

## Attribute Reference

The following attributes are exported:

* `architecture` - The image architecture (e.g. amd64, i386).

* `created_at` - The datetime of image creation, in Unix time.

* `fingerprint` - The unique hash fingperint of the image.

* `copied_aliases` - The list of aliases that were copied from the
  `source_image`.

## Notes

* See the LXD [documentation](https://documentation.ubuntu.com/lxd/en/latest/howto/images_remote) for more info on default image remotes.
