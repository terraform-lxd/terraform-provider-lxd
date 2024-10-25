package storage_test

import (
	"fmt"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccStorageVolume_basic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_basic(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "dir"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "type", "custom"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "filesystem"),
				),
			},
		},
	})
}

func TestAccStorageVolume_instanceAttach(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_instanceAttach(poolName, volumeName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.name", "volume1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.path", "/mnt"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.pool", poolName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.source", volumeName),
				),
			},
		},
	})
}

func TestAccStorageVolume_target(t *testing.T) {
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_target(volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", "default"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "target", "node-2"),
				),
			},
		},
	})
}

func TestAccStorageVolume_project(t *testing.T) {
	volumeName := petname.Generate(2, "-")
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_project(projectName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", "default"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "project", projectName),
				),
			},
		},
	})
}

func TestAccStorageVolume_contentType(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_contentType(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "block"),
				),
			},
		},
	})
}

func TestAccStorageVolume_importBasic(t *testing.T) {
	volName := petname.Generate(2, "-")
	poolName := petname.Generate(2, "-")
	resourceName := "incus_storage_volume.volume1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_basic(poolName, volName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("/%s/%s", poolName, volName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func TestAccStorageVolume_importProject(t *testing.T) {
	volName := petname.Generate(2, "-")
	projectName := petname.Generate(2, "-")
	resourceName := "incus_storage_volume.volume1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_project(projectName, volName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/default/%s", projectName, volName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func TestAccStorageVolume_inheritedStoragePoolKeys(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_inheritedStoragePoolVolumeKeys(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "config.volume.zfs.remove_snapshots", "true"),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "config.volume.zfs.use_refquota", "true"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "type", "custom"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "block"),

					// Ensure computed keys are not tracked.
					resource.TestCheckNoResourceAttr("incus_storage_volume.volume1", "config.zfs.remove_snapshots"),
					resource.TestCheckNoResourceAttr("incus_storage_volume.volume1", "config.zfs.use_refquota"),
				),
			},
		},
	})
}

func TestAccStorageVolume_sourceVolume(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_sourceVolume(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "lvm"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1_copy", "name", fmt.Sprintf("%s-copy", volumeName)),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1_copy", "pool", "default"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1_copy", "source_volume.name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1_copy", "source_volume.pool", poolName),
				),
			},
			{
				Config: testAccStorageVolume_sourceVolume(poolName, volumeName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccStorageVolume_basic(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver = "dir"
}

resource "incus_storage_volume" "volume1" {
  name = "%s"
  pool = incus_storage_pool.pool1.name
}
	`, poolName, volumeName)
}

func testAccStorageVolume_instanceAttach(poolName, volumeName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver = "zfs"
}

resource "incus_storage_volume" "volume1" {
  name = "%s"
  pool = incus_storage_pool.pool1.name
}

resource "incus_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  running = false

  device {
    name = "volume1"
    type = "disk"
    properties = {
      path   = "/mnt"
      source = incus_storage_volume.volume1.name
      pool   = incus_storage_pool.pool1.name
    }
  }
}
	`, poolName, volumeName, instanceName, acctest.TestImage)
}

func testAccStorageVolume_target(volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_volume" "volume1" {
  name   = "%s"
  pool   = "default"
  target = "node-2"
}
	`, volumeName)
}

func testAccStorageVolume_project(projectName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
  config = {
    "features.storage.volumes" = false
  }
}

resource "incus_storage_volume" "volume1" {
  name    = "%s"
  pool    = "default"
  project = incus_project.project1.name
}
	`, projectName, volumeName)
}

func testAccStorageVolume_contentType(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver = "zfs"
}

resource "incus_storage_volume" "volume1" {
  name         = "%s"
  pool         = incus_storage_pool.pool1.name
  content_type = "block"
}
	`, poolName, volumeName)
}

func testAccStorageVolume_inheritedStoragePoolVolumeKeys(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver  = "zfs"
  config = {
    "volume.zfs.remove_snapshots" = "true",
	"volume.zfs.use_refquota" = "true"
  }
}

resource "incus_storage_volume" "volume1" {
  name = "%s"
  pool = incus_storage_pool.pool1.name
  content_type = "block"
  config = {
    "size" = "1GiB"
  }
}
	`, poolName, volumeName)
}

func testAccStorageVolume_sourceVolume(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "lvm"
}

resource "incus_storage_volume" "volume1" {
  name = "%[2]s"
  pool = incus_storage_pool.pool1.name
}

resource "incus_storage_volume" "volume1_copy" {
  name = "%[2]s-copy"
  pool = "default"

  source_volume = {
    pool = incus_storage_pool.pool1.name
    name = incus_storage_volume.volume1.name
  }
}
`,
		poolName, volumeName)
}
