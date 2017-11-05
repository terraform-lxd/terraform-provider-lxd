package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/lxc/lxd/shared/api"
)

func TestAccVolume_basic(t *testing.T) {
	var volume api.StorageVolume
	poolName := strings.ToLower(petname.Generate(2, "-"))
	volumeName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVolume_basic(poolName, volumeName),
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

		v := NewVolumeIDFromResourceID(rs.Primary.ID)
		client, err := testAccProvider.Meta().(*LxdProvider).GetContainerServer("")
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

func testAccVolume_basic(poolName, volumeName string) string {
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
	`, poolName, volumeName)
}
