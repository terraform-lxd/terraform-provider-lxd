# lxd_project

Manages an LXD project.

## Example Usage

```hcl
resource "lxd_project" "project" {
  name        = "project1"
  description = "Terraform provider example project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.profiles" = false
	"features.storage.buckets" = false
  }
}

resource "lxd_container" "container" {
  name = "container1"
  image = "images:alpine/3.16/amd64"
  project = lxd_project.project.name
}
```

## Argument Reference

* `name` - *Required* - Name of the project. 

* `description` - *Optional* - Description of the project. 

* `config` - *Optional* - Map of key/value pairs of [project config settings](https://documentation.ubuntu.com/lxd/en/latest/reference/projects/).

* `target` - *Optional* - Specify a target node in a cluster. 
