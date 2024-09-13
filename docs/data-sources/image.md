# incus_image

Provides information about an Incus image.

## Example Usage

```hcl
data "incus_image" "debian_custom" {
  name = "debian_custom"
}

resource "incus_instance" "d1" {
  image    = data.incus_image.debian_custom.fingerprint
  name     = "d1"
}
```

## Argument Reference

* `name` - *Optional* - Name of the image.

* `fingerprint` - *Optional* - Fingerprint of the image.

* `type` - *Optional* - Type of image. Must be one of `container` or `virtual-machine`.

* `architecture` - *Optional* - The image architecture (e.g. x86_64, aarch64). See [Architectures](https://linuxcontainers.org/incus/docs/main/architectures/) for all possible values.

* `project` - *Optional* - Name of the project where the image is stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `aliases` - The list of aliases for the image.

* `created_at` - The datetime of image creation, in Unix time.
