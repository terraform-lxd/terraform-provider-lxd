package lxd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/lxc/lxd/shared/api"
)

func TestAccContainer_basic(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_basic(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
				),
			},
		},
	})
}

func TestAccContainer_remoteImage(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_remoteImage(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
				),
			},
		},
	})
}

func TestAccContainer_config(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_config(containerName),
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
			{
				Config: testAccContainer_addProfile_1(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerProfile(&container, "default"),
				),
			},
			{
				Config: testAccContainer_addProfile_2(profileName, containerName),
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
			{
				Config: testAccContainer_removeProfile_1(profileName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerProfile(&container, "default"),
					testAccContainerProfile(&container, profileName),
				),
			},
			{
				Config: testAccContainer_removeProfile_2(profileName, containerName),
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
			{
				Config: testAccContainer_device_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.1834377448.properties.path", "/tmp/shared"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device1),
				),
			},
			{
				Config: testAccContainer_device_2(containerName),
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
			{
				Config: testAccContainer_addDevice_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
			{
				Config: testAccContainer_addDevice_2(containerName),
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
			{
				Config: testAccContainer_removeDevice_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.1834377448.properties.path", "/tmp/shared"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device),
				),
			},
			{
				Config: testAccContainer_removeDevice_2(containerName),
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
			{
				Config: testAccContainer_fileUploadContent_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
			{
				Config: testAccContainer_fileUploadContent_2(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
		},
	})
}

func TestAccContainer_fileUploadSource(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_fileUploadSource(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
		},
	})
}

func TestAccContainer_defaultProfile(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_defaultProfile(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "profiles.0", "default"),
					testAccContainerProfile(&container, "default"),
				),
			},
		},
	})
}

func TestAccContainer_configLimits(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_configLimits_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "limits.cpu", "1"),
				),
			},
			{
				Config: testAccContainer_configLimits_2(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "limits.cpu", "2"),
				),
			},
		},
	})
}

func TestAccContainer_accessInterface(t *testing.T) {
	var container api.Container
	networkName1 := strings.ToLower(petname.Generate(1, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_accessInterface(networkName1, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "ip_address", "10.150.19.200"),
					resource.TestCheckResourceAttr("lxd_container.container1", "ipv4_address", "10.150.19.200"),
					resource.TestCheckResourceAttr("lxd_container.container1", "ipv6_address", "fd42:474b:622d:259d:216:3eff:fe39:7f36"),
				),
			},
		},
	})
}

func TestAccContainer_withDevice(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_withDevice(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
				),
			},
		},
	})
}

func TestAccContainer_isStopped(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_isStopped(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerState(t, "lxd_container.container1", &container, api.Stopped),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
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

		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
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

func testAccContainerState(t *testing.T, n string, container *api.Container, state api.StatusCode) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		ct, _, err := client.GetContainer(rs.Primary.ID)
		if err != nil {
			return err
		}

		if ct != nil {
			if ct.StatusCode != state {
				return fmt.Errorf("Wrong container state. Container has: %s", ct.StatusCode)
			}
			*container = *ct
			return nil
		}

		return fmt.Errorf("Container not found: %s", rs.Primary.ID)
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

func testAccContainer_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainer_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
  config = {
    "boot.autostart" = 1
  }
}
	`, name)
}

func testAccContainer_addProfile_1(profileName, containerName string) string {
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

func testAccContainer_addProfile_2(profileName, containerName string) string {
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

func testAccContainer_removeProfile_1(profileName, containerName string) string {
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

func testAccContainer_removeProfile_2(profileName, containerName string) string {
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

func testAccContainer_device_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path = "/tmp/shared"
    }
  }
}
	`, name)
}

func testAccContainer_device_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path = "/tmp/shared2"
    }
  }
}
	`, name)
}

func testAccContainer_addDevice_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainer_addDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path = "/tmp/shared"
    }
  }
}
	`, name)
}

func testAccContainer_removeDevice_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path = "/tmp/shared"
    }
  }
}
	`, name)
}

func testAccContainer_removeDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainer_fileUploadContent_1(name string) string {
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

func testAccContainer_fileUploadContent_2(name string) string {
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

func testAccContainer_fileUploadSource(name string) string {
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

func testAccContainer_remoteImage(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainer_defaultProfile(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9"
}
	`, name)
}

func testAccContainer_configLimits_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  limits = {
	  "cpu" = "1"
  }
}
	`, name)
}

func testAccContainer_configLimits_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  limits = {
	  "cpu" = "2"
  }
}
	`, name)
}

func testAccContainer_accessInterface(networkName1, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network_1" {
  name = "%s"

  config = {
    "ipv4.address" = "10.150.19.1/24"
    "ipv4.nat" = "true"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
    "ipv6.nat" = "false"
  }
}

resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  config = {
    "user.access_interface" = "eth0"
  }

  device {
    name = "eth0"
    type = "nic"

    properties = {
      nictype = "bridged"
      parent = "${lxd_network.network_1.name}"
      hwaddr = "00:16:3e:39:7f:36"
      "ipv4.address" = "10.150.19.200"
    }
  }
}
	`, networkName1, containerName)
}

func testAccContainer_withDevice(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  device {
    name = "foo"
    type = "nic"
    properties = {
      name    = "bar"
      nictype = "bridged"
      parent  = "lxdbr0"
    }
  }
}
	`, name)
}

func testAccContainer_isStopped(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]

  start_container = false
}
	`, name)
}
