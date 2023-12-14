# incus_instance_file

Manages a file in an Incus instance.

This resource is useful for managing files on an existing Incus instance.
If you need to preload files in an instance before the instance first
starts, use the `file` block in the `incus_instance` resource.

## Example

```hcl
resource "incus_instance" "instance" {
  name      = "my-instance"
  image     = "ubuntu"
  ephemeral = false
}

resource "incus_instance_file" "file1" {
  instance           = incus_instance.instance.name
  source_path        = "/path/to/local/file"
  target_path        = "/foo/bar.txt"
  create_directories = true
}
```

## Argument Reference

- `instance` - **Required** - Name of the instance.

- `content` - _**Required** unless source_path is used_ - The _contents_ of the file.
  Use the `file()` function to read in the content of a file from disk.

- `source_path` - _**Required** unless content is used_ - The source path to a file to
  copy to the instance.

- `target_path` - **Required** - The absolute path of the file on the instance,
  including the filename.

- `uid` - _Optional_ - The UID of the file. Must be an unquoted integer.
  Defaults to `0`.

- `gid` - _Optional_ - The GID of the file. Must be an unquoted integer.
  Defaults to `0`.

- `mode` - _Optional_ - The octal permissions of the file, must be quoted. Defaults to `0755`.

- `create_directories` - _Optional_ - Whether to create the directories leading
  to the target if they do not exist.

- `append` - _Optional_ - Whether to append the content to the target file. Defaults to false, where target file will be overwritten.

- `project` - _Optional_ - Name of the project where the instance to which this file will be appended exist.

- `remote` - _Optional_ - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

No attributes are exported.
