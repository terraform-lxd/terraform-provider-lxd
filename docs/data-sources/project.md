# incus_project

Provides information about an Incus project.

## Example Usage

```hcl
data "incus_project" "default" {
  name = "default"
}

resource "incus_instance" "d1" {
  project = data.incus_project.default.name
  image    = "images:debian/12"
  name     = "d1"
}
```

## Argument Reference

* `name` - **Required** - Name of the project.

* `remote` - *Optional* - The remote in which the resource was created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

* `description` - Description of the project.

* `config` - Map of key/value pairs of
  [instance config settings](https://linuxcontainers.org/incus/docs/main/reference/instance_options/).

