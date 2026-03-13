# Terraform LXD Provider 3.0.0 Upgrade Guide

Version `3.0.0` of the LXD provider for Terraform is a major release that significantly changes
how provider authentication and remote configuration work. Existing configurations **will require
updates** before upgrading.

We strongly advise reviewing the plan produced by Terraform after updating your configuration to
ensure no resources are accidentally removed or altered. If you encounter any unexpected behavior,
please report it by opening a [GitHub issue](https://github.com/terraform-lxd/terraform-provider-lxd/issues/new).

## Why version 3.0.0?

Version `3.0.0` decouples the provider from local machine state. Previously, the provider
implicitly loaded remotes and credentials from the local LXC configuration file
(`~/.config/lxc/config.yml` or `~/snap/lxd/common/config/config.yml`), which led to
non-portable and surprising behavior in CI environments and shared usage scenarios.

The goals of this release are:

- Make provider behavior predictable and portable by default.
- Remove provider's remote configuration and replace it with multiple aliased providers.
  The LXD provider will no longer load remotes from a local LXD configuration.
  Instead, a user needs to define multiple providers, one for each remote.
- Add bearer token authentication.
- Remove trust password and join token support.
- Remove automatic client certificate generation.
- Remove automatic acceptance of remote server certificate.


## Breaking changes

### Provider represents a single remote

The provider no longer manages multiple remotes via nested `remote` blocks. Each provider
instance now represents exactly one LXD server. To work with multiple servers, use Terraform's
[provider aliases](https://developer.hashicorp.com/terraform/language/providers/configuration#alias-multiple-provider-configurations).

**Before:**
```hcl
provider "lxd" {
  remote {
    name    = "local"
    address = "unix://"
    default = true
  }

  remote {
    name    = "prod"
    address = "https://10.1.1.8:8443"
    token   = "abc123"
  }
}

resource "lxd_instance" "app" {
  name   = "app"
  image  = "ubuntu-daily:22.04"
  remote = "prod"
}
```

**After:**
```hcl
provider "lxd" {
  address = "unix://"
}

provider "lxd" {
  alias                          = "prod"
  address                        = "https://10.1.1.8:8443"
  client_certificate_file        = "~/.config/lxc/client.crt"
  client_key_file                = "~/.config/lxc/client.key"
  server_certificate_fingerprint = "<fingerprint>"
}

resource "lxd_instance" "app" {
  provider = lxd.prod
  name     = "app"
  image    = "ubuntu-daily:22.04"
}
```

---

### Removed: `remote` field from all resources and data sources

The per-resource `remote` attribute has been removed. Since each provider instance now
represents a single remote, use the `provider` meta-argument to select which server
a resource belongs to.

**Before:**
```hcl
resource "lxd_instance" "app" {
  name   = "app"
  image  = "ubuntu-daily:22.04"
  remote = "prod"
}

data "lxd_image" "img" {
  name   = "myimage"
  remote = "prod"
}
```

**After:**
```hcl
resource "lxd_instance" "app" {
  provider = lxd.prod
  name     = "app"
  image    = "ubuntu-daily:22.04"
}

data "lxd_image" "img" {
  provider = lxd.prod
  image    = "myimage"
}
```

---

### Removed: `source_image` and `source_remote` on `lxd_cached_image`

The `source_image` and `source_remote` attributes have been replaced by a single `image`
attribute using the format `[<remote>:]<image>`.

**Before:**
```hcl
resource "lxd_cached_image" "img" {
  source_remote = "images"
  source_image  = "ubuntu/22.04"
}
```

**After:**
```hcl
resource "lxd_cached_image" "img" {
  image = "images:ubuntu/22.04"
}
```

---

### Removed: `name` and `fingerprint` on `lxd_image` data source

The `name` and `fingerprint` attributes have been replaced by a single `image`
attribute using the format `[<remote>:]<image>`, where the image part can be
an alias or fingerprint.

**Before:**
```hcl
data "lxd_image" "img" {
  name   = "myimage"
  remote = "images"
}
```

**After:**
```hcl
data "lxd_image" "img" {
  image = "images:myimage"
}
```

---

### Removed: `config_dir`

The `config_dir` attribute has been removed. The provider no longer reads from or writes to the
local LXD configuration directory.

---

### Removed: `generate_client_certificates`

The `generate_client_certificates` provider attribute has been removed. The provider no longer
generates or manages client certificates on disk.

To authenticate using mTLS, generate your client certificate externally and provide it directly
either via the `client_certificate` and `client_key` provider attributes.
Alternatively, the paths to client certificate and corresponding key can be provided via
`client_certificate_file` and `client_key_file`, respectively.

---

### Removed: `remote.token` (join token)

The one-time trust token (`token`) used for initial client registration has been removed because
the provider no longer stores client and server certificates within the local LXD directory, and
storing them in the Terraform state is not an option.

If your remote has not yet established trust, register the client certificate out of band using
the `lxc` CLI before running Terraform:

```bash
lxc remote add myremote https://myhost:8443 --token <token>
```

Once trust is established, provide `client_certificate` and `client_key` in the provider configuration.

---

### Removed: `remote.password` (trust password)

Trust password authentication has been removed from LXD.
While it is still possible to use trust passwords on older LXD versions, we highly discourage their
usage and recommend using TLS authentication instead. On newer LXD versions, bearer tokens can be
used instead.

---

### Removed: `remote.scheme` and `remote.port`

These attributes were deprecated in 2.x and have now been fully removed.
Use a full `address` value instead:

```hcl
address = "https://myhost:8443"
```

---

### Removed: global environment variables

The following global environment variables are no longer supported:

| Removed variable | Replacement |
|---|---|
| `LXD_REMOTE` | Define a separate provider with `alias` |
| `LXD_ADDR` | `LXD_<ALIAS_UPPERCASE>_ADDRESS` |
| `LXD_TOKEN` | `LXD_<ALIAS_UPPERCASE>_BEARER_TOKEN` |
| `LXD_PASSWORD` | Removed (trust passwords no longer supported) |
| `LXD_GENERATE_CLIENT_CERTS` | Removed (certificate generation no longer supported) |
| `LXD_ACCEPT_SERVER_CERTIFICATE` | Use `server_certificate_fingerprint` |

For sensitive information, such as bearer tokens and certificates, we suggest using Terraform
ephemeral variables that prevent Terraform from storing their value in the Terraform state.

```hcl
variable "bearer_token" {
  type      = string
  sensitive = true
  ephemeral = true
}
```

---

## New features

### Server certificate pinning (`server_certificate_fingerprint`)

Instead of blindly accepting or rejecting a server certificate, you can pin the expected
SHA-256 fingerprint of the server's TLS certificate. The provider fetches and verifies the
certificate on first connection:

```hcl
provider "lxd" {
  address                        = "https://myhost:8443"
  server_certificate_fingerprint = "75329c73266fc8f36581ae5508ef4347e4eaa696eed09089c4dc7be68198bf7a"
  client_certificate_file        = "~/.config/lxc/client.crt"
  client_key_file                = "~/.config/lxc/client.key"
}
```

To obtain the fingerprint of an LXD server:

```bash
fingerprint=$(lxc query /1.0 | jq -r .environment.certificate_fingerprint)
```

---

### mTLS with explicit certificate fields

Instead of relying on certificates stored in the local LXD configuration directory,
certificates must now be provided explicitly.

```hcl
provider "lxd" {
  address                        = "https://myhost:8443"
  client_certificate_file        = "~/.config/lxc/client.crt"
  client_key_file                = "~/.config/lxc/client.key"
  server_certificate_fingerprint = "<fingerprint>"
}
```

If providing client certificate and key as raw values, make sure to use ephemeral variables to
prevent Terraform from storing them into state.

```hcl
variable "client_cert" {
  type      = string
  sensitive = true
  ephemeral = true
}

variable "client_key" {
  type      = string
  sensitive = true
  ephemeral = true
}

provider "lxd" {
  address                        = "https://myhost:8443"
  client_certificate             = var.client_cert
  client_key                     = var.client_key
  server_certificate_fingerprint = "<fingerprint>"
}
```

---

### Bearer token authentication

Remotes can now authenticate using a bearer token.
Additionally, bearer tokens issued by LXD server have server certificate fingerprint already encoded,
therefore, it does not have to be explicitly provided.

Make sure to use ephemeral variables when providing sensitive information, such as bearer tokens, to
prevent Terraform from storing them into state.


```hcl
variable "bearer_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

provider "lxd" {
  address      = "https://myhost:8443"
  bearer_token = var.bearer_token
}
```

Alternatively, the bearer token can also be sourced from a file.

```hcl
provider "lxd" {
  address           = "https://myhost:8443"
  bearer_token_file = "/path/to/bearer_token_file"
}
```
