package instance_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccDevice_basic(t *testing.T) {
	instanceName := acctest.GenerateName(2, "-")
	deviceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDevice_basic(instanceName, deviceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.%", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.type", deviceName), "disk"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.path", deviceName), fmt.Sprintf("/tmp/%s", deviceName)),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.source", deviceName), "/tmp"),
				),
			},
		},
	})
}

func TestAccDevice_volumeAttach(t *testing.T) {
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
				// Attach a volume to an instance.
				Config: testAccDevice_volumeAttach(poolName, volumeName, instanceName, "/mnt"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.%", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.type", volumeName), "disk"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.path", volumeName), "/mnt"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.pool", volumeName), poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.source", volumeName), volumeName),
				),
			},
			{
				// Try reattaching the volume.
				Config: testAccDevice_volumeAttach(poolName, volumeName, instanceName, "/data"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.%", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.type", volumeName), "disk"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.path", volumeName), "/data"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.pool", volumeName), poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.source", volumeName), volumeName),
				),
			},
			{
				// Try detaching the volume.
				Config: testAccDevice_volumeDetach(poolName, volumeName, instanceName),
			},
			{
				// Validate detaching here. Otherwise, the datasource for lxd instance will
				// read the state before device resource gets destroyed.
				// By validating in a separate step with the same config, the final
				// state should remain the same, but the datasource for lxd instance will
				// see the changes made in the previous "terraform apply".
				Config: testAccDevice_volumeDetach(poolName, volumeName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.%", "0"),
				),
			},
		},
	})
}

func testAccDevice_basic(instanceName string, deviceName string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "inst" {
   name    = %q
   image   = %q
   running = false
}

resource "lxd_instance_device" "disk_attach" {
   name = %q
   instance_name = lxd_instance.inst.name
   type = "disk"
   properties = {
      source = "/tmp"
      path   = "/tmp/%s"
   }
}

data "lxd_instance" "inst" {
   name = lxd_instance.inst.name

   depends_on = [
      lxd_instance_device.disk_attach
   ]
}
   `, instanceName, acctest.TestImage, deviceName, deviceName)
}

func testAccDevice_volumeAttach(poolName string, volumeName string, instanceName string, mountPath string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = %q
  driver = "zfs"
}

resource "lxd_volume" "volume1" {
  name = %q
  pool = lxd_storage_pool.pool1.name
}

resource "lxd_instance" "inst" {
  name    = %q
  image   = %q
  running = false
}

resource "lxd_instance_device" "vol_attach" {
   name          = lxd_volume.volume1.name
   instance_name = lxd_instance.inst.name
   type          = "disk"

   properties = {
      path   = %q
      source = lxd_volume.volume1.name
      pool   = lxd_storage_pool.pool1.name
   }
}

data "lxd_instance" "inst" {
   name = lxd_instance.inst.name

   depends_on = [
      lxd_instance_device.vol_attach
   ]
}
	`, poolName, volumeName, instanceName, acctest.TestImage, mountPath)
}

func testAccDevice_volumeDetach(poolName string, volumeName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = %q
  driver = "zfs"
}

resource "lxd_volume" "volume1" {
  name = %q
  pool = lxd_storage_pool.pool1.name
}

resource "lxd_instance" "inst" {
  name    = %q
  image   = %q
  running = false
}

data "lxd_instance" "inst" {
   name = lxd_instance.inst.name
}
	`, poolName, volumeName, instanceName, acctest.TestImage)
}
