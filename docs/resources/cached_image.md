# lxd_cached_image

Manages a locally-stored LXD image.

## Example Usage

```hcl
resource "lxd_cached_image" "xenial" {
  source_remote = "ubuntu"
  source_image  = "xenial/amd64"
}

resource "lxd_container" "test1" {
  name      = "test1"
  image     = "${lxd_cached_image.xenial.fingerprint}"
  ephemeral = false
}
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If it
	is not provided, the default provider remove will be used.

* `source_remote` - *Required* - Name of the LXD remote from where image will
	be pulled.

* `source_image` - *Required* - Fingerprint or alias of image to pull.

* `aliases` - *Optional* - A list of aliases to assign to the image after
	pulling.

* `copy_aliases` - *Optional* - Whether to copy the aliases of the image from
	the remote. Valid values are `true` and `false`. Defaults to `true`.

## Attribute Reference

The following attributes are exported:

* `architecture` - The image architecture (e.g. amd64, i386).

* `created_at` - The datetime of image creation, in Unix time.

* `fingerprint` - The unique hash fingperint of the image.

* `copied_aliases` - The list of aliases that were copied from the
  `source_image`.

## Notes

* See the LXD [documentation](https://linuxcontainers.org/lxd/getting-started-cli/#using-the-built-in-image-remotes) for more info on default image remotes.
