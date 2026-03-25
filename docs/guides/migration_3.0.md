# Terraform LXD Provider 3.0.0 Upgrade Guide

Version `3.0.0` of the LXD provider for Terraform is a major release that decouples the provider
from local LXC configuration, introduces explicit authentication methods, and removes several
deprecated features.

We strongly advise reviewing the plan produced by Terraform after upgrading to ensure no resources
are accidentally removed or altered in an undesired way. If you encounter any unexpected behavior,
please report it by opening a [GitHub issue](https://github.com/terraform-lxd/terraform-provider-lxd/issues/new).

## Why version 3.0.0?

Previous versions of the provider silently loaded remotes and certificates from the local LXC
configuration directory (`~/snap/lxd/common/config` or `~/.config/lxc`). This made provider
behavior dependent on local machine state, leading to non-portable configurations and surprising
behavior in CI environments.

Version `3.0.0` makes all provider configuration explicit. Every remote must be defined in the
Terraform provider block with its authentication credentials. This makes configurations portable,
predictable, and avoids implicit leakage of authentication material into Terraform state.

Additionally, the new version of the LXD Terraform provider removes the need to define per-member
storage pool and network resources. Instead, a single resource handles cluster-wide provisioning
by detecting LXD cluster members, applying local (member-specific) configuration across all members,
and finally creating the storage pool or network using the global configuration.

Please note that minimum required LXD version was increased from `4.0` to `5.0`.

## Remotes must be explicitly defined

The provider no longer reads remotes from local LXC configuration or falls back to a local Unix
socket when no remote is defined. **At least one remote must be defined** in the provider block.

```hcl
# Implicit local remote (provider version < 3.0):
# Before 3.0:
provider "lxd" {
}

# Explicit remote required (provider version >= 3.0)
provider "lxd" {
  remote {
    name    = "local"
    address = "unix://"
  }
}
```

## Default remote selection

The `default` attribute on remote blocks has been replaced by the top-level `default_remote`
attribute. When only one remote is defined, it is automatically used as the default.

```hcl
# Provider version < 3.0:
provider "lxd" {
  remote {
    name    = "server-1"
    address = "https://10.1.1.8:8443"
    default = true
  }

  remote {
    name    = "server-2"
    address = "https://10.1.2.8:8443"
  }
}

# Provider version >= 3.0:
provider "lxd" {
  default_remote = "server-1"

  remote {
    name    = "server-1"
    address = "https://10.1.1.8:8443"
    # authentication attributes ...
  }

  remote {
    name    = "server-2"
    address = "https://10.1.2.8:8443"
    # authentication attributes ...
  }
}
```

## Authentication methods

The provider now supports two explicit authentication methods for HTTPS remotes:

### Bearer token authentication

The `bearer_token` attribute contains sensitive information.
To prevent storing credentials in Terraform state, we recommend using ephemeral variables:

```hcl
variable "bearer_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

provider "lxd" {
  remote {
    name         = "remote-1"
    address      = "https://10.1.1.8:8443"
    bearer_token = var.bearer_token
  }
}
```

Alternatively, a bearer token can be read from a file:

```hcl
provider "lxd" {
  remote {
    name              = "remote-1"
    address           = "https://10.1.1.8:8443"
    bearer_token_file = "/path/to/token"
  }
}
```

### Explicit TLS authentication

The `client_certificate` and `client_key` attributes contain sensitive information.
To prevent storing credentials in Terraform state, we recommend using ephemeral variables:

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
  remote {
    name                         = "remote-1"
    address                      = "https://10.1.1.8:8443"
    client_certificate           = var.client_cert
    client_key                   = var.client_key
    server_certificate_fingerprint = "abc123..."
  }
}
```

Certificate and key values can also be read from files:

```hcl
provider "lxd" {
  remote {
    name                         = "remote-1"
    address                      = "https://10.1.1.8:8443"
    client_certificate_file      = "/path/to/client.crt"
    client_key_file              = "/path/to/client.key"
    server_certificate_fingerprint = "abc123..."
  }
}
```

Trust token can also be used to insert client certificate into LXD trust store:

```hcl
provider "lxd" {
  remote {
    name                     = "remote-1"
    address                  = "https://10.1.1.8:8443"
    client_certificate_file  = "/path/to/client.crt"
    client_key_file          = "/path/to/client.key"
    trust_token              = "token..."
  }
}
```

Bearer token and mTLS are mutually exclusive — only one method can be used per remote.

## Removed provider attributes

The following top-level provider attributes have been removed:

| Removed Attribute                | Replacement
|----------------------------------|------------
| `config_dir`                     | No replacement. Provider no longer reads local LXC config.
| `generate_client_certificates`   | No replacement. Generate certificates externally and provide them via `client_certificate` and `client_key`.
| `accept_remote_certificate`      | No replacement. Use `server_certificate_fingerprint` to pin and verify the server certificate.

### Removed provider `remote` attributes

The following remote block attributes have been removed:

| Removed Attribute | Replacement
|-------------------|------------
| `default`         | Use the top-level `default_remote` attribute.
| `password`        | No replacement. Trust passwords are insecure and removed from newer LXD releases. Use `bearer_token` or mTLS instead.
| `token`           | Renamed to `trust_token`.
| `port`            | Set the port directly in `address` (e.g., `https://10.1.1.8:8443`).
| `scheme`          | Set the scheme directly in `address` (e.g., `https://...` or `unix://...`).

## Removed environment variables

The following environment variables are no longer supported:

| Removed Variable                 |
|----------------------------------|
| `LXD_REMOTE`                     |
| `LXD_ADDR`                       |
| `LXD_PASSWORD`                   |
| `LXD_TOKEN`                      |
| `LXD_SCHEME`                     |
| `LXD_PORT`                       |
| `LXD_GENERATE_CLIENT_CERTS`      |
| `LXD_ACCEPT_SERVER_CERTIFICATE`  |

## Instance `limits` merged into `config`

The `limits` attribute on `lxd_instance` has been removed. Resource limits should now be specified
directly in the `config` map using the `limits.` prefix.

```hcl
# Provider version < 3.0:
resource "lxd_instance" "instance1" {
  name  = "instance1"
  image = "ubuntu-daily:26.04"

  limits = {
    cpu    = 2
    memory = "128MiB"
  }
}

# Provider version >= 3.0:
resource "lxd_instance" "instance1" {
  name  = "instance1"
  image = "ubuntu-daily:26.04"

  config = {
    "limits.cpu"    = 2
    "limits.memory" = "128MiB"
  }
}
```

## Instance `wait_for_network` replaced by `wait_for`

The boolean `wait_for_network` attribute on `lxd_instance` has been replaced by the new `wait_for`
block, which provides more granular control over what conditions to wait for after an instance starts.

The `wait_for` block supports the following types:
- `agent` — Wait for the LXD agent to start (virtual machines only).
- `delay` — Wait for a specified duration (requires the `delay` attribute).
- `ipv4` — Wait for a global IPv4 address (optionally on a specific `nic`).
- `ipv6` — Wait for a global IPv6 address (optionally on a specific `nic`).
- `ready` — Wait for the instance to report a *Ready* status.

Multiple `wait_for` blocks can be specified to wait for several conditions.

```hcl
# Provider version < 3.0:
resource "lxd_instance" "instance1" {
  name             = "instance1"
  image            = "ubuntu-daily:26.04"
  wait_for_network = true
}

# Provider version >= 3.0 (wait for IPv4 address):
resource "lxd_instance" "instance2" {
  name  = "instance2"
  image = "ubuntu-daily:26.04"

  wait_for {
    type = "ipv4"
  }
}

# Provider version >= 3.0 (wait for IPv4 on a specific interface):
resource "lxd_instance" "instance3" {
  name  = "instance3"
  image = "ubuntu-daily:26.04"

  wait_for {
    type = "ipv4"
    nic  = "eth0"
  }
}

# Provider version >= 3.0 (wait for both IPv4 and IPv6):
resource "lxd_instance" "instance4" {
  name  = "instance4"
  image = "ubuntu-daily:26.04"

  wait_for {
    type = "ipv4"
  }

  wait_for {
    type = "ipv6"
  }
}

# Provider version >= 3.0 (wait for LXD agent to start within a virtual machine):
resource "lxd_instance" "instance5" {
  name  = "instance5"
  type  = "virtual-machine"
  image = "ubuntu-daily:26.04"

  wait_for {
    type = "agent"
  }
}
```

## Unified resource for managing local LXD images

Image workflows are now unified under a single `lxd_image` resource, using different configuration blocks for each action.

Use the `source_image` block to copy (cache) an image from a remote source.
This replaces the previous `lxd_cached_image` resource.

```hcl
# Provider version < 3.0:
resource "lxd_cached_image" "xenial" {
  source_remote = "ubuntu"
  source_image  = "xenial/amd64"
  copy_aliases  = true
}

# Provider version >= 3.0:
resource "lxd_image" "xenial" {
  source_image = {
    image        = "ubuntu:xenial/amd64"
    copy_aliases = true
  }
}
```

Use the `source_instance` block to publish an image from an instance or instance snapshot.
This replaces the previous `lxd_publish_image` resource.

```hcl
# Provider version < 3.0:
resource "lxd_publish_image" "test1" {
  instance = lxd_instance.test1.name
  aliases  = ["test1_img"]
}

# Provider version >= 3.0:
resource "lxd_image" "test1" {
  source_instance = {
    name = lxd_instance.test1.name
  }

  aliases = ["test1_img"]
}
```

## Unified cluster-wide storage pool configuration

The `lxd_storage_pool` field `target` has been removed.

Before 3.0, clustered storage pools required one global resource with one targeted
resource per cluster member for local keys.

Now, a single `lxd_storage_pool` resource definition is used for the cluster-wide pool configuration
with explicit per-member local overrides.

Use `config` for global storage pool keys and default values for local keys.
To override member-specific local keys use `member_overrides`.

```hcl
# Provider version < 3.0 (global + targeted member resources)
resource "lxd_storage_pool" "pool_member_1" {
  name   = "mypool"
  driver = "zfs"
  target = "member-1"

  config = {
    source = "/dev/sda"
  }
}

resource "lxd_storage_pool" "pool_member_2" {
  name   = "mypool"
  driver = "zfs"
  target = "member-2"

  config = {
    source = "/dev/sdb"
  }
}

resource "lxd_storage_pool" "pool_global" {
  name   = "mypool"
  driver = "zfs"

  # Global definition created after per-member local definitions.
  depends_on = [
    lxd_storage_pool.pool_member_1,
    lxd_storage_pool.pool_member_2,
  ]
}

# Provider version >= 3.0
resource "lxd_storage_pool" "pool" {
  name   = "mypool"
  driver = "zfs"

  # Global keys and default local values.
  config = {
    source = "/dev/sda"
  }

  # Optionally, override per-member local configuration.
  member_overrides = {
    "member-2" = {
      config = {
        source = "/dev/sdb"
      }
    }
  }
}
```

## Unified cluster-wide network configuration

Similar to the storage pool configuration, `lxd_network` configuration field `target` has been removed.

Before 3.0, clustered network required one global resource with one targeted
resource per cluster member for local keys.

Now, a single `lxd_network` resource definition is used for the cluster-wide network configuration
with explicit per-member local overrides.

Use `config` for global network keys and default values for local keys.
To override member-specific local keys use `member_overrides`.

```hcl
# Provider version < 3.0 (global + targeted member resources)
resource "lxd_network" "network_member_1" {
  name   = "mynetwork"
  type   = "bridge"
  target = "member-1"

  config = {
    "bridge.external_interfaces" = "eth0"
  }
}

resource "lxd_network" "network_member_2" {
  name   = "mynetwork"
  type   = "bridge"
  target = "member-2"

  config = {
    "bridge.external_interfaces" = "eth1"
  }
}

resource "lxd_network" "network_global" {
  name   = "mynetwork"
  type   = "bridge"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat"     = "true"
  }

  # Global definition created after per-member local definitions.
  depends_on = [
    lxd_network.network_member_1,
    lxd_network.network_member_2,
  ]
}

# Provider version >= 3.0
resource "lxd_network" "net" {
  name = "mynetwork"
  type = "bridge"

  # Global keys and default local values.
  config = {
    "ipv4.address"               = "10.150.19.1/24"
    "ipv4.nat"                   = "true"
    "bridge.external_interfaces" = "eth0"
  }

  # Optionally, override per-member local configuration.
  member_overrides = {
    "member-2" = {
      config = {
        "bridge.external_interfaces" = "eth1"
      }
    }
  }
}
```

## Renamed resources

The following resources have been renamed:

| Old name          | New name                  |
| ----------------- | ------------------------- |
| `lxd_volume`      | `lxd_storage_volume`      |
| `lxd_volume_copy` | `lxd_storage_volume_copy` |
| `lxd_snapshot`    | `lxd_instance_snapshot`   |


## Migration checklist

1. Add explicit `remote` blocks for all LXD servers you connect to.
2. Replace `default = true` with the top-level `default_remote` attribute.
3. Remove `generate_client_certificates` and provide client certificates as remote arguments.
4. Remove `accept_remote_certificate` and provide remote server certificate fingerprint (`lxc info` show certificate fingerprint).
5. Replace `password` with mTLS or bearer tokens.
6. Remove `port` and `scheme` — set them directly in `address`.
7. Replace any `LXD_REMOTE`, `LXD_ADDR`, `LXD_PASSWORD`, `LXD_TOKEN` environment variables with explicit provider configuration.
8. Move instance `limits` into `config` using the `limits.` prefix (e.g., `"limits.cpu" = 2`).
9. Replace `wait_for_network` with the appropriate `wait_for` block (e.g., `wait_for { type = "ipv4" }`).
10. For clustered `lxd_network`, replace targeted per-member resources with a single network resource. Keep default local values in `config`, and use `member_overrides` for members that differ.
11. For clustered `lxd_storage_pool`, replace targeted per-member resources with a single pool resource. Keep default local values in `config`, and use `member_overrides` for members that differ.
12. Move `source` into `config` in `lxd_storage_pool` resources.
13. Replace image resources:
    -  `lxd_cached_image` with `lxd_image` using the `source_image` block
    -  `lxd_publish_image` with `lxd_image` using the `source_instance` block
14. Rename resources:
    -  `lxd_volume` to `lxd_storage_volume`
    -  `lxd_volume_copy` to `lxd_storage_volume_copy`
    -  `lxd_snapshot` to `lxd_instance_snapshot`
15. Run `terraform plan` and verify no unexpected changes.
