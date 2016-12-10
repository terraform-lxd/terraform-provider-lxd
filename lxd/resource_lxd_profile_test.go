package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/lxc/lxd/shared"
)

func TestAccProfile_basic(t *testing.T) {
	var profile shared.ProfileConfig
	profileName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfile_basic(profileName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
				),
			},
		},
	})
}

func TestAccProfile_config(t *testing.T) {
	var profile shared.ProfileConfig
	profileName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfile_config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "config.limits.cpu", "2"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileConfig(&profile, "limits.cpu", "2"),
				),
			},
		},
	})
}

func TestAccProfile_device(t *testing.T) {
	var profile shared.ProfileConfig
	profileName := strings.ToLower(petname.Generate(2, "-"))

	device1 := shared.Device{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared",
	}

	device2 := shared.Device{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared2",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfile_device_1(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileDevice(&profile, "shared", device1),
				),
			},
			resource.TestStep{
				Config: testAccProfile_device_2(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared2"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileDevice(&profile, "shared", device2),
				),
			},
		},
	})
}

func TestAccProfile_addDevice(t *testing.T) {
	var profile shared.ProfileConfig
	profileName := strings.ToLower(petname.Generate(2, "-"))

	device := shared.Device{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfile_addDevice_1(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
				),
			},
			resource.TestStep{
				Config: testAccProfile_addDevice_2(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileDevice(&profile, "shared", device),
				),
			},
		},
	})
}

func TestAccProfile_removeDevice(t *testing.T) {
	var profile shared.ProfileConfig
	profileName := strings.ToLower(petname.Generate(2, "-"))

	device := shared.Device{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfile_removeDevice_1(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileDevice(&profile, "shared", device),
				),
			},
			resource.TestStep{
				Config: testAccProfile_removeDevice_2(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileNoDevice(&profile, "shared", device),
				),
			},
		},
	})
}

func TestAccProfile_containerBasic(t *testing.T) {
	var profile shared.ProfileConfig
	var container shared.ContainerInfo
	profileName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfile_containerBasic(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerProfile(&container, profileName),
				),
			},
		},
	})
}

func TestAccProfile_containerConfig(t *testing.T) {
	var profile shared.ProfileConfig
	var container shared.ContainerInfo
	profileName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfile_containerConfig(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "config.limits.cpu", "2"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccProfileConfig(&profile, "limits.cpu", "2"),
					testAccContainerExpandedConfig(&container, "limits.cpu", "2"),
				),
			},
		},
	})
}

func TestAccProfile_containerDevice(t *testing.T) {
	var profile shared.ProfileConfig
	var container shared.ContainerInfo
	profileName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	device := shared.Device{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfile_containerDevice(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccProfileDevice(&profile, "shared", device),
					testAccContainerExpandedDevice(&container, "shared", device),
				),
			},
		},
	})
}

func testAccProfileRunning(t *testing.T, n string, profile *shared.ProfileConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*LxdProvider).Client
		p, err := client.ProfileConfig(rs.Primary.ID)
		if err != nil {
			return err
		}

		if p != nil {
			*profile = *p
			return nil
		}

		return fmt.Errorf("Profile not found: %s", rs.Primary.ID)
	}
}

func testAccProfileConfig(profile *shared.ProfileConfig, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if profile.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range profile.Config {
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

func testAccProfileDevice(profile *shared.ProfileConfig, deviceName string, device shared.Device) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if profile.Devices == nil {
			return fmt.Errorf("No devices")
		}

		if profile.Devices.Contains(deviceName, device) {
			return nil
		}

		return fmt.Errorf("Device not found: %s", deviceName)
	}
}

func testAccProfileNoDevice(profile *shared.ProfileConfig, deviceName string, device shared.Device) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if profile.Devices == nil {
			return nil
		}

		if profile.Devices.Contains(deviceName, device) {
			return fmt.Errorf("Device still exists: %s", deviceName)
		}

		return nil
	}
}

func testAccProfile_basic(name string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
  name = "%s"
}`, name)
}

func testAccProfile_config(name string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
  name = "%s"
  config {
    limits.cpu = 2
  }
}`, name)
}

func testAccProfile_device_1(name string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
	name = "%s"

	device {
		name = "shared"
		type = "disk"
		properties {
			source = "/tmp"
			path = "/tmp/shared"
		}
	}
}`, name)
}

func testAccProfile_device_2(name string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
	name = "%s"

	device {
		name = "shared"
		type = "disk"
		properties {
			source = "/tmp"
			path = "/tmp/shared2"
		}
	}
}`, name)
}

func testAccProfile_addDevice_1(name string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
	name = "%s"
}`, name)
}

func testAccProfile_addDevice_2(name string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
	name = "%s"

	device {
		name = "shared"
		type = "disk"
		properties {
			source = "/tmp"
			path = "/tmp/shared"
		}
	}
}`, name)
}

func testAccProfile_removeDevice_1(name string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
	name = "%s"

	device {
		name = "shared"
		type = "disk"
		properties {
			source = "/tmp"
			path = "/tmp/shared"
		}
	}
}`, name)
}

func testAccProfile_removeDevice_2(name string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
	name = "%s"
}`, name)
}

func testAccProfile_containerBasic(profileName, containerName string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
	name = "%s"
}

resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default", "${lxd_profile.profile1.name}"]
}`, profileName, containerName)
}

func testAccProfile_containerConfig(profileName, containerName string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
  name = "%s"
  config {
    limits.cpu = 2
  }
}

resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default", "${lxd_profile.profile1.name}"]
}`, profileName, containerName)

}
func testAccProfile_containerDevice(profileName, containerName string) string {
	return fmt.Sprintf(`resource "lxd_profile" "profile1" {
  name = "%s"
	device {
		name = "shared"
		type = "disk"
		properties {
			source = "/tmp"
			path = "/tmp/shared"
		}
	}
}

resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default", "${lxd_profile.profile1.name}"]
}`, profileName, containerName)
}
