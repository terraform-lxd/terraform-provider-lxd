package instance_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccInstanceDevice_basic(t *testing.T) {
	instanceName := acctest.GenerateName(2, "-")
	deviceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDevice_basic(instanceName, deviceName),
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

func TestAccInstanceDevice_volumeAttach(t *testing.T) {
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
				Config: testAccInstanceDevice_volumeAttach(poolName, volumeName, instanceName, "/mnt"),
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
				Config: testAccInstanceDevice_volumeAttach(poolName, volumeName, instanceName, "/data"),
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
				Config: testAccInstanceDevice_volumeDetach(poolName, volumeName, instanceName),
			},
			{
				// Validate detaching here. Otherwise, the datasource for lxd instance will
				// read the state before device resource gets destroyed.
				// By validating in a separate step with the same config, the final
				// state should remain the same, but the datasource for lxd instance will
				// see the changes made in the previous "terraform apply".
				Config: testAccInstanceDevice_volumeDetach(poolName, volumeName, instanceName),
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

func TestAccInstanceDevice_coexistingDevices(t *testing.T) {
	instanceName := acctest.GenerateName(2, "-")
	poolName := acctest.GenerateName(2, "-")
	volumeName := acctest.GenerateName(2, "-")
	sharedDiskName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDevice_coexistingDevices(poolName, volumeName, sharedDiskName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),

					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "pool", poolName),

					// "lxd_instance" resource state should contain only its own device.
					resource.TestCheckResourceAttr("lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.inst", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_instance.inst", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.inst", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_instance.inst", "device.0.properties.path", "/tmp/shared"),
					resource.TestCheckResourceAttr("lxd_instance.inst", "device.0.properties.source", "/tmp"),

					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "instance_name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "type", "disk"),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "properties.path", "/data"),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "properties.source", volumeName),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "properties.pool", poolName),

					// Check with datasource to make sure that the instance actually contains both devices.
					resource.TestCheckResourceAttr("data.lxd_instance.inst", "devices.%", "2"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.type", volumeName), "disk"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.path", volumeName), "/data"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.pool", volumeName), poolName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.source", volumeName), volumeName),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.type", sharedDiskName), "disk"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.path", sharedDiskName), "/tmp/shared"),
					resource.TestCheckResourceAttr("data.lxd_instance.inst", fmt.Sprintf("devices.%s.properties.source", sharedDiskName), "/tmp"),
				),
			},
		},
	})
}

func TestAccInstanceDevice_volumeAttachCluster(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)
	instanceName := acctest.GenerateName(2, "-")
	poolName := acctest.GenerateName(2, "-")
	driverName := "dir"
	volumeName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceDevice_volumeAttachCluster(instanceName, poolName, driverName, volumeName, targets),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool_node1", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool_node1", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool_node1", "target", targets[0]),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool_node2", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool_node2", "driver", driverName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool_node2", "target", targets[1]),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool", "name", poolName),
					resource.TestCheckResourceAttr("lxd_storage_pool.storage_pool", "driver", driverName),

					resource.TestCheckResourceAttr("lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.inst", "status", "Stopped"),

					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "name", volumeName),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "instance_name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "type", "disk"),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "properties.path", "/data"),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "properties.source", volumeName),
					resource.TestCheckResourceAttr("lxd_instance_device.vol_attach", "properties.pool", poolName),
				),
			},
		},
	})
}

func testAccInstanceDevice_basic(instanceName string, deviceName string) string {
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

func testAccInstanceDevice_volumeAttach(poolName string, volumeName string, instanceName string, mountPath string) string {
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

func testAccInstanceDevice_volumeDetach(poolName string, volumeName string, instanceName string) string {
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

func testAccInstanceDevice_coexistingDevices(poolName string, volumeName string, sharedDiskName string, instanceName string) string {
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

  device {
     name = %q
     type = "disk"
     properties = {
        source = "/tmp"
        path = "/tmp/shared"
     }
  }
}

resource "lxd_instance_device" "vol_attach" {
   name          = lxd_volume.volume1.name
   instance_name = lxd_instance.inst.name
   type          = "disk"

   properties = {
      path   = "/data"
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
   `, poolName, volumeName, instanceName, acctest.TestImage, sharedDiskName)
}

func testAccInstanceDevice_volumeAttachCluster(instanceName string, poolName string, driver string, volumeName string, targets []string) string {
	var config string
	var deps []string

	for i, target := range targets {
		deps = append(deps, "lxd_storage_pool.storage_pool_node"+strconv.Itoa(i+1))
		config += fmt.Sprintf(`
resource "lxd_storage_pool" "storage_pool_node%d" {
  name   = "%s"
  driver = "%s"
  target = "%s"
}`, i+1, poolName, driver, target)
	}

	config += fmt.Sprintf(`
resource "lxd_storage_pool" "storage_pool" {
  depends_on = [ %[3]s ]
  name       = "%[1]s"
  driver     = "%[2]s"
}`, poolName, driver, strings.Join(deps, ", "))

	config += fmt.Sprintf(`
resource "lxd_instance" "inst" {
   name    = %q
   image   = %q
   running = false
}

resource "lxd_instance_device" "vol_attach" {
   name          = lxd_volume.volume1.name
   instance_name = lxd_instance.inst.name
   target        = lxd_instance.inst.location
   type          = "disk"

   properties = {
      path   = "/data"
      source = lxd_volume.volume1.name
      pool   = lxd_storage_pool.storage_pool.name
   }
}

resource "lxd_volume" "volume1" {
  name   = %q
  pool   = lxd_storage_pool.storage_pool.name
  target = lxd_instance.inst.location
}
   `, instanceName, acctest.TestImage, volumeName)

	return config
}
