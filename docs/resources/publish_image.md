# lxd_publish_image

Create a LXD image from a container

## Example Usage

```hcl
resource "lxd_cached_image" "xenial" {
  source_remote = "ubuntu"
  source_image  = "xenial/amd64"
}

resource "lxd_container" "test1" {
  name      = "test1"
  image     = "${lxd_cached_image.xenial.fingerprint}"
  profiles  = ["default"]

  start_container = false
}

resource "lxd_publish_image" "test1" {
  depends_on = [ lxd_container.test1 ]

  container = "test1"
  aliases   = [ "test1_img" ]
}

```

## Argument Reference

* `container` - *Required* - The name of the container

* `aliases` - *Optional* - A list of aliases to assign to the image 

* `properties` - *Optional* - A map of properties to assign to the image 

* `public` - *Optional* - Whether the image can be downloaded by untrusted users.
	Valid values are `true` and `false`. Defaults to `false`.

* `filename` - *Optional* - Used for export

* `compression_algorithm` - *Optional* - Override the compression algorithm for the image. 
    Valid values are (`bzip2`, `gzip`, `lzma`, `xz` or `none`). Defaults to `gzip`

* `triggers` - *Optional* - A map of arbitrary strings that, when changed, will force the resource to be replaced.

* `project` - *Optional* - Name of the project where the published image will be stored.

## Attribute Reference

The following attributes are exported:

* `fingerprint` - The fingerprint of the published image

* `architecture` - The architecture of the published image

* `created_at` - The creation timestamp of the published image

## Notes

* The container must be stopped
