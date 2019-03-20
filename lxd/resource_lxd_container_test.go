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

func TestAccContainerBasicConfig(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerBasicConfig(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
				),
			},
		},
	})
}

func TestAccContainerRemoteImageConfig(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerRemoteImageConfig(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
				),
			},
		},
	})
}

func TestAccContainerGetConfig(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerGetConfig(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "config.boot.autostart", "1"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerConfig(&container, "boot.autostart", "1"),
				),
			},
		},
	})
}

func TestAccContainer_addProfile(t *testing.T) {
	var profile api.Profile
	var container api.Container
	profileName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerAddProfile1Config(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerProfile(&container, "default"),
				),
			},
			resource.TestStep{
				Config: testAccContainerAddProfile2Config(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerProfile(&container, "default"),
					testAccContainerProfile(&container, profileName),
				),
			},
		},
	})
}

func TestAccContainer_removeProfile(t *testing.T) {
	var profile api.Profile
	var container api.Container
	profileName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerRemoveProfile1Config(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerProfile(&container, "default"),
					testAccContainerProfile(&container, profileName),
				),
			},
			resource.TestStep{
				Config: testAccContainerRemoveProfile2Config(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerProfile(&container, "default"),
					testAccContainerNoProfile(&container, profileName),
				),
			},
		},
	})
}

func TestAccContainer_device(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

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
				Config: testAccContainerDevice1Config(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.1834377448.properties.path", "/tmp/shared"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device1),
				),
			},
			resource.TestStep{
				Config: testAccContainerDevice2Config(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.2643642920.properties.path", "/tmp/shared2"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device2),
				),
			},
		},
	})
}

func TestAccContainer_addDevice(t *testing.T) {
	var container api.Container
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
				Config: testAccContainerAddDevice1Config(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
			resource.TestStep{
				Config: testAccContainerAddDevice2Config(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.1834377448.properties.path", "/tmp/shared"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device),
				),
			},
		},
	})
}

func TestAccContainer_removeDevice(t *testing.T) {
	var container api.Container
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
				Config: testAccContainerRemoveDevice1Config(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.1834377448.properties.path", "/tmp/shared"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device),
				),
			},
			resource.TestStep{
				Config: testAccContainerRemoveDevice2Config(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerNoDevice(&container, "shared"),
				),
			},
		},
	})
}

func TestAccContainer_fileUploadContent(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerFileUploadContent1Config(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
			resource.TestStep{
				Config: testAccContainerFileUploadContent2Config(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
		},
	})
}

func TestAccContainerFileUploadSourceConfig(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerFileUploadSourceConfig(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
		},
	})
}

func TestAccContainerDefaultProfileConfig(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerDefaultProfileConfig(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "profiles.0", "default"),
					testAccContainerProfile(&container, "default"),
				),
			},
		},
	})
}

func TestAccContainerGetConfigLimits(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerGetConfigLimits1(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "limits.cpu", "1"),
				),
			},
			resource.TestStep{
				Config: testAccContainerGetConfigLimits2(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "limits.cpu", "2"),
				),
			},
		},
	})
}

func TestAccContainerAccessInterface(t *testing.T) {
	var container api.Container
	networkName1 := strings.ToLower(petname.Generate(1, "-"))
	networkName2 := strings.ToLower(petname.Generate(1, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerAccessInterfaceConfig(networkName1, networkName2, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "ip_address", "10.150.19.200"),
				),
			},
		},
	})
}

func testAccContainerRunning(t *testing.T, n string, container *api.Container) resource.TestCheckFunc {
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
		ct, _, err := client.GetContainer(rs.Primary.ID)
		if err != nil {
			return err
		}

		if ct != nil {
			*container = *ct
			return nil
		}

		return fmt.Errorf("Container not found: %s", rs.Primary.ID)
	}
}

func testAccContainerConfig(container *api.Container, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range container.Config {
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

func testAccContainerExpandedConfig(container *api.Container, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.ExpandedConfig == nil {
			return fmt.Errorf("No expanded config")
		}

		for key, value := range container.ExpandedConfig {
			if k != key {
				continue
			}

			if v == value {
				return nil
			}

			return fmt.Errorf("Bad value for %s: %s", k, value)
		}

		return fmt.Errorf("Expanded Config not found: %s", k)
	}
}

func testAccContainerProfile(container *api.Container, profile string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.Profiles == nil {
			return fmt.Errorf("No profiles")
		}

		for _, v := range container.Profiles {
			if v == profile {
				return nil
			}
		}

		return fmt.Errorf("Profile not found: %s", profile)
	}
}

func testAccContainerNoProfile(container *api.Container, profileName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.Profiles == nil {
			return fmt.Errorf("No profiles")
		}

		for _, v := range container.Profiles {
			if v == profileName {
				return fmt.Errorf("Profile still attached to container: %s", profileName)
			}
		}

		return nil
	}
}

func testAccContainerDevice(container *api.Container, deviceName string, device map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.Devices == nil {
			return fmt.Errorf("No devices")
		}

		if v, ok := container.Devices[deviceName]; ok {
			if reflect.DeepEqual(v, device) {
				return nil
			}
		}

		return fmt.Errorf("Device not found: %s", deviceName)
	}
}

func testAccContainerExpandedDevice(container *api.Container, deviceName string, device map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.ExpandedDevices == nil {
			return fmt.Errorf("No expanded devices")
		}

		if v, ok := container.ExpandedDevices[deviceName]; ok {
			if reflect.DeepEqual(v, device) {
				return nil
			}
		}

		return fmt.Errorf("Expanded Device not found: %s", deviceName)
	}
}

func testAccContainerNoDevice(container *api.Container, deviceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.Devices == nil {
			return nil
		}

		if _, ok := container.Devices[deviceName]; ok {
			return fmt.Errorf("Device still exists: %s", deviceName)
		}

		return nil
	}
}

func testAccContainerBasicConfig(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainerGetConfig(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
  config {
    boot.autostart = 1
  }
}
	`, name)
}

func testAccContainerAddProfile1Config(profileName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9"
  profiles = ["default"]
}
	`, profileName, containerName)
}

func testAccContainerAddProfile2Config(profileName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9"
  profiles = ["default", "${lxd_profile.profile1.name}"]
}
	`, profileName, containerName)
}

func testAccContainerRemoveProfile1Config(profileName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9"
  profiles = ["default", "${lxd_profile.profile1.name}"]
}
	`, profileName, containerName)
}

func testAccContainerRemoveProfile2Config(profileName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9"
  profiles = ["default"]
}
	`, profileName, containerName)
}

func testAccContainerDevice1Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

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

func testAccContainerDevice2Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

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

func testAccContainerAddDevice1Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainerAddDevice2Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

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

func testAccContainerRemoveDevice1Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

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

func testAccContainerRemoveDevice2Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainerFileUploadContent1Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  file {
    content = "Hello, World!\n"
    target_file = "/foo/bar.txt"
    mode = "0644"
    create_directories = true
  }
}
	`, name)
}

func testAccContainerFileUploadContent2Config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  file {
    content = "Goodbye, World!\n"
    target_file = "/foo/bar.txt"
    mode = "0644"
    create_directories = true
  }
}
	`, name)
}

func testAccContainerFileUploadSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  file {
    source = "test-fixtures/test-file.txt"
    target_file = "/foo/bar.txt"
    mode = "0644"
    create_directories = true
  }
}
	`, name)
}

func testAccContainerRemoteImageConfig(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainerDefaultProfileConfig(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9"
}
	`, name)
}

func testAccContainerGetConfigLimits1(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  limits {
	  "cpu" = "1"
  }
}
	`, name)
}

func testAccContainerGetConfigLimits2(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  limits {
	  "cpu" = "2"
  }
}
	`, name)
}

func testAccContainerAccessInterfaceConfig(networkName1, networkName2, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network_1" {
  name = "%s"

  config {
    ipv4.address = "10.150.19.1/24"
    ipv4.nat = "true"
    ipv6.address = "fd42:474b:622d:259d::1/64"
    ipv6.nat = "false"
  }
}

resource "lxd_network" "network_2" {
  name = "%s"

  config {
    ipv4.address = "10.150.18.1/24"
    ipv4.nat = "true"
    ipv6.address = "fd42:474b:622d:259c::1/64"
    ipv6.nat = "false"
  }
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  config {
    user.access_interface = "eth0"
  }

  device {
    name = "eth0"
    type = "nic"

    properties {
      nictype = "bridged"
      parent = "${lxd_network.network_1.name}"
      ipv4.address = "10.150.19.200"
    }
  }

  device {
    name = "eth1"
    type = "nic"

    properties {
      nictype = "bridged"
      parent = "${lxd_network.network_2.name}"
      ipv4.address = "10.150.18.200"
    }
  }

}
	`, networkName1, networkName2, containerName)
}
