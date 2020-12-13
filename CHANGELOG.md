## 0.5.0 (Unreleased)

## 0.4.1 (December 12, 2020)

No changes. This release was only created to be published to the Terraform Registry.

## 0.4.0 (November 9, 2020)

IMPROVEMENTS

* Expose `lxd_container` IPv6 address [GH-173](https://github.com/terraform-lxd/terraform-provider-lxd/pull/173)
* Added append to `lxd_container_file` [GH-206](https://github.com/terraform-lxd/terraform-provider-lxd/pull/206)
* Allow updating `lxd_network` in-place [GH-210](https://github.com/terraform-lxd/terraform-provider-lxd/pull/210)
* Added `lxd_publish_image` [GH-209](https://github.com/terraform-lxd/terraform-provider-lxd/pull/209)
* Allow `linux.*` attributes in `lxd_container` [GH-194](https://github.com/terraform-lxd/terraform-provider-lxd/pull/194)

## 0.3.0 (January 27, 2020)

BUG FIXES

* Fixed retrieved storage pool source in `lxd_storage_pool` [GH-181](https://github.com/terraform-lxd/terraform-provider-lxd/pull/181)
* Fixed accidental deletion of `lxd_container` device name [GH-184](https://github.com/terraform-lxd/terraform-provider-lxd/pull/184)

## 0.2.0 (June 20, 2019)

IMPROVEMENTS

* Added new valid device types [GH-150](https://github.com/terraform-lxd/terraform-provider-lxd/pull/150)
* Changed `lxd_container.device` from a list to a set [GH-152](https://github.com/terraform-lxd/terraform-provider-lxd/pull/152)
* Deprecated `lxd_snapshot.creation_date` in favour of `created_at` [GH-155](https://github.com/terraform-lxd/terraform-provider-lxd/pull/155)
