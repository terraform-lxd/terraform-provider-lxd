package lxd

import (
	"fmt"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/lxc/lxd/shared/api"
)

func TestAccVolume_basic(t *testing.T) {
	var volume api.StorageVolume
	poolName := strings.ToLower(petname.Generate(2, "-"))
	volumeName := strings.ToLower(petname.Generate(2, "-"))
	source := t.TempDir()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccVolume_basic(poolName, source, volumeName),
				Check: resource.ComposeTestCheckFunc(
					testAccVolumeExists(t, "lxd_volume.volume1", &volume),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
				),
			},
		},
	})
}

func TestAccVolume_containerAttach(t *testing.T) {
	var volume api.StorageVolume
	containerName := strings.ToLower(petname.Generate(2, "-"))
	poolName := strings.ToLower(petname.Generate(2, "-"))
	volumeName := strings.ToLower(petname.Generate(2, "-"))
	source := t.TempDir()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccVolume_containerAttach(poolName, source, volumeName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccVolumeExists(t, "lxd_volume.volume1", &volume),
					resource.TestCheckResourceAttr("lxd_volume.volume1", "name", volumeName),
				),
			},
		},
	})
}

func testAccVolumeExists(t *testing.T, n string, volume *api.StorageVolume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		v := newVolumeIDFromResourceID(rs.Primary.ID)
		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		vol, _, err := client.GetStoragePoolVolume(v.pool, v.volType, v.name)
		if err != nil {
			return err
		}

		*volume = *vol

		return nil
	}
}

func testAccVolumeConfig(volume *api.StorageVolume, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if volume.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range volume.Config {
			if k != key {
				continue
			}

			if v == value {
				return nil
			}

			return fmt.Errorf("Bad value for %s: %s", k, value)
		}

		return fmt.Errorf("Config not found: %s", k)
	}
}

func testAccVolume_basic(poolName, source, volumeName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name = "%s"
  driver = "dir"
  config = {
    source = "%s"
  }
}

resource "lxd_volume" "volume1" {
  name = "%s"
  pool = "${lxd_storage_pool.pool1.name}"
}
	`, poolName, source, volumeName)
}

func testAccVolume_containerAttach(poolName, source, volumeName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name = "%s"
  driver = "dir"
  config = {
    source = "%s"
  }
}

resource "lxd_volume" "volume1" {
  name = "%s"
  pool = "${lxd_storage_pool.pool1.name}"
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.12/amd64"
  profiles = ["default"]

  device {
    name = "volume1"
    type = "disk"
    properties = {
      path = "/mnt"
      source = "${lxd_volume.volume1.name}"
      pool = "${lxd_storage_pool.pool1.name}"
    }
  }
}
	`, poolName, source, volumeName, containerName)
}
