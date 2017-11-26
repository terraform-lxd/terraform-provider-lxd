package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccVolumeProfileAttach_basic(t *testing.T) {
	poolName := strings.ToLower(petname.Generate(2, "-"))
	volumeName := strings.ToLower(petname.Generate(2, "-"))
	profileName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVolumeProfileAttach_basic(poolName, volumeName, profileName),
				Check: resource.ComposeTestCheckFunc(
					testAccVolumeProfileAttachExists(t, "lxd_volume_profile_attach.attach1"),
					resource.TestCheckResourceAttr("lxd_volume_profile_attach.attach1", "volume_name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume_profile_attach.attach1", "profile_name", profileName),
					resource.TestCheckResourceAttr("lxd_volume_profile_attach.attach1", "device_name", volumeName),
				),
			},
		},
	})
}

func TestAccVolumeProfileAttach_deviceName(t *testing.T) {
	poolName := strings.ToLower(petname.Generate(2, "-"))
	volumeName := strings.ToLower(petname.Generate(2, "-"))
	profileName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVolumeProfileAttach_deviceName(poolName, volumeName, profileName),
				Check: resource.ComposeTestCheckFunc(
					testAccVolumeProfileAttachExists(t, "lxd_volume_profile_attach.attach1"),
					resource.TestCheckResourceAttr("lxd_volume_profile_attach.attach1", "volume_name", volumeName),
					resource.TestCheckResourceAttr("lxd_volume_profile_attach.attach1", "profile_name", profileName),
					resource.TestCheckResourceAttr("lxd_volume_profile_attach.attach1", "device_name", "foo"),
				),
			},
		},
	})
}

func testAccVolumeProfileAttachExists(t *testing.T, n string) resource.TestCheckFunc {
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
		_, _, err = resourceLxdVolumeProfileAttachedVolume(client, v)
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccVolumeProfileAttach_basic(poolName, volumeName, profileName string) string {
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

resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_volume_profile_attach" "attach1" {
  pool = "${lxd_storage_pool.pool1.name}"
  volume_name = "${lxd_volume.volume1.name}"
  profile_name = "${lxd_profile.profile1.name}"
  path = "/tmp"
}
	`, poolName, volumeName, profileName)
}

func testAccVolumeProfileAttach_deviceName(poolName, volumeName, profileName string) string {
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

resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_volume_profile_attach" "attach1" {
  pool = "${lxd_storage_pool.pool1.name}"
  volume_name = "${lxd_volume.volume1.name}"
  profile_name = "${lxd_profile.profile1.name}"
  path = "/tmp"
  device_name = "foo"
}
	`, poolName, volumeName, profileName)
}
