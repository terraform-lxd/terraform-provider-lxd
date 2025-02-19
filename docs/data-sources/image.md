# lxd_image

Provides information about an existing LXD image.

## Example Usage

```hcl
data "lxd_image" "debian_custom" {
  name = "debian_custom"
}

resource "lxd_instance" "inst" {
  name  = "my-instance"
  image = data.lxd_image.debian_custom.fingerprint
}
```

## Argument Reference

* `name` - *Optional* - Name of the image.

* `fingerprint` - *Optional* - Fingerprint of the image.

* `type` - *Optional* - Type of image. Must be one of `container` or `virtual-machine`.

* `architecture` - *Optional* - The image architecture (e.g. `x86_64`, `aarch64`). See [Architectures](https://documentation.ubuntu.com/lxd/en/latest/architectures/) for all possible values.

* `project` - *Optional* - Name of the project where the image is stored.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote is used.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `aliases` - The list of aliases for the image.

* `created_at` - The datetime of image creation, in Unix time.
