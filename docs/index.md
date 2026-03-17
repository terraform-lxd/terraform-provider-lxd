# LXD Provider

The LXD provider allows infrastructure as code tools to manage resources on [LXD](https://documentation.ubuntu.com/lxd/latest/) servers, such as instances, networks, storage, and more.

LXD is a modern, secure, and powerful system container and virtual machine manager.
If you are new to LXD, see the [Getting started guide](https://documentation.ubuntu.com/lxd/latest/tutorial/first_steps/) in the official documentation.

Minimum supported LXD version is **4.0**.

## Getting Started

### Prerequisites

1. A running LXD server. See [How to install LXD](https://documentation.ubuntu.com/lxd/latest/installing/).
2. An installed infrastructure as code tool, such as [Terraform](https://developer.hashicorp.com/terraform/install).
3. Authentication credentials for connecting to your LXD server (see [Authentication](#authentication) below).

### Minimal Example

The following configuration launches an Ubuntu container on a local LXD server:

```hcl
terraform {
  required_providers {
    lxd = {
      source = "terraform-lxd/lxd"
    }
  }
}

provider "lxd" {
  remote {
    name    = "local"
    address = "unix://"
  }
}

resource "lxd_instance" "my_container" {
  name  = "my-container"
  image = "ubuntu-daily:24.04"
}
```

Save this as `main.tf`, then:

```shell
terraform init
terraform apply
```

## Provider Configuration

The provider connects to the LXD daemon via local Unix socket or HTTPS.

All LXD remotes used by the provider must be explicitly defined in the provider configuration.
The LXD built-in image remotes (such as `ubuntu:` and `images:`) are predefined and do not need to be manually configured.
For more information on image remotes, see [Remote image servers](https://documentation.ubuntu.com/lxd/latest/reference/remote_image_servers/).

### Authentication

LXD supports multiple authentication methods. See [Remote API authentication](https://documentation.ubuntu.com/lxd/latest/authentication/) in the LXD documentation for background.

This provider supports the following methods:

- **Bearer token** - For remote servers that support API extension `auth_bearer`. LXD bearer tokens also embed the server certificate fingerprint, so `server_certificate_fingerprint` does not need to be set separately.
- **Mutual TLS (mTLS)** - Client certificate authentication. Requires a client certificate that is already trusted by the server, or a trust token to bootstrap trust on the first connection.
- **Unix socket** - For local connections. Requires access to the local LXD unix socket.

#### Handling Sensitive Information

When providing sensitive values, such as tokens or certificates, directly in HCL, use [ephemeral variables](https://developer.hashicorp.com/terraform/language/manage-sensitive-data#omit-values-from-state-and-plan-files) to prevent them from being stored in Terraform state:

```hcl
variable "bearer_token" {
  type      = string
  ephemeral = true
}

provider "lxd" {
  remote {
    name         = "lxd-server-1"
    address      = "https://10.1.1.8:8443"
    bearer_token = var.bearer_token
  }
}
```

Alternatively, the provider can source sensitive values from local files using the `*_file` variants (e.g. `bearer_token_file`, `client_certificate_file`, `client_key_file`).

#### Unix Socket

Connect to a local LXD server via Unix socket.
Setting the remote address to `unix://` instructs the provider to search for a local LXD Unix socket in the standard locations.

```hcl
provider "lxd" {
  remote {
    name    = "local"
    address = "unix://"
  }
}
```

#### Bearer Token Authentication

Authenticate with an LXD server using a bearer token.
See [Bearer token authentication](https://documentation.ubuntu.com/lxd/latest/authentication/#bearer-token-authentication) for background and setup instructions.

```hcl
variable "bearer_token" {
  type      = string
  ephemeral = true
}

provider "lxd" {
  remote {
    name         = "lxd-server-1"
    address      = "https://10.1.1.8:8443"
    bearer_token = var.bearer_token
  }
}
```

#### Mutual TLS Authentication

Provide the client certificate and key. The client certificate must already be [trusted by the LXD server](https://documentation.ubuntu.com/lxd/latest/authentication/#tls-client-certificates).

```hcl
provider "lxd" {
  remote {
    name                           = "lxd-server-1"
    address                        = "https://10.1.1.8:8443"
    client_certificate_file        = "/path/to/client.crt"
    client_key_file                = "/path/to/client.key"
    server_certificate_fingerprint = "7dc4ebe...37e7bfbe"
  }
}
```

If the server certificate is self-signed or not otherwise trusted by the client, set `server_certificate_fingerprint` so the provider can verify the server identity. Retrieve the fingerprint with `lxc info` or by calling the LXD `/1.0` API endpoint.

##### Bootstrap mTLS Using a Trust Token

For a first-time connection, a [trust token](https://documentation.ubuntu.com/lxd/latest/howto/server_expose/#authenticate-with-the-lxd-server) can bootstrap trust. The token allows the server to add the client certificate to its trust store automatically, after which subsequent connections use mTLS.

```hcl
provider "lxd" {
  remote {
    name                    = "lxd-server-1"
    address                 = "https://10.1.1.8:8443"
    client_certificate_file = "/path/to/client.crt"
    client_key_file         = "/path/to/client.key"
    trust_token             = "eyJjbGllbn...GUiOiIifQ=="
  }
}
```

### Multiple Remotes

When defining multiple remotes, set `default_remote` to specify which remote is used when one is not specified in a resource:

```hcl
provider "lxd" {
  default_remote = "lxd-server-1"

  remote {
    name         = "lxd-server-1"
    address      = "https://10.0.21.10:8443"
    bearer_token = var.lxd_token_1
  }

  remote {
    name         = "lxd-server-2"
    address      = "https://10.0.42.10:8443"
    bearer_token = var.lxd_token_2
  }
}
```

When only one remote is defined, it is automatically used as the default remote.

## Configuration Reference

### Provider Arguments

* `remote` - **Required** - Defines a LXD or simplestreams remote the provider can use. At least one remote must be defined. See the `remote` block reference below.

* `default_remote` - *Optional* - Name of the default LXD remote to use when no remote is specified in a resource. Required when two or more remotes are defined.

### `remote` Block

* `name` - **Required** - The name of the remote.

* `address` - **Required** - The remote address. Must start with `https://` for HTTPS connections or `unix://` for Unix socket connections.

* `protocol` - *Optional* - The protocol of remote server (`lxd` or `simplestreams`). Defaults to `lxd`.

* `bearer_token` - *Optional* - Bearer token for authentication.

* `bearer_token_file` - *Optional* - Path to a file containing the bearer token.

* `client_certificate` - *Optional* - PEM-encoded client certificate for mTLS authentication. Must be provided together with `client_key` or `client_key_file`.

* `client_certificate_file` - *Optional* - Path to the PEM-encoded client certificate file. Must be provided together with `client_key` or `client_key_file`.

* `client_key` - *Optional* - PEM-encoded private key for mTLS authentication. Must be provided together with `client_certificate` or `client_certificate_file`.

* `client_key_file` - *Optional* - Path to the PEM-encoded private key file. Must be provided together with `client_certificate` or `client_certificate_file`.

* `server_certificate_fingerprint` - *Optional* - SHA-256 fingerprint of the remote server's TLS certificate. Used to pin and verify the server certificate.

* `trust_token` - *Optional* - Trust token for adding the client certificate to the server's trust store on first connection. Used together with `client_certificate`/`client_certificate_file` and `client_key`/`client_key_file`.
