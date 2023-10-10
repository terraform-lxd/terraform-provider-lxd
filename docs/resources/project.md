# lxd_project

Manages an LXD project.

## Example Usage

```hcl
resource "lxd_project" "project" {
  name        = "project1"
  description = "Terraform provider example project"
  config = {
    "features.storage.volumes" = false
    "features.images"          = false
    "features.profiles"        = false
    "features.storage.buckets" = false
  }
}

resource "lxd_instance" "container" {
  name    = "container1"
  image   = "images:alpine/3.16/amd64"
  project = lxd_project.project.name
}
```

## Argument Reference

* `name` - *Required* - Name of the project. 

* `description` - *Optional* - Description of the project. 

* `config` - *Optional* - Map of key/value pairs of [project config settings](https://documentation.ubuntu.com/lxd/en/latest/reference/projects/).

* `remote` - *Optional* - The remote in which the resource will be created. If
    it is not provided, the default provider remote is used.
