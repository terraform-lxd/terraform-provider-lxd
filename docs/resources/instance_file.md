# lxd_instance_file

Manages a file in an LXD instance.

This resource is useful for managing files on an existing LXD instance.
If you need to preload files in an instance before the instance first
starts, use the `file` block in the `lxd_instance` resource.

## Example

```hcl
resource "lxd_instance" "instance" {
  name      = "my-instance"
  image     = "ubuntu"
  ephemeral = false
}

resource "lxd_instance_file" "file1" {
  instance           = lxd_instance.instance.name
  target_file        = "/foo/bar.txt"
  source             = "/path/to/local/file"
  create_directories = true
}
```

## Argument Reference

* `instance` - **Required** - Name of the instance.

* `target_file` - **Required** - The absolute path of the file on the instance,
	including the filename.

* `content` - *__Required__ unless source is used* - The _contents_ of the file.
	Use the `file()` function to read in the content of a file from disk.

* `source` - *__Required__ unless content is used* The source path to a file to
	copy to the instance.

* `uid` - *Optional* - The UID of the file. Must be an unquoted integer.
  Defaults to `0`.

* `gid` - *Optional* - The GID of the file. Must be an unquoted integer.
  Defaults to `0`.

* `mode` - *Optional* - The octal permissions of the file, must be quoted. Defaults to `0755`.

* `create_directories` - *Optional* - Whether to create the directories leading
	to the target if they do not exist.

* `append` - *Optional* - Whether to append the content to the target file. Defaults to false, where target file will be overwritten.

* `project` - *Optional* - Name of the project where the instance to which this file will be appended exist.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.


## Attribute Reference

No attributes are exported.
