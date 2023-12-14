## 1.10.3 (October 30, 2023)

### New Features 🎉

- Add load balancer resource by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/360

### Maintenance and Chores 🛠

- build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-incus/terraform-provider-incus/pull/341
- build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-incus/terraform-provider-incus/pull/345
- build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-incus/terraform-provider-incus/pull/350
- build(deps): bump actions/checkout from 3 to 4 by @dependabot in https://github.com/terraform-incus/terraform-provider-incus/pull/352
- build(deps): bump crazy-max/ghaction-import-gpg from 5 to 6 by @dependabot in https://github.com/terraform-incus/terraform-provider-incus/pull/351
- build(deps): bump goreleaser/goreleaser-action from 4 to 5 by @dependabot in https://github.com/terraform-incus/terraform-provider-incus/pull/354
- build(deps): bump golang.org/x/net from 0.13.0 to 0.17.0 by @dependabot in https://github.com/terraform-incus/terraform-provider-incus/pull/357
- build(deps): bump google.golang.org/grpc from 1.57.0 to 1.57.1 by @dependabot in https://github.com/terraform-incus/terraform-provider-incus/pull/359

### Other Changes ❓

- Static analysis by @simondeziel in https://github.com/terraform-incus/terraform-provider-incus/pull/343
- docs/snapshot: stateful defaults to false by @simondeziel in https://github.com/terraform-incus/terraform-provider-incus/pull/347
- Add `remote` attribute to `incus_project` and deprecate `target` by @simondeziel in https://github.com/terraform-incus/terraform-provider-incus/pull/349
- docs: Remove resources section by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/356
- Determine Incus socket before retrieving a client by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/355
- Add support for network zones by @mandrav in https://github.com/terraform-incus/terraform-provider-incus/pull/346
- Skip determining unix socket if a remote config address is set by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/358
- Remove extension check from network zones by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/361

## New Contributors

- @mandrav made their first contribution in https://github.com/terraform-incus/terraform-provider-incus/pull/346

## 1.10.2 (August 24, 2023)

### New Features 🎉

- Add incus_instance/incus_instance_file and deprecate incus_container/incus_container_file by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/306
- cached_image: add virtual-machine image alias support by @adamcstephens in https://github.com/terraform-incus/terraform-provider-incus/pull/325
- Search for certificates in snap's config directory by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/324
- Use unix as default remote scheme instead of https by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/337

### Bug Fixes 🐝

- Fix not found error checks by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/327
- incus_volume: fix not found error check by @adamcstephens in https://github.com/terraform-incus/terraform-provider-incus/pull/331

### Maintenance and Chores 🛠

- Update dependencies and incus client rename by @adamcstephens in https://github.com/terraform-incus/terraform-provider-incus/pull/317
- fix broken links and update docs for incus_instance by @adamcstephens in https://github.com/terraform-incus/terraform-provider-incus/pull/316
- README: explain how to use the provider built from sources by @simondeziel in https://github.com/terraform-incus/terraform-provider-incus/pull/323
- Update links to clustering docs by @simondeziel in https://github.com/terraform-incus/terraform-provider-incus/pull/329
- Update actions, go modules, and enable dependabot by @adamcstephens in https://github.com/terraform-incus/terraform-provider-incus/pull/314
- update readme and make for testing by @adamcstephens in https://github.com/terraform-incus/terraform-provider-incus/pull/332
- Bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-incus/terraform-provider-incus/pull/330
- Update Go and Ubuntu CI targets by @simondeziel in https://github.com/terraform-incus/terraform-provider-incus/pull/334
- Add acceptance test prechecks by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/335
- Add Incus version constraint by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/338
- Use test precheck to detect if Incus is in clustered mode by @MusicDin in https://github.com/terraform-incus/terraform-provider-incus/pull/339

### New Contributors

- @simondeziel made their first contribution in https://github.com/terraform-incus/terraform-provider-incus/pull/323

**Full Changelog**: https://github.com/terraform-incus/terraform-provider-incus/compare/v1.10.1...v1.10.2

## 1.10.1 (July 13, 2023)

IMPROVEMENTS

- Added ceph support to `incus_storage_pool` [GH-308](https://github.com/terraform-incus/terraform-provider-incus/pull/308)

BUG FIXES

- Fix `incus_volume` resources being replaced unnecessarily when upgrading from < 1.10.0 [GH-312](https://github.com/terraform-incus/terraform-provider-incus/pull/312)

OTHER

- Updated links for Incus migration from linuxcontainers to canonical [GH-303](https://github.com/terraform-incus/terraform-provider-incus/pull/303)

## 1.10.0 (June 12, 2023)

IMPROVEMENTS

- Added `incus_volume_copy` resource [GH-293](https://github.com/terraform-incus/terraform-provider-incus/pull/293)
- Added `content_type` to `incus_volume` [GH-294](https://github.com/terraform-incus/terraform-provider-incus/pull/294)
- Provider is now based on Terraform Plugin SDK v2 [GH-289](https://github.com/terraform-incus/terraform-provider-incus/pull/289)
- Provider is now using Terraform Plugin Testing framework [GH-292](https://github.com/terraform-incus/terraform-provider-incus/pull/292)
- Set defaults for provider certificate options to avoid unclear type warnings [GH-296](https://github.com/terraform-incus/terraform-provider-incus/pull/296)

BUG FIXES

- Fixed a bug where externally deleted instances caused a hard failure requiring manual state intervention [GH-298](https://github.com/terraform-incus/terraform-provider-incus/pull/298)

## 1.9.1 (February 26, 2023)

OTHER

- Updating dependencies [GH-284](https://github.com/terraform-incus/terraform-provider-incus/pull/284)
- Updating dependencies [GH-284](https://github.com/terraform-incus/terraform-provider-incus/pull/286)

## 1.9.0 (December 19, 2022)

IMPROVEMENTS

- Added `incus_project` resource [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)
- Added `project` to `incus_cached_image` [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)
- Added `project` to `incus_container` [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)
- Added `project` to `incus_container_file` [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)
- Added `project` to `incus_network` [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)
- Added `project` to `incus_profile` [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)
- Added `project` to `incus_publish_image` [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)
- Added `project` to `incus_snapshot` [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)
- Added `project` to `incus_storage_pool` [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)
- Added `project` to `incus_volume` [GH-279](https://github.com/terraform-incus/terraform-provider-incus/pull/279)

## 1.8.0 (November 3, 2022)

IMPROVEMENTS

- Replace `GetContainerState` with `GetInstanceState` in `incus_publish_image` to support VM imaging [GH-276](https://github.com/terraform-incus/terraform-provider-incus/pull/276)

## 1.7.3 (November 1, 2022)

BUG FIXES

- Fixed a nil pointer error with `incus_publish_image` [GH-274](https://github.com/terraform-incus/terraform-provider-incus/pull/274)

## 1.7.2 (May 8, 2022)

BUG FIXES

- Fixed a bug to keep device names in profiles [GH-259](https://github.com/terraform-incus/terraform-provider-incus/pull/259)

## 1.7.1 (February 8, 2022)

OTHER

- Support for Apple M1 [GH-255](https://github.com/terraform-incus/terraform-provider-incus/pull/255)

## 1.7.0 (January 20, 2022)

IMPROVEMENTS

- Added `virtual-machine` support to `incus_container_snapshot` [GH-248](https://github.com/terraform-incus/terraform-provider-incus/pull/248)
- Added `location` to `incus_volume` [GH-252](https://github.com/terraform-incus/terraform-provider-incus/pull/252)

## 1.6.0 (October 7, 2021)

BUG FIXES

- Fixed an issue in `incus_container` where delete would fail for ephemeral containers [GH-230](https://github.com/terraform-incus/terraform-provider-incus/pull/230)

IMPROVEMENTS

- Don't cause `incus_container` trigger a rebuild when the config changes [GH-227](https://github.com/terraform-incus/terraform-provider-incus/pull/227)
- Update the containers configuration when it has changed [GH-240](https://github.com/terraform-incus/terraform-provider-incus/pull)

## 1.5.0 (December 15, 2020)

IMPROVEMENTS

- Added `type` to `incus_container` to enable the choice of either a container or virtual machine to be created [GH-215](https://github.com/terraform-incus/terraform-provider-incus/pull/215)
- Allow the `type` in `incus_network` to be set [GH-220](https://github.com/terraform-incus/terraform-provider-incus/pull/220)
- Added `target` to `incus_network` to help support clustering mode [GH-222](https://github.com/terraform-incus/terraform-provider-incus/pull/222)
- Added `target` to `incus_storage_pool` to help support clustering mode [GH-224](https://github.com/terraform-incus/terraform-provider-incus/pull/224)
- Added `target` to `incus_volume` to help support clustering mode [GH-225](https://github.com/terraform-incus/terraform-provider-incus/pull/225)
- Added support for importing `incus_network` resources [GH-226](https://github.com/terraform-incus/terraform-provider-incus/pull/226)

## 1.4.1 (December 12, 2020)

This version bump was primarily to do the initial publication to the Terraform Registry.

IMPROVEMENTS

- Added `target` argument to `incus_container` to help support clustering mode [GH-212](https://github.com/terraform-incus/terraform-provider-incus/pull/212)

## 1.4.0 (November 9, 2020)

IMPROVEMENTS

- Expose `incus_container` IPv6 address [GH-173](https://github.com/terraform-incus/terraform-provider-incus/pull/173)
- Added append to `incus_container_file` [GH-206](https://github.com/terraform-incus/terraform-provider-incus/pull/206)
- Allow updating `incus_network` in-place [GH-210](https://github.com/terraform-incus/terraform-provider-incus/pull/210)
- Added `incus_publish_image` [GH-209](https://github.com/terraform-incus/terraform-provider-incus/pull/209)
- Allow `linux.*` attributes in `incus_container` [GH-194](https://github.com/terraform-incus/terraform-provider-incus/pull/194)

## 1.3.0 (January 27, 2020)

BUG FIXES

- Fixed retrieved storage pool source in `incus_storage_pool` [GH-181](https://github.com/terraform-incus/terraform-provider-incus/pull/181)
- Fixed accidental deletion of `incus_container` device name [GH-184](https://github.com/terraform-incus/terraform-provider-incus/pull/184)

## 1.2.0 (June 20, 2019)

IMPROVEMENTS

- Added new valid device types [GH-150](https://github.com/terraform-incus/terraform-provider-incus/pull/150)
- Changed `incus_container.device` from a list to a set [GH-152](https://github.com/terraform-incus/terraform-provider-incus/pull/152)
- Deprecated `incus_snapshot.creation_date` in favour of `created_at` [GH-155](https://github.com/terraform-incus/terraform-provider-incus/pull/155)
