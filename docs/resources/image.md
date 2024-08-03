# incus_image

Manages a locally-stored Incus image.

## Example Usage

```hcl
resource "incus_image" "xenial" {
  source_remote = "ubuntu"
  source_image  = "xenial/amd64"
}

resource "incus_instance" "test1" {
  name      = "test1"
  image     = incus_image.xenial.fingerprint
  ephemeral = false
}
```

## Argument Reference

* `source_image` - **Required** - Fingerprint or alias of image to pull.

* `source_remote` - **Required** - Name of the Incus remote from where image will
	be pulled.

* `type` - *Optional* - Type of image to cache. Must be one of `container` or
  `virtual-machine`. Defaults to `container`.

* `aliases` - *Optional* - A list of aliases to assign to the image after
	pulling.

* `copy_aliases` - *Optional* - Whether to copy the aliases of the image from
	the remote. Valid values are `true` and `false`. Defaults to `true`.

* `project` - *Optional* - Name of the project where the image will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

* `architecture` - The image architecture (e.g. x86_64, aarch64). See [Architectures](https://linuxcontainers.org/incus/docs/main/architectures/) for all possible values.

## Attribute Reference

The following attributes are exported:

* `created_at` - The datetime of image creation, in Unix time.

* `fingerprint` - The unique hash fingperint of the image.

* `copied_aliases` - The list of aliases that were copied from the
  `source_image`.

## Notes

* See the Incus [documentation](https://linuxcontainers.org/incus/docs/main/howto/images_remote) for more info on default image remotes.
