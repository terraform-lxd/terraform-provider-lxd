# lxd_server

Manages LXD server configuration.

~> **Warning:**
  The `lxd_server` resource mutates the global state of an existing LXD server or cluster.
  Avoid using this resource to manage the same keys on the same LXD server from different
  Terraform configurations, as they will likely conflict.

-> **Note:**
  When LXD is clustered, local keys are applied to all members with the same value.

This resource requires the LXD server to support the `metadata_configuration` API extension.

Only the configuration keys explicitly set in `config` are managed by this resource.
Any other server configuration is left untouched.

`config` may contain both global (cluster-wide) configuration keys and member-specific (local) configuration keys.
On a clustered server, local keys are applied to every cluster member with the same value.
On a non-clustered server, local keys apply directly to the single server.

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

## Argument Reference

* `config` - *Optional* - Map of key/value pairs of
	[server config settings](https://documentation.ubuntu.com/lxd/latest/reference/server_settings/).
	May contain both global and member-specific (local) configuration keys.
	On a clustered server, local keys are applied to all cluster members.

* `remote` - *Optional* - The remote in which the resource will be configured.
	If not provided, the provider's default remote will be used.
