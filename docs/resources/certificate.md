# incus_certificate

Manages an Incus certificate.

## Example Usage

```hcl
resource "incus_certificate" "client1" {
  name        = "client1"
  certificate = file("${path.module}/metrics.crt")
}
```

## Project Restriction Example

```hcl
resource "incus_project" "project1" {
  name = "project1"
}

resource "incus_certificate" "prometheus" {
  name        = "prometheus"
  description = "Prometheus Node Exporter Access"
  restricted  = true
  projects    = [incus_project.project1.name]
  type        = "metrics"
  certificate = file("${path.module}/metrics.crt")
}
```

## Argument Reference

* `name` - **Required** - Name of the certificate.

* `certificate` - **Required** - The certificate.

* `description` - *Optional* - Description of the certificate.

* `type` - *Optional* - The type of certificate to create. Can be one of: client,
  or metrics. If no type is specified, a client certificate is created.

* `projects` - *Optional* -  List of projects to restrict the certificate to.

* `restricted` - *Optional* -  Restrict the certificate to one or more projects.

* `remote` - *Optional* - The remote in which the resource will be created. If
  not provided, the provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

* `fingerprint` - The fingerprint of the certificate.
