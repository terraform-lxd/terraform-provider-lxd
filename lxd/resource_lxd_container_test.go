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

func TestAccContainerBasic(t *testing.T) {
	var container shared.ContainerInfo
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerBasic(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
				),
			},
		},
	})
}

func TestAccContainerConfig(t *testing.T) {
	var container shared.ContainerInfo
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerWConfig(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "config.limits.cpu", "2"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerConfig(&container, "limits.cpu", "2"),
				),
			},
		},
	})
}

func TestAccContainer_profile(t *testing.T) {
	var container shared.ContainerInfo
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainer_profile_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "profiles.0", "default"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerProfile(&container, "default"),
				),
			},
			resource.TestStep{
				Config: testAccContainer_profile_2(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "profiles.1", "docker"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerProfile(&container, "docker"),
				),
			},
		},
	})
}

func TestAccContainer_device(t *testing.T) {
	var container shared.ContainerInfo
	containerName := strings.ToLower(petname.Generate(2, "-"))

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
				Config: testAccContainer_device_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.0.properties.path", "/tmp/shared"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device1),
				),
			},
			resource.TestStep{
				Config: testAccContainer_device_2(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.0.properties.path", "/tmp/shared2"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device2),
				),
			},
		},
	})
}

func TestAccContainer_addDevice(t *testing.T) {
	var container shared.ContainerInfo
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
				Config: testAccContainer_addDevice_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
			resource.TestStep{
				Config: testAccContainer_addDevice_2(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.0.properties.path", "/tmp/shared"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device),
				),
			},
		},
	})
}

func TestAccContainer_removeDevice(t *testing.T) {
	var container shared.ContainerInfo
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
				Config: testAccContainer_removeDevice_1(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "device.0.properties.path", "/tmp/shared"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerDevice(&container, "shared", device),
				),
			},
			resource.TestStep{
				Config: testAccContainer_removeDevice_2(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerNoDevice(&container, "shared", device),
				),
			},
		},
	})
}

func testAccContainerRunning(t *testing.T, n string, container *shared.ContainerInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*APIClient).Client
		ct, err := client.ContainerInfo(rs.Primary.ID)
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

func testAccContainerConfig(container *shared.ContainerInfo, k, v string) resource.TestCheckFunc {
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

func testAccContainerExpandedConfig(container *shared.ContainerInfo, k, v string) resource.TestCheckFunc {
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

func testAccContainerProfile(container *shared.ContainerInfo, profile string) resource.TestCheckFunc {
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

func testAccContainerDevice(container *shared.ContainerInfo, deviceName string, device shared.Device) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.Devices == nil {
			return fmt.Errorf("No devices")
		}

		if container.Devices.Contains(deviceName, device) {
			return nil
		}

		return fmt.Errorf("Device not found: %s", deviceName)
	}
}

func testAccContainerExpandedDevice(container *shared.ContainerInfo, deviceName string, device shared.Device) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.ExpandedDevices == nil {
			return fmt.Errorf("No expanded devices")
		}

		if container.ExpandedDevices.Contains(deviceName, device) {
			return nil
		}

		return fmt.Errorf("Expanded Device not found: %s", deviceName)
	}
}

func testAccContainerNoDevice(container *shared.ContainerInfo, deviceName string, device shared.Device) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.Devices == nil {
			return nil
		}

		if container.Devices.Contains(deviceName, device) {
			return fmt.Errorf("Device still exists: %s", deviceName)
		}

		return nil
	}
}

func testAccContainerBasic(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
  name = "%s"
  image = "ubuntu"
  profiles = ["default"]
}`, name)
}

func testAccContainerWConfig(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
  name = "%s"
  image = "ubuntu"
  profiles = ["default"]
  config {
    limits.cpu = 2
  }
}`, name)
}

func testAccContainer_profile_1(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default"]
}`, name)
}

func testAccContainer_profile_2(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default", "docker"]
}`, name)
}

func testAccContainer_device_1(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default"]

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

func testAccContainer_device_2(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default"]

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

func testAccContainer_addDevice_1(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default"]
}`, name)
}

func testAccContainer_addDevice_2(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default"]

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

func testAccContainer_removeDevice_1(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default"]

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

func testAccContainer_removeDevice_2(name string) string {
	return fmt.Sprintf(`resource "lxd_container" "container1" {
	name = "%s"
	image = "ubuntu"
	profiles = ["default"]
}`, name)
}
