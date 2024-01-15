# Terraform LXD Provider 2.0.0 Upgrade Guide

Version `2.0.0` of the LXD provider for Terraform is a major release that includes several changes
requiring updates to existing configuration files.

While we do not anticipate these changes to affect existing resources, we strongly advice reviewing
the plan produced by Terraform to ensure no resources are accidentally removed or altered in an
undesired way. If you encounter any unexpected behavior, please report it by opening a
[GitHub issue](https://github.com/terraform-lxd/terraform-provider-lxd/issues/new).

## Why version 2.0.0?

We introduced version `2.0.0` of the LXD provider in order to upgrade to the
[Terraform Framework](https://developer.hashicorp.com/terraform/plugin/framework), the new SDK for
Terraform. This transition represents a substantial change, justifying the move to a major version
release.

## Removal of deprecated attributes and resources

In addition to the aforementioned SDK upgrade all previously deprecated attributes,
and resources have been removed.

List of removed resources:
- `lxd_container` - Replaced by `lxd_instance`.
- `lxd_container_file` - Replaced by `lxd_instance_file`.
- `lxd_volume_container_attach` - Volumes can now be attached using `lxd_instance.device` block.

List of removed attributes:
- `lxd_instance.privileged` - Use `"security.privileged" = true` in the `lxd_instance.config` block
  as an alternative.
- `lxd_project.target` - Removed as it was unintentionally added - projects do not support target.
- `lxd_provider.refresh_interval` - Removed as it was unintentionally used as an initial delay
   before the first request when waiting for a specific resource state, rather than as an interval
   between multiple requests. Now, the initial interval is set to 2 seconds and is gradually
   increased, up to a maximum of 10 seconds.
- `lxd_snapshot.creation_date` - Replaced by `lxd_snapshot.created_at`.

## Renamed attributes and resources

The major release also provided an opportunity to rename certain attributes for consistency across
the provider.

List of renamed attributes:
- `lxd_instance.file.instance_name` -> `lxd_instance.file.instance`
- `lxd_instance.file.source` -> `lxd_instance.file.source_path`
- `lxd_instance.file.target_file` -> `lxd_instance.file.target_path`
- `lxd_instance.start_on_create` -> `lxd_instance.running`
- `lxd_instance_file.source` -> `lxd_instance_file.source_path`
- `lxd_instance_file.target_file` -> `lxd_instance_file.target_path`
- `lxd_provider.lxd_remote` -> `lxd_provider.remote`
- `lxd_publish_image.container` -> `lxd_publish_image.instance`
- `lxd_snapshot.container_name` -> `lxd_snapshot.instance`

All the above attributes have been simply renamed, with the exception of `lxd_instance.running`.
This attribute now additionally allows the instance to be started or stopped on demand.
