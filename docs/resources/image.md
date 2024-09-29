# incus_image

Manages a locally-stored Incus image.

## Example Usage

```hcl
resource "incus_image" "alpine" {
  source_image = {
    remote = "images"
    name   = "alpine/edge"
  }
}

resource "incus_instance" "test1" {
  name      = "test1"
  image     = incus_image.alpine.fingerprint
  ephemeral = false
}
```

## Argument Reference

* `source_image` - *Optional* - The source image from which the image will be created. See reference below.

* `source_instance` - *Optional* - The source instance from which the image will be created. See reference below.

* `aliases` - *Optional* - A list of aliases to assign to the image after
	pulling.

* `project` - *Optional* - Name of the project where the image will be stored.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

The `source_image` block supports:

* `remote` - **Required** - The remote where the image will be pulled from.

* `name` - **Required** - Name of the image.

* `type` - *Optional* - Type of image to cache. Must be one of `container` or
  `virtual-machine`. Defaults to `container`.

* `architecture` - *Optional* - The image architecture (e.g. x86_64, aarch64). See [Architectures](https://linuxcontainers.org/incus/docs/main/architectures/) for all possible values.

* `copy_aliases` - *Optional* - Whether to copy the aliases of the image from
  the remote. Valid values are `true` and `false`. Defaults to `true`.

The `source_instance` block supports:

* `name` - **Required** - Name of the source instance.

* `snapshot`- *Optional* - Name of the snapshot of the source instance

## Attribute Reference

The following attributes are exported:

* `created_at` - The datetime of image creation, in Unix time.

* `fingerprint` - The unique hash fingperint of the image.

* `copied_aliases` - The list of aliases that were copied from the
  `source_image`.

## Notes

* See the Incus [documentation](https://linuxcontainers.org/incus/docs/main/howto/images_remote) for more info on default image remotes.
