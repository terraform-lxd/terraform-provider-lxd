package storage_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccVolume_basic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVolume_basic(poolName, volumeName),
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

func TestAccVolume_instanceAttach(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVolume_instanceAttach(poolName, volumeName, instanceName),
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

// TODO:
// - clustering precheck
// func TestAccVolume_target(t *testing.T) {
// 	volumeName := petname.Generate(2, "-")

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:                 func() { acctest.PreCheck(t) },
// 		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccVolume_target(volumeName),
// 				Check: resource.ComposeTestCheckFunc(
// 					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
// 					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", "default"),
// 					resource.TestCheckResourceAttr("lxd_volume.volume1", "target", "node2"),
// 				),
// 			},
// 		},
// 	})
// }

func TestAccVolume_project(t *testing.T) {
	volumeName := petname.Generate(2, "-")
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVolume_project(projectName, volumeName),
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

func TestAccVolume_contentType(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVolume_contentType(poolName, volumeName),
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

func testAccVolume_basic(poolName, volumeName string) string {
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

func testAccVolume_instanceAttach(poolName, volumeName, instanceName string) string {
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
  name            = "%s"
  image           = "images:alpine/3.18/amd64"
  start_on_create = false

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
	`, poolName, volumeName, instanceName)
}

func testAccVolume_target(volumeName string) string {
	return fmt.Sprintf(`
resource "lxd_volume" "volume1" {
  name   = "%s"
  target = "node2"
  pool   = "default"
}
	`, volumeName)
}

func testAccVolume_project(project, volumeName string) string {
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
	`, project, volumeName)
}

func testAccVolume_contentType(poolName, volumeName string) string {
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
