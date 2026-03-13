# terraform-provider-lxd

Use Terraform to manage LXD resources.

## Description

This provider connects to the LXD daemon over a local Unix socket or HTTPS.
Each provider block represents one LXD remote.

The provider can also be used to connect to the SimpleStream server to be able to consume images.

Minimum required LXD version is **`4.0`**.

## Authentication

Each LXD server remote supports exactly one of the following authentication methods:

- **Unix socket** — Used when address starts with `unix://`.

- **Bearer token** — Used for HTTPS addresses when bearer token is provided.

- **mTLS** — Used for HTTPS addresses when the client certificate and key are provided.

## Examples

### Local Unix socket

```hcl
provider "lxd" {
  alias   = "local"
  address = "unix://"
}
```

### mTLS authentication

```hcl
provider "lxd" {
  address                        = "https://10.1.1.8:8443"
  server_certificate_fingerprint = "75329c73266fc8f36581ae5508ef4347e4eaa696eed09089c4dc7be68198bf7a"
  client_certificate_file        = "~/.config/lxc/client.crt"
  client_key_file                = "~/.config/lxc/client.key"
}
```

### Bearer token authentication

```hcl
provider "lxd" {
  address      = "https://10.1.1.8:8443"
  bearer_token = "<bearer_token>"
}
```

### Multiple remotes

```hcl
provider "lxd" {
  alias        = "server"
  address      = "https://example.com:8443"
  bearer_token = "<bearer_token>"
}

provider "lxd" {
  alias    = "images"
  protocol = "simplestreams"
  address  = "https://images.lxd.canonical.com:443"
}
```

### Handling sensitive values

To prevent credentials from being stored in Terraform state, either use **ephemeral Terraform variables** or provide paths to local files containing sensitive information (use fields with `_file` suffix).

Example using ephemeral variable:
```hcl
variable "bearer_token" {
  type      = string
  sensitive = true
  ephemeral = true  # Prevents the value from getting stored in Terraform state.
}

provider "lxd" {
  address      = "https://10.0.0.1:8443"
  bearer_token = var.bearer_token
}
```

Example using local file:
```hcl
provider "lxd" {
  address           = "https://10.0.0.1:8443"
  bearer_token_file = "/path/to/bearer_token_file"
}
```

## Configuration Reference

### Provider arguments

* `address` - *Optional* - Address of the LXD server. Accepts `https://<host>[:<port>]` and `unix://[<path>]`. For `lxd` protocol the port defaults to `8443`. For `simplestreams` it defaults to `443`.

* `protocol` - *Optional* - Remote protocol. One of `lxd` (default) or `simplestreams`.

* `bearer_token` - *Optional* - Bearer token for authentication.

* `bearer_token_file` - *Optional* - Path to a file containing the bearer token for authentication.

* `client_certificate` - *Optional* - PEM-encoded client certificate for mTLS authentication. Requires `client_key` or `client_key_file`.

* `client_certificate_file` - *Optional* - Path to PEM-encoded client certificate for mTLS authentication. Requires `client_key` or `client_key_file`.

* `client_key` - *Optional* - PEM-encoded private key for mTLS authentication.

* `client_key_file` - *Optional* - Path to PEM-encoded private key for mTLS authentication.

* `server_certificate_fingerprint` - *Optional* - SHA-256 fingerprint of the remote server's TLS certificate. It is not required when authenticating using `bearer_token`.
