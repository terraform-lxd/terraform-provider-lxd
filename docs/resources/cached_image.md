# incus_cached_image

Manages a locally-stored Incus image.

## Example Usage

```hcl
resource "incus_cached_image" "xenial" {
  source_remote = "ubuntu"
  source_image  = "xenial/amd64"
}

resource "incus_instance" "test1" {
  name      = "test1"
  image     = incus_cached_image.xenial.fingerprint
  ephemeral = false
}
```

## Argument Reference

- `source_image` - **Required** - Fingerprint or alias of image to pull.

- `source_remote` - **Required** - Name of the Incus remote from where image will
  be pulled.

- `type` - _Optional_ - Type of image to cache. Must be one of `container` or
  `virtual-machine`. Defaults to `container`.

- `aliases` - _Optional_ - A list of aliases to assign to the image after
  pulling.

- `copy_aliases` - _Optional_ - Whether to copy the aliases of the image from
  the remote. Valid values are `true` and `false`. Defaults to `true`.

- `project` - _Optional_ - Name of the project where the image will be stored.

- `remote` - _Optional_ - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

- `architecture` - The image architecture (e.g. amd64, i386).

- `created_at` - The datetime of image creation, in Unix time.

- `fingerprint` - The unique hash fingperint of the image.

- `copied_aliases` - The list of aliases that were copied from the
  `source_image`.

## Notes

- See the Incus [documentation](https://documentation.ubuntu.com/incus/en/latest/howto/images_remote) for more info on default image remotes.
