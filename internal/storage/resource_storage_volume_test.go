package storage_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccStorageVolume_basic(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	volumeName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_basic(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "dir"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "type", "custom"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "content_type", "filesystem"),
				),
			},
		},
	})
}

func TestAccStorageVolume_instanceAttach(t *testing.T) {
	instanceName := acctest.GenerateName(2, "-")
	poolName := acctest.GenerateName(2, "-")
	volumeName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_instanceAttach(poolName, volumeName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.name", "volume1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.path", "/mnt"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.pool", poolName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.source", volumeName),
				),
			},
		},
	})
}

func TestAccStorageVolume_target(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 1)
	volumeName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_target(volumeName, targets[0]),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", "default"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "target", targets[0]),
				),
			},
		},
	})
}

func TestAccStorageVolume_project(t *testing.T) {
	volumeName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_project(projectName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", "default"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "project", projectName),
				),
			},
		},
	})
}

func TestAccStorageVolume_contentType(t *testing.T) {
	poolName := acctest.GenerateName(2, "-")
	volumeName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_contentType(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "content_type", "block"),
				),
			},
		},
	})
}

func TestAccStorageVolume_importBasic(t *testing.T) {
	volName := acctest.GenerateName(2, "-")
	poolName := acctest.GenerateName(2, "-")
	resourceName := "lxd_volume.volume1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
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
	volName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")
	resourceName := "lxd_volume.volume1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
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
	poolName := acctest.GenerateName(2, "-")
	volumeName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_inheritedStoragePoolVolumeKeys(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "config.volume.zfs.remove_snapshots", "true"),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "config.volume.zfs.use_refquota", "true"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "type", "custom"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "content_type", "block"),

					// Ensure computed keys are not tracked.
					resource.TestCheckNoResourceAttr("lxd_volume.volume1", "config.zfs.remove_snapshots"),
					resource.TestCheckNoResourceAttr("lxd_volume.volume1", "config.zfs.use_refquota"),
				),
			},
		},
	})
}

func testAccStorageVolume_basic(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = "%s"
  driver = "dir"
}

resource "lxd_volume" "volume1" {
  name = "%s"
  pool = lxd_storage_pool.pool1.name
}
	`, poolName, volumeName)
}

func testAccStorageVolume_instanceAttach(poolName, volumeName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = "%s"
  driver = "zfs"
}

resource "lxd_volume" "volume1" {
  name = "%s"
  pool = lxd_storage_pool.pool1.name
}

resource "lxd_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  running = false

  device {
    name = "volume1"
    type = "disk"
    properties = {
      path   = "/mnt"
      source = lxd_volume.volume1.name
      pool   = lxd_storage_pool.pool1.name
    }
  }
}
	`, poolName, volumeName, instanceName, acctest.TestImage)
}

func testAccStorageVolume_target(volumeName string, target string) string {
	return fmt.Sprintf(`
resource "lxd_volume" "volume1" {
  name   = "%s"
  pool   = "default"
  target = "%s"
}
	`, volumeName, target)
}

func testAccStorageVolume_project(projectName, volumeName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
  config = {
    "features.storage.volumes" = false
  }
}

resource "lxd_volume" "volume1" {
  name    = "%s"
  pool    = "default"
  project = lxd_project.project1.name
}
	`, projectName, volumeName)
}

func testAccStorageVolume_contentType(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = "%s"
  driver = "zfs"
}

resource "lxd_volume" "volume1" {
  name         = "%s"
  pool         = lxd_storage_pool.pool1.name
  content_type = "block"
}
	`, poolName, volumeName)
}

func testAccStorageVolume_inheritedStoragePoolVolumeKeys(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = "%s"
  driver = "zfs"
  config = {
    "volume.zfs.remove_snapshots" = true
    "volume.zfs.use_refquota"     = true
  }
}

resource "lxd_volume" "volume1" {
  name         = "%s"
  pool         = lxd_storage_pool.pool1.name
  content_type = "block"
  config = {
    "size" = "1GiB"
  }
}
	`, poolName, volumeName)
}
