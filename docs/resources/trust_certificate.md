# lxd_trust_certificate

The `lxd_trust_certificate` resource allows you to register new client certificates in the LXD trust store.

## Example Usage

```hcl
resource "lxd_trust_certificate" "cert1" {
  name = "cert1"
  path = "/path/to/cert"
}

resource "lxd_trust_certificate" "cert2" {
  name    = "cert2"
  content = <<EOF
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
EOF
}
```

## Argument Reference

* `name` - **Required** - Name of the certificate.

* `content` - *__Required__ unless path is used* - The _contents_ of the certificate. Storing the
        certificate directly in the Terraform configuration as plain text is not recommended. Instead,
        use the `file()` function to read the content from a file on disk, or use the `path` attribute.

* `path` - *__Required__ unless content is used* - The path to a file containing a certificate.

* `projects` - *Optional* - List of projects to restrict the certificate to.

* `remote` - *Optional* - The remote in which the resource will be created. If
	not provided, the provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

* `fingerprint` - The unique hash fingerprint of the certificate.

## Notes

* See the LXD [documentation](https://documentation.ubuntu.com/lxd/en/latest/authentication/#tls-client-certificates) for more information on client certificates.
