## 1.10.1 (July 13, 2023)

IMPROVEMENTS

* Added ceph support to `lxd_storage_pool` [GH-308](https://github.com/terraform-lxd/terraform-provider-lxd/pull/308)

BUG FIXES

* Fix `lxd_volume` resources being replaced unnecessarily when upgrading from < 1.10.0 [GH-312](https://github.com/terraform-lxd/terraform-provider-lxd/pull/312)

OTHER

* Updated links for LXD migration from linuxcontainers to canonical [GH-303](https://github.com/terraform-lxd/terraform-provider-lxd/pull/303)

## 1.10.0 (June 12, 2023)

IMPROVEMENTS

* Added `lxd_volume_copy` resource [GH-293](https://github.com/terraform-lxd/terraform-provider-lxd/pull/293)
* Added `content_type` to `lxd_volume` [GH-294](https://github.com/terraform-lxd/terraform-provider-lxd/pull/294)
* Provider is now based on Terraform Plugin SDK v2 [GH-289](https://github.com/terraform-lxd/terraform-provider-lxd/pull/289)
* Provider is now using Terraform Plugin Testing framework [GH-292](https://github.com/terraform-lxd/terraform-provider-lxd/pull/292)
* Set defaults for provider certificate options to avoid unclear type warnings [GH-296](https://github.com/terraform-lxd/terraform-provider-lxd/pull/296)

BUG FIXES

* Fixed a bug where externally deleted instances caused a hard failure requiring manual state intervention [GH-298](https://github.com/terraform-lxd/terraform-provider-lxd/pull/298)

## 1.9.1 (February 26, 2023)

OTHER

* Updating dependencies [GH-284](https://github.com/terraform-lxd/terraform-provider-lxd/pull/284)
* Updating dependencies [GH-284](https://github.com/terraform-lxd/terraform-provider-lxd/pull/286)

## 1.9.0 (December 19, 2022)

IMPROVEMENTS

* Added `lxd_project` resource [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)
* Added `project` to `lxd_cached_image` [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)
* Added `project` to `lxd_container` [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)
* Added `project` to `lxd_container_file` [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)
* Added `project` to `lxd_network` [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)
* Added `project` to `lxd_profile` [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)
* Added `project` to `lxd_publish_image` [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)
* Added `project` to `lxd_snapshot` [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)
* Added `project` to `lxd_storage_pool` [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)
* Added `project` to `lxd_volume` [GH-279](https://github.com/terraform-lxd/terraform-provider-lxd/pull/279)

## 1.8.0 (November 3, 2022)

IMPROVEMENTS

* Replace `GetContainerState` with `GetInstanceState` in `lxd_publish_image` to support VM imaging [GH-276](https://github.com/terraform-lxd/terraform-provider-lxd/pull/276)

## 1.7.3 (November 1, 2022)

BUG FIXES

* Fixed a nil pointer error with `lxd_publish_image` [GH-274](https://github.com/terraform-lxd/terraform-provider-lxd/pull/274)

## 1.7.2 (May 8, 2022)

BUG FIXES

* Fixed a bug to keep device names in profiles [GH-259](https://github.com/terraform-lxd/terraform-provider-lxd/pull/259)

## 1.7.1 (February 8, 2022)

OTHER

* Support for Apple M1 [GH-255](https://github.com/terraform-lxd/terraform-provider-lxd/pull/255)

## 1.7.0 (January 20, 2022)

IMPROVEMENTS
* Added `virtual-machine` support to `lxd_container_snapshot` [GH-248](https://github.com/terraform-lxd/terraform-provider-lxd/pull/248)
* Added `location` to `lxd_volume` [GH-252](https://github.com/terraform-lxd/terraform-provider-lxd/pull/252)

## 1.6.0 (October 7, 2021)

BUG FIXES

* Fixed an issue in `lxd_container` where delete would fail for ephemeral containers [GH-230](https://github.com/terraform-lxd/terraform-provider-lxd/pull/230)

IMPROVEMENTS

* Don't cause `lxd_container` trigger a rebuild when the config changes [GH-227](https://github.com/terraform-lxd/terraform-provider-lxd/pull/227)
* Update the containers configuration when it has changed [GH-240](https://github.com/terraform-lxd/terraform-provider-lxd/pull)

## 1.5.0 (December 15, 2020)

IMPROVEMENTS

* Added `type` to `lxd_container` to enable the choice of either a container or virtual machine to be created [GH-215](https://github.com/terraform-lxd/terraform-provider-lxd/pull/215)
* Allow the `type` in `lxd_network` to be set [GH-220](https://github.com/terraform-lxd/terraform-provider-lxd/pull/220)
* Added `target` to `lxd_network` to help support clustering mode [GH-222](https://github.com/terraform-lxd/terraform-provider-lxd/pull/222)
* Added `target` to `lxd_storage_pool` to help support clustering mode [GH-224](https://github.com/terraform-lxd/terraform-provider-lxd/pull/224)
* Added `target` to `lxd_volume` to help support clustering mode [GH-225](https://github.com/terraform-lxd/terraform-provider-lxd/pull/225)
* Added support for importing `lxd_network` resources [GH-226](https://github.com/terraform-lxd/terraform-provider-lxd/pull/226)

## 1.4.1 (December 12, 2020)

This version bump was primarily to do the initial publication to the Terraform Registry.

IMPROVEMENTS

* Added `target` argument to `lxd_container` to help support clustering mode [GH-212](https://github.com/terraform-lxd/terraform-provider-lxd/pull/212)

## 1.4.0 (November 9, 2020)

IMPROVEMENTS

* Expose `lxd_container` IPv6 address [GH-173](https://github.com/terraform-lxd/terraform-provider-lxd/pull/173)
* Added append to `lxd_container_file` [GH-206](https://github.com/terraform-lxd/terraform-provider-lxd/pull/206)
* Allow updating `lxd_network` in-place [GH-210](https://github.com/terraform-lxd/terraform-provider-lxd/pull/210)
* Added `lxd_publish_image` [GH-209](https://github.com/terraform-lxd/terraform-provider-lxd/pull/209)
* Allow `linux.*` attributes in `lxd_container` [GH-194](https://github.com/terraform-lxd/terraform-provider-lxd/pull/194)

## 1.3.0 (January 27, 2020)

BUG FIXES

* Fixed retrieved storage pool source in `lxd_storage_pool` [GH-181](https://github.com/terraform-lxd/terraform-provider-lxd/pull/181)
* Fixed accidental deletion of `lxd_container` device name [GH-184](https://github.com/terraform-lxd/terraform-provider-lxd/pull/184)

## 1.2.0 (June 20, 2019)

IMPROVEMENTS

* Added new valid device types [GH-150](https://github.com/terraform-lxd/terraform-provider-lxd/pull/150)
* Changed `lxd_container.device` from a list to a set [GH-152](https://github.com/terraform-lxd/terraform-provider-lxd/pull/152)
* Deprecated `lxd_snapshot.creation_date` in favour of `created_at` [GH-155](https://github.com/terraform-lxd/terraform-provider-lxd/pull/155)
