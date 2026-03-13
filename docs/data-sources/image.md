# lxd_image

Provides information about an existing LXD image.

## Example Usage

```hcl
data "lxd_image" "debian_custom" {
  image = "debian_custom"
}

resource "lxd_instance" "inst" {
  name  = "my-instance"
  image = data.lxd_image.debian_custom.fingerprint
}
```

## Argument Reference

* `image` - *Optional* - Image to look up, specified as `[<remote>:]<image>`.
  The remote is the name of a remote image server (e.g. `images`). If no remote
  is specified, the image is looked up on the provider's server. The image part
  can be an alias or fingerprint.

* `type` - *Optional* - Type of image. Must be one of `container` or `virtual-machine`.

* `architecture` - *Optional* - The image architecture (e.g. `x86_64`, `aarch64`). See [Architectures](https://documentation.ubuntu.com/lxd/latest/architectures/) for all possible values.

* `project` - *Optional* - Name of the project where the image is stored.

## Attribute Reference

This data source exports the following attributes in addition to the arguments above:

* `aliases` - The list of aliases for the image.

* `created_at` - The datetime of image creation, in Unix time.

* `fingerprint` - The unique hash fingerprint of the image.
