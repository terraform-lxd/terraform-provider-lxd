# lxd_container_file

Manages a file in an LXD container.

This resource is useful for managing files on an existing LXD container.
If you need to preload files in a container before the container first
starts, use the `file` block in the `lxd_container` resource.

## Example

```hcl
resource "lxd_container" "test1" {
  name      = "test1"
  image     = "ubuntu"
  ephemeral = false
}

resource "lxd_container_file" "file1" {
  container_name     = "${lxd_container.test1.name}"
  target_file        = "/foo/bar.txt"
  source             = "/path/to/local/file"
  create_directories = true
```

## Argument Reference

* `remote` - *Optional* - The remote in which the resource will be created. If
	it is not provided, the default provider remote is used.

* `container_name` - *Required* - Name of the container.

* `content` - *Required unless source is used* - The _contents_ of the file.
	Use the `file()` function to read in the content of a file from disk.

* `source` - *Required unless content is used* The source path to a file to
	copy to the container.

* `target_file` - *Required* - The absolute path of the file on the container,
	including the filename.

* `uid` - *Optional* - The UID of the file. Must be an unquoted integer.
  Defaults to `0`.

* `gid` - *Optional* - The GID of the file. Must be an unquoted integer.
  Defaults to `0`.

* `mode` - *Optional* - The octal permissions of the file, must be quoted.

* `create_directories` - *Optional* - Whether to create the directories leading
	to the target if they do not exist.
