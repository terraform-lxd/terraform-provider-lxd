## 2.6.0 (October 14, 2025)

## What's Changed

### New Features 🎉
* Add instance device resource by @nmezhenskyi in https://github.com/terraform-lxd/terraform-provider-lxd/pull/584

### Maintenance and Chores 🛠
* .github/release: Improve generated release notes by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/577
* github: require all GH actions to be pinned to their SHA commit ID by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/581
* docs: update links to LXD docs by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/585
* build(deps): bump github.com/hashicorp/terraform-plugin-testing from 1.11.0 to 1.12.0 in the hashicorp group by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/578
* build(deps): bump golang.org/x/net from 0.37.0 to 0.38.0 by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/582
* build(deps): bump actions/setup-go from 5.4.0 to 5.5.0 by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/589
* build(deps): bump the hashicorp group with 5 updates by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/592
* build(deps): bump the hashicorp group with 2 updates by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/594
* build(deps): bump github.com/cloudflare/circl from 1.6.0 to 1.6.1 by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/597
* build(deps): bump github.com/hashicorp/terraform-plugin-testing from 1.13.1 to 1.13.2 in the hashicorp group by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/598
* Bump min go version and update deps by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/601
* build(deps): bump github.com/hashicorp/terraform-plugin-framework from 1.15.0 to 1.15.1 in the hashicorp group by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/605
* build(deps): bump actions/checkout from 4.2.2 to 4.3.0 by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/606
* build(deps): bump github.com/stretchr/testify from 1.10.0 to 1.11.0 by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/610
* build(deps): bump github.com/hashicorp/terraform-plugin-testing from 1.13.2 to 1.13.3 in the hashicorp group by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/609
* build(deps): bump goreleaser/goreleaser-action from 6.3.0 to 6.4.0 by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/611
* build(deps): bump github.com/stretchr/testify from 1.11.0 to 1.11.1 by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/614
* build(deps): bump actions/setup-go from 5.5.0 to 6.0.0 by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/615
* build(deps): bump the hashicorp group with 4 updates by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/617
* build(deps): bump github.com/hashicorp/terraform-plugin-sdk/v2 from 2.38.0 to 2.38.1 in the hashicorp group by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/618
* build(deps): bump github.com/hashicorp/terraform-plugin-framework from 1.16.0 to 1.16.1 in the hashicorp group by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/619
* build(deps): bump github.com/hashicorp/terraform-plugin-framework-validators from 0.18.0 to 0.19.0 in the hashicorp group by @dependabot[bot] in https://github.com/terraform-lxd/terraform-provider-lxd/pull/621

### Other Changes ❓
* Fix cluster tests and linter by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/579
* glangci-lint: Remove unnecessary exclusions by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/580
* Tweak GitHub workflows by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/593
* Fix unsupported content-type during file upload by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/602
* Rename `lxd_shared` to `lxdShared` by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/603
* tests/instance: Use different network address to prevent race condition by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/607
* test/storage: Remove resource state output by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/608
* Instance device resource: follow-up tests by @nmezhenskyi in https://github.com/terraform-lxd/terraform-provider-lxd/pull/604
* Add issue templates by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/612
* go: Update deps and bump Go version by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/613
* github: reclaim disk space in `actions/cluster` by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/622
* github: silence `lxc file push` of `minio` binaries to instances by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/623
* goreleaser: replace deprecated `archives.format` by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/625
* workflows/release: Add write content perm for token by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/624

## 2.5.0 (March 13, 2025)

## What's Changed

### New Features 🎉
* Allow cluster member group as target and handle in-cluster migration by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/559
* Profile, project, and image datasources (from Incus) by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/560
* Add network, storage pool and instance datasources by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/562
* feat: allow creation of empty VMs by @xvzf in https://github.com/terraform-lxd/terraform-provider-lxd/pull/563
* datasource: LXD info by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/573
* server/info: Report instance types by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/574

### Bug Fixes 🐝
* provider-config: Append port to hostname instead of the URL by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/545
* instance_file: Fix file removal and panic when file is not found by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/568
* Fix segfault when taking instance snapshot for an instance doesn't exist by @stew3254 in https://github.com/terraform-lxd/terraform-provider-lxd/pull/572
* Fix leftover resources on error by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/570

### Maintenance and Chores 🛠
* build(deps): bump github.com/hashicorp/terraform-plugin-framework-validators from 0.13.0 to 0.14.0 in the hashicorp group by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/537
* build(deps): bump the hashicorp group with 4 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/538
* build(deps): bump github.com/stretchr/testify from 1.9.0 to 1.10.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/541
* build(deps): bump github.com/hashicorp/terraform-plugin-testing from 1.10.0 to 1.11.0 in the hashicorp group by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/540
* docs: Include content_type in volume example by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/544
* Gomod updates by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/548
* Improve golangci-lint coverage by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/549
* build(deps): bump golang.org/x/crypto from 0.30.0 to 0.31.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/550
* build(deps): bump github.com/hashicorp/terraform-plugin-framework-validators from 0.15.0 to 0.16.0 in the hashicorp group by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/551
* gomod: Updates. by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/552
* github: Fix MinIO install step condition by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/553
* build(deps): bump github.com/hashicorp/terraform-plugin-framework-timeouts from 0.4.1 to 0.5.0 in the hashicorp group by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/555
* build(deps): bump github.com/hashicorp/terraform-plugin-go from 0.25.0 to 0.26.0 in the hashicorp group by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/556
* tests: Add cluster tests by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/554
* Misc cleanup by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/557
* Misc fix by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/558
* build(deps): bump the hashicorp group with 3 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/564
* build(deps): bump github.com/go-jose/go-jose/v4 from 4.0.4 to 4.0.5 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/567
* build(deps): bump golang.org/x/net from 0.34.0 to 0.36.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/575

## 2.4.0 (October 11, 2024)

## What's Changed
### New Features 🎉
* Add storage bucket resource by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/527
* Allow instance rename by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/526
* Combine the computed volume keys with the inherited storage pool keys (from Incus) by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/531
### Bug Fixes 🐝
* Do not blindly accept remote certificate when using trust token by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/529
* provider: Fix incorrect check for configured default remote  by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/534
### Maintenance and Chores 🛠
* docs: Move attach custom volume example to lxd_volume by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/528
* build(deps): bump the hashicorp group with 2 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/530

## 2.3.0 (September 3, 2024)

## What's Changed
### Breaking Changes ⚠️
* Refactor provider config and test trust token/password by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/518
### New Features 🎉
* Add network peer resource by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/504
* Allow managing default profiles in non-default projects by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/510
* Add trust token resource by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/512
### Bug Fixes 🐝
* Allow null description in LB backend and port by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/505
* Don't check fingerprint if 'Content' is unknown by @jsimpso in https://github.com/terraform-lxd/terraform-provider-lxd/pull/509
* Remove LXD version check during password auth by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/516
### Maintenance and Chores 🛠
* build(deps): bump the hashicorp group with 3 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/500
* build(deps): bump the hashicorp group with 2 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/513
### Other Changes ❓
* Fix server already trusted check by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/511
* Test truststore certificate using cert generated within the same TF config by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/519
* docs: Emphasize the trust password is no longer supported by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/521
* Ignore network peer status inconsistency in import test by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/522

## 2.2.0 (July 10, 2024)

## What's Changed
### New Features 🎉
* Add support for custom simplestreams remote by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/463
* Add support for client certificates by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/471
* Allow remote LXD authentication using token by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/469
* Support null values in config - from Incus by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/492
* Add network forward and ACL resources - from Incus by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/493
### Bug Fixes 🐝
* Do not error out if client certificate already exists by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/468
* Fix update of cached/published images by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/472
* Fix storage pool source inconsistencies by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/467
* Fix cached image being searched only in default project by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/490
* Fix cached image not found if instance remote is set by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/491
### Maintenance and Chores 🛠
* Update gomod by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/473
* docs: Refer to instance rather than container by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/476
* Use trust token instead of password for tests by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/477
* Update gomod by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/478
* build(deps): bump goreleaser/goreleaser-action from 5 to 6 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/479
* Use Alpine images for cached image test by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/475
* goreleaser: fix config for version 2 schema by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/480
* update go modules by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/482
* Pass trust token explicitly if server support it by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/484
* Let LXD decide the unix socket by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/485
* build(deps): bump golang.org/x/sys from 0.21.0 to 0.22.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/495
* build(deps): bump google.golang.org/grpc from 1.64.0 to 1.64.1 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/496
### Other Changes ❓
* Use math/rand/v2 for string generation by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/470
* goreleaser: re-enable arm64 builds for Windows by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/481
* docs/resources/instance: add multiple ordered `execs` by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/488
* Fix network zone minor issue - from Incus by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/494
* Remove project attribute on provider by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/497
* github: set goreleaser version to v2 by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/498

## 2.1.0 (May 27, 2024)

### New Features 🎉
* Export instance's network interfaces by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/460
### Bug Fixes 🐝
* Replace image resource_id by reading fingerprint from state by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/421
* Add instance timeouts by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/423
* Upload files after VM is started by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/436
* Expand ConfigDir env vars when config.yml is missing by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/446
* Read access_interface from expanded config by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/449
### Maintenance and Chores 🛠
* Fix minimum LXD version in docs by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/419
* build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/425
* Generate unique names for test resources by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/424
* build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/429
* build(deps): bump golang.org/x/sys from 0.16.0 to 0.17.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/431
* Bump Go to 1.21 by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/434
* Update go deps by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/435
* Update gomod by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/437
* build(deps): bump github.com/stretchr/testify from 1.8.4 to 1.9.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/438
* build(deps): bump golang.org/x/sys from 0.17.0 to 0.18.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/440
* build(deps): bump the hashicorp group with 2 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/439
* build(deps): bump google.golang.org/protobuf from 1.32.0 to 1.33.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/441
* github: run tests against 5.21/stable and 5.21/edge by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/442
* build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/443
* internal/common/lxd_file: use ParseInt() for LXD file mode by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/444
* build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/447
* Go 1.22 by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/450
* build(deps): bump golang.org/x/sys from 0.18.0 to 0.19.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/451
* github: stop testing with Go 1.21 by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/455
* build(deps): bump golang.org/x/net from 0.21.0 to 0.23.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/454
* build(deps): bump the hashicorp group with 2 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/456
* build(deps): bump golang.org/x/sys from 0.19.0 to 0.20.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/458
* build(deps): bump github.com/hashicorp/terraform-plugin-go from 0.22.2 to 0.23.0 in the hashicorp group by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/459
* build(deps): bump the hashicorp group with 2 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/462
* build(deps): bump github.com/hashicorp/go-version from 1.6.0 to 1.7.0 in the hashicorp group by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/465
* Bump lxd client and use Alpine image for tests by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/461
### Other Changes ❓
* github: remove DCO check replaced by DCO app by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/426
* github: resume testing on 5.0/stable now that 5.0.3 is released by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/430

## 2.0.0 (January 23, 2024)

### Breaking Changes ⚠️
* Drop deprecated lxd_volume_container_attach resource by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/348
* Make LXD 4.0 the minimum officially supported version by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/379
* Drop deprecated provider parameters by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/378
* Drop deprecated lxd_container and lxd_container_file resources by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/377
* Replace start_on_create with running attribute by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/387
### New Features 🎉
* Add missing instance device types by @kapows in https://github.com/terraform-lxd/terraform-provider-lxd/pull/366
* Support `config.cloud-init.*` keys by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/383
* Add exec block to the instance resource by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/386
### Maintenance and Chores 🛠
* fix tests using alpine images by @adamcstephens in https://github.com/terraform-lxd/terraform-provider-lxd/pull/370
* build(deps): bump hashicorp/setup-terraform from 2 to 3 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/368
* test: Fix instance reference within test by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/372
* test: Drop VM test for old lxd_container resource by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/373
* build(deps): bump goreleaser/goreleaser-action from 4 to 5 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/367
* build(deps): bump golang.org/x/sys from 0.13.0 to 0.14.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/369
* build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/371
* Test for required API extensions in network LB and zones tests by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/375
* tests: don't run pr when converting from draft by @adamcstephens in https://github.com/terraform-lxd/terraform-provider-lxd/pull/384
* Update from SDKv2 to terraform-framework by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/385
* build(deps): bump golang.org/x/sys from 0.14.0 to 0.15.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/388
* Test against `5.0/stable` and `5.0/candidate` snap by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/390
* build(deps): bump the hashicorp group with 2 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/393
* build(deps): bump actions/setup-go from 4 to 5 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/392
* Cleanup SDKv2 resources by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/389
* Update go dependencies by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/397
* build(deps): bump the hashicorp group with 2 updates by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/400
* build(deps): bump golang.org/x/crypto from 0.16.0 to 0.17.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/401
* build(deps): bump golang.org/x/sys from 0.15.0 to 0.16.0 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/411
* build(deps): bump github.com/cloudflare/circl from 1.3.6 to 1.3.7 by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/412
* build(deps): bump the hashicorp group with 1 update by @dependabot in https://github.com/terraform-lxd/terraform-provider-lxd/pull/414
* Add migration guide for 2.0 by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/415
### Other Changes ❓
* Better workaround for LVM bug by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/391
* Use version variable which is overwritten by goreleaser by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/395
* Remove refresh_interval attribute from provider by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/394
* README: improve `~/.terraformrc` to allow using other providers easily by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/402
* Sync from provider incus by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/404
* Wait exec output to be flushed and use -1 as default exec exit code by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/407
* github: add DCO check by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/403
* Switch away from `images:` and use `ubuntu-daily:` remote instead by @simondeziel in https://github.com/terraform-lxd/terraform-provider-lxd/pull/406
* Add project import by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/410
* Remove unused test-infra by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/416
* Exec trigger and enabled attributes by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/413
* Fix typo in migration guide by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/417

## 1.10.4 (November 3, 2023)

### Bug Fixes 🐝
* Fix unix socket check on Windows by @MusicDin in https://github.com/terraform-lxd/terraform-provider-lxd/pull/365
### Maintenance and Chores 🛠
* testing: cleanup and catch more by @adamcstephens in https://github.com/terraform-lxd/terraform-provider-lxd/pull/364

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
