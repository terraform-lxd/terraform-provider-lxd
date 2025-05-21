package device_test

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
				Config: testAccDevice_volumeAttach(poolName, volumeName, instanceName, "/mnt"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.%", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.volume1.type", "disk"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.volume1.properties.path", "/mnt"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.volume1.properties.pool", poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.volume1.properties.source", volumeName),
				),
			},
			{
				// Try reattaching.
				Config: testAccDevice_volumeAttach(poolName, volumeName, instanceName, "/data"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.%", "1"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.volume1.type", "disk"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.volume1.properties.path", "/data"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.volume1.properties.pool", poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.volume1.properties.source", volumeName),
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

resource "lxd_device" "disk_attach" {
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
      lxd_device.disk_attach
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

resource "lxd_device" "vol_attach" {
   name          = "volume1"
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
      lxd_device.vol_attach
   ]
}
	`, poolName, volumeName, instanceName, acctest.TestImage, mountPath)
}
