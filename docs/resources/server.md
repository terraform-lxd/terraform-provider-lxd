# lxd_server

Manages LXD server configuration.

~> **Warning:**
  The `lxd_server` resource mutates the global state of an existing LXD server or cluster.
  Avoid using this resource to manage the same keys on the same LXD server from different
  Terraform configurations, as they will likely conflict.

-> **Note:**
  When LXD is clustered, local keys are applied to all members with the same value,
  unless overridden per member through `member_overrides`.

This resource requires the LXD server to support the `metadata_configuration` API extension.

Only the configuration keys explicitly set in `config` are managed by this resource.
Any other server configuration is left untouched.

`config` may contain both global (cluster-wide) configuration keys and member-specific (local) configuration keys.
On a clustered server, local keys are applied to every cluster member with the same value.
On a non-clustered server, local keys apply directly to the single server.

For clustered servers, per-member values can be set using `member_overrides`, which accepts only local keys.

Removing a key from `config` stops managing it without changing the live value.
To clear a key, set it to an empty string (`""`).

Destroying the resource leaves the live server configuration untouched and only stops Terraform from tracking the keys.

## Example Usage

### Global configuration

```hcl
resource "lxd_server" "global" {
  config = {
    "images.auto_update_interval" = "15"
    "core.https_allowed_origin"   = "*"
  }
}
```

### Per-member configuration

```hcl
resource "lxd_server" "global" {
  # Local key "core.bgp_routerid" is applied to all cluster members, unless overridden.
  config = {
    "core.bgp_asn"      = "65000"
    "core.bgp_routerid" = "127.0.0.1"
  }

  member_overrides = {
    "member-1" = {
      config = {
        "core.bgp_address" = "10.0.0.1:179"
      }
    }

    "member-2" = {
      config = {
        "core.bgp_address" = "10.0.0.2:179"
      }
    }
  }
}
```

## Argument Reference

* `config` - *Optional* - Map of key/value pairs of
	[server config settings](https://documentation.ubuntu.com/lxd/latest/reference/server_settings/).
	May contain both global and default member-specific (local) configuration keys.
	On a clustered server, local keys are applied to all cluster members.

* `member_overrides` - *Optional* - Map of per-member local config overrides for clustered servers.
	Each key is a cluster member name. Each value is an object with a config map of local-scoped keys to apply for that member.
	Values in `member_overrides` take precedence over values from `config`.

* `remote` - *Optional* - The remote in which the resource will be configured.
	If not provided, the provider's default remote will be used.

## Attribute Reference

The following attributes are exported:

* `members` - Map of resolved local config for every cluster member. Used by the provider to
	detect out-of-band changes (drift) on individual cluster members.
