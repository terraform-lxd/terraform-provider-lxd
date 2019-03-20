package lxd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/lxc/lxd/shared/api"
)

func TestAccProfileBasicConfig(t *testing.T) {
	var profile api.Profile
	profileName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfileBasicConfig(profileName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
				),
			},
		},
	})
}

func TestAccProfileSetConfig(t *testing.T) {
	var profile api.Profile
	profileName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfileSetConfig(profileName),
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
	var profile api.Profile
	profileName := strings.ToLower(petname.Generate(2, "-"))

	device1 := map[string]string{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared",
	}

	device2 := map[string]string{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared2",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfileDevice1Config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.1834377448.properties.path", "/tmp/shared"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileDevice(&profile, "shared", device1),
				),
			},
			resource.TestStep{
				Config: testAccProfileDevice2Config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.2643642920.properties.path", "/tmp/shared2"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileDevice(&profile, "shared", device2),
				),
			},
		},
	})
}

func TestAccProfile_addDevice(t *testing.T) {
	var profile api.Profile
	profileName := strings.ToLower(petname.Generate(2, "-"))

	device1 := map[string]string{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared1",
	}

	device2 := map[string]string{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared2",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfileAddDevice1Config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
				),
			},
			resource.TestStep{
				Config: testAccProfileAddDevice2Config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.3028205791.properties.path", "/tmp/shared1"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileDevice(&profile, "shared1", device1),
				),
			},
			resource.TestStep{
				Config: testAccProfileAddDevice3Config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.1620449630.properties.path", "/tmp/shared2"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileDevice(&profile, "shared1", device1),
					testAccProfileDevice(&profile, "shared2", device2),
				),
			},
		},
	})
}

func TestAccProfile_removeDevice(t *testing.T) {
	var profile api.Profile
	profileName := strings.ToLower(petname.Generate(2, "-"))

	device := map[string]string{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfileRemoveDevice1Config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.1834377448.properties.path", "/tmp/shared"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileDevice(&profile, "shared", device),
				),
			},
			resource.TestStep{
				Config: testAccProfileRemoveDevice2Config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccProfileNoDevice(&profile, "shared"),
				),
			},
		},
	})
}

func TestAccProfile_containerConfig(t *testing.T) {
	var profile api.Profile
	var container api.Container
	profileName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfileContainerConfig(profileName, containerName),
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
	var profile api.Profile
	var container api.Container
	profileName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	device := map[string]string{
		"type":   "disk",
		"source": "/tmp",
		"path":   "/tmp/shared",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfileContainerDeviceConfig(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.1834377448.properties.path", "/tmp/shared"),
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccProfileDevice(&profile, "shared", device),
					testAccContainerExpandedDevice(&container, "shared", device),
				),
			},
		},
	})
}

func testAccProfileRunning(t *testing.T, n string, profile *api.Profile) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := testAccProvider.Meta().(*lxdProvider).GetContainerServer("")
		if err != nil {
			return err
		}
		p, _, err := client.GetProfile(rs.Primary.ID)
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

func testAccProfileConfig(profile *api.Profile, k, v string) resource.TestCheckFunc {
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

func testAccProfileDevice(profile *api.Profile, deviceName string, device map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if profile.Devices == nil {
			return fmt.Errorf("No devices")
		}

		if v, ok := profile.Devices[deviceName]; ok {
			if reflect.DeepEqual(v, device) {
				return nil
			}
		}

		return fmt.Errorf("Device not found: %s", deviceName)
	}
}

func testAccProfileNoDevice(profile *api.Profile, deviceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if profile.Devices == nil {
			return nil
		}

		if _, ok := profile.Devices[deviceName]; ok {
			return fmt.Errorf("Device still exists: %s", deviceName)
		}

		return nil
	}
}

func testAccProfileBasicConfig(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}
	`, name)
}

func testAccProfileSetConfig(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
  config {
    limits.cpu = 2
  }
}
	`, name)
}

func testAccProfileDevice1Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
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
	`, name)
}

func testAccProfileDevice2Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"

  device {
    name = "shared"
    type = "disk"
    properties {
      source = "/tmp"
      path = "/tmp/shared2"
    }
  }
}
	`, name)
}

func testAccProfileAddDevice1Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}
`, name)
}

func testAccProfileAddDevice2Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"

  device {
    name = "shared1"
    type = "disk"
    properties {
      source = "/tmp"
      path = "/tmp/shared1"
    }
  }
}
	`, name)
}

func testAccProfileAddDevice3Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"

  device {
    name = "shared2"
    type = "disk"
    properties {
      source = "/tmp"
      path = "/tmp/shared2"
    }
  }

  device {
    name = "shared1"
    type = "disk"
    properties {
      source = "/tmp"
      path = "/tmp/shared1"
    }
  }
}
	`, name)
}

func testAccProfileRemoveDevice1Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
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
	`, name)
}

func testAccProfileRemoveDevice2Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}
`, name)
}

func testAccProfileContainerConfig(profileName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
  config {
    limits.cpu = 2
  }
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9"
  profiles = ["default", "${lxd_profile.profile1.name}"]
}
	`, profileName, containerName)
}

func testAccProfileContainerDeviceConfig(profileName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
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
  image = "images:alpine/3.9"
  profiles = ["default", "${lxd_profile.profile1.name}"]
}
	`, profileName, containerName)
}
