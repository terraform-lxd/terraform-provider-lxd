package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccVolumeContainerAttach_basic(t *testing.T) {
	t.Skip("lxd_volume_container_attach is deprecated and will be removed in the future")

	poolName := strings.ToLower(petname.Generate(2, "-"))
	volumeName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVolumeContainerAttachBasicConfig(poolName, volumeName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccVolumeContainerAttachExists(t, "lxd_volume_container_attach.attach1"),
					resource.TestCheckResourceAttr("lxd_volume_container_attach.attach1", "volume_name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume_container_attach.attach1", "container_name", containerName),
					resource.TestCheckResourceAttr("lxd_volume_container_attach.attach1", "device_name", volumeName),
				),
			},
		},
	})
}

func TestAccVolumeContainerAttach_deviceName(t *testing.T) {
	t.Skip("lxd_volume_container_attach is deprecated and will be removed in the future")

	poolName := strings.ToLower(petname.Generate(2, "-"))
	volumeName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVolumeContainerAttachDeviceNameConfig(poolName, volumeName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccVolumeContainerAttachExists(t, "lxd_volume_container_attach.attach1"),
					resource.TestCheckResourceAttr("lxd_volume_container_attach.attach1", "volume_name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume_container_attach.attach1", "container_name", containerName),
					resource.TestCheckResourceAttr("lxd_volume_container_attach.attach1", "device_name", "foo"),
				),
			},
		},
	})
}

func testAccVolumeContainerAttachExists(t *testing.T, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		v := newVolumeAttachmentIDFromResourceID(rs.Primary.ID)
		client, err := testAccProvider.Meta().(*lxdProvider).GetContainerServer("")
		if err != nil {
			return err
		}
		_, _, err = resourceLxdVolumeContainerAttachedVolume(client, v)
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccVolumeContainerAttachBasicConfig(poolName, volumeName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name = "%s"
  driver = "dir"
  config {
    source = "/tmp/foo"
  }
}

resource "lxd_volume" "volume1" {
  name = "%s"
  pool = "${lxd_storage_pool.pool1.name}"
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9"
  profiles = ["default"]
}

resource "lxd_volume_container_attach" "attach1" {
  pool = "${lxd_storage_pool.pool1.name}"
  volume_name = "${lxd_volume.volume1.name}"
  container_name = "${lxd_container.container1.name}"
  path = "/tmp"
}
	`, poolName, volumeName, containerName)
}

func testAccVolumeContainerAttachDeviceNameConfig(poolName, volumeName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name = "%s"
  driver = "dir"
  config {
    source = "/tmp/foo"
  }
}

resource "lxd_volume" "volume1" {
  name = "%s"
  pool = "${lxd_storage_pool.pool1.name}"
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9"
  profiles = ["default"]
}

resource "lxd_volume_container_attach" "attach1" {
  pool = "${lxd_storage_pool.pool1.name}"
  volume_name = "${lxd_volume.volume1.name}"
  container_name = "${lxd_container.container1.name}"
  path = "/tmp"
  device_name = "foo"
}
	`, poolName, volumeName, containerName)
}
