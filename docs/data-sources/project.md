# lxd_project

Provides information about an existing LXD project.

## Example Usage

```hcl
data "lxd_project" "default" {
  name = "default"
}

resource "lxd_instance" "inst" {
  name    = "my-instance"
  image   = "ubuntu:24.04"
  project = data.lxd_project.default.name
}
```

## Argument Reference

* `name` - **Required** - Name of the project.

## Attribute Reference

* `description` - Description of the project.

* `config` - Map of key/value pairs of [project config settings](https://documentation.ubuntu.com/lxd/latest/reference/projects/).

