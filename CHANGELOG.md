## 0.0.2 (December 19, 2023)

### New Features 🎉
* Add storage bucket resource by @maveonair in https://github.com/lxc/terraform-provider-incus/pull/5
### Maintenance and Chores 🛠
* build(deps): bump golang.org/x/crypto from 0.16.0 to 0.17.0 in https://github.com/lxc/terraform-provider-incus/pull/11
### Other Changes ❓
* docs: Update README to link to official Terraform registry by @maveonair in https://github.com/lxc/terraform-provider-incus/pull/8
* docs: Refer to instance rather than container by @maveonair in https://github.com/lxc/terraform-provider-incus/pull/9
* docs: Rename LXD to Incus by @maveonair in https://github.com/lxc/terraform-provider-incus/pull/12

## 1.10.3 (October 30, 2023)

### New Features 🎉
* Add load balancer resource by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/360
### Maintenance and Chores 🛠
* build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/341
* build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/345
* build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/350
* build(deps): bump actions/checkout from 3 to 4 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/352
* build(deps): bump crazy-max/ghaction-import-gpg from 5 to 6 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/351
* build(deps): bump goreleaser/goreleaser-action from 4 to 5 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/354
* build(deps): bump golang.org/x/net from 0.13.0 to 0.17.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/357
* build(deps): bump google.golang.org/grpc from 1.57.0 to 1.57.1 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/359
### Other Changes ❓
* Static analysis by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/343
* docs/snapshot: stateful defaults to false by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/347
* Add `remote` attribute to `lxd_project` and deprecate `target` by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/349
* docs: Remove resources section by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/356
* Determine LXD socket before retrieving a client by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/355
* Add support for network zones by @mandrav in https://github.com/terraform-lxd/terraform-provider-lxd/pull/346
* Skip determining unix socket if a remote config address is set by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/358
* Remove extension check from network zones by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/361

## New Contributors
* @mandrav made their first contribution in https://github.com/terraform-lxd/terraform-provider-lxd/pull/346

## 1.10.2 (August 24, 2023)

### New Features 🎉
* Add lxd_instance/lxd_instance_file and deprecate lxd_container/lxd_container_file by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/306
* cached_image: add virtual-machine image alias support by @adamcstephens in https://github.com/terraform-lxd/terraform-provider-lxd/pull/325
* Search for certificates in snap's config directory by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/324
* Use unix as default remote scheme instead of https by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/337
### Bug Fixes 🐝
* Fix not found error checks by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/327
* lxd_volume: fix not found error check by @adamcstephens in https://github.com/terraform-lxd/terraform-provider-lxd/pull/331
### Maintenance and Chores 🛠
* Update dependencies and lxd client rename by @adamcstephens in https://github.com/terraform-lxd/terraform-provider-lxd/pull/317
* fix broken links and update docs for lxd_instance by @adamcstephens in https://github.com/terraform-lxd/terraform-provider-lxd/pull/316
* README: explain how to use the provider built from sources by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/323
* Update links to clustering docs by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/329
* Update actions, go modules, and enable dependabot by @adamcstephens in https://github.com/terraform-lxd/terraform-provider-lxd/pull/314
* update readme and make for testing by @adamcstephens in https://github.com/terraform-lxd/terraform-provider-lxd/pull/332
* Bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/330
* Update Go and Ubuntu CI targets by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/334
* Add acceptance test prechecks  by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/335
* Add LXD version constraint by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/338
* Use test precheck to detect if LXD is in clustered mode by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/339

### New Contributors
* @simondeziel made their first contribution in https://github.com/terraform-lxd/terraform-provider-lxd/pull/323

**Full Changelog**: https://github.com/terraform-lxd/terraform-provider-lxd/compare/v1.10.1...v1.10.2

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
