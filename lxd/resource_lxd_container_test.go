package lxd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

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

func TestAccContainer_basicEphemeral(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_basicEphemeral(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
				),
			},
		},
	})
}

func TestAccContainer_typeContainer(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_type(containerName, "container"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "type", "container"),
				),
			},
		},
	})
}

func TestAccContainer_typeVirtualMachine(t *testing.T) {
	t.Skip("Travis CI environment does not support virtualization")

	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_type(containerName, "virtual-machine"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "type", "virtual-machine"),
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

func TestAccContainer_updateConfig(t *testing.T) {
	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_updateConfig1(containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "config.boot.autostart", "1"),
					resource.TestCheckResourceAttr("lxd_container.container1", "config.user.dummy", "5"),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerConfig(&container, "boot.autostart", "1"),
					testAccContainerConfig(&container, "user.dummy", "5"),
					testAccContainerConfigAbsent(&container, "user.user-data"),
				),
			},
			{
				Config: testAccContainer_updateConfig2(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "config.user.dummy", "5"),
					resource.TestCheckResourceAttr("lxd_container.container1", "config.user.user-data", "#cloud-config"),
					testAccContainerConfigAbsent(&container, "boot.autostart"),
					testAccContainerConfig(&container, "user.dummy", "5"),
					testAccContainerConfig(&container, "user.user-data", "#cloud-config"),
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
	t.Skip("Test is failing in CI but passing locally")

	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	device := map[string]string{
		"type":    "nic",
		"name":    "bar",
		"nictype": "bridged",
		"parent":  "lxdbr0",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_withDevice(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccContainerDevice(&container, "foo", device),
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

func TestAccContainer_target(t *testing.T) {
	t.Skip("Test environment does not support clustering yet")

	var container api.Container
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_target(containerName, "node-2"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
					testAccContainerRunning(t, "lxd_container.container2", &container),
					resource.TestCheckResourceAttr("lxd_container.container1", "target", "node-2"),
					resource.TestCheckResourceAttr("lxd_container.container2", "target", "node-2"),
				),
			},
		},
	})
}

func TestAccContainer_createProject(t *testing.T) {
	var container api.Container
	var project api.Project
	containerName := strings.ToLower(petname.Generate(2, "-"))
	projectName := strings.ToLower(petname.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_project(projectName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccContainerRunningInProject(t, "lxd_container.container1", &container, projectName),
					resource.TestCheckResourceAttr("lxd_container.container1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
				),
			},
		},
	})
}

func TestAccContainer_removeProject(t *testing.T) {
	var project api.Project
	var container api.Container
	projectName := strings.ToLower(petname.Generate(2, "-"))
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_removeProject_1(projectName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccContainerRunningInProject(t, "lxd_container.container1", &container, projectName),
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					resource.TestCheckResourceAttr("lxd_container.container1", "project", projectName),
				),
			},
			{
				Config: testAccContainer_removeProject_2(projectName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccContainerRunning(t, "lxd_container.container1", &container),
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
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

func testAccContainerRunningInProject(t *testing.T, n string, container *api.Container, project string) resource.TestCheckFunc {
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

		client = client.UseProject(project)
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

func testAccContainerConfigAbsent(container *api.Container, k string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if container.Config == nil {
			return fmt.Errorf("No config")
		}

		for key := range container.Config {
			if k == key {
				return fmt.Errorf("Config key present, but be absent: %s", k)
			}
		}

		return nil
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
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainer_basicEphemeral(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
  ephemeral = true
}
	`, name)
}

func testAccContainer_type(name string, cType string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  type = "%s"
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
}
	`, name, cType)
}

func testAccContainer_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
  config = {
    "boot.autostart" = 1
  }
}
	`, name)
}

func testAccContainer_updateConfig1(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16"
  profiles = ["default"]
  config = {
    "boot.autostart" = 1
	"user.dummy" = 5
  }
}
	`, name)
}

func testAccContainer_updateConfig2(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16"
  profiles = ["default"]
  config = {
	"user.dummy" = 5
    "user.user-data" = "#cloud-config"
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
  image = "images:alpine/3.16"
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
  image = "images:alpine/3.16"
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
  image = "images:alpine/3.16"
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
  image = "images:alpine/3.16"
  profiles = ["default"]
}
	`, profileName, containerName)
}

func testAccContainer_device_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainer_addDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainer_fileUploadContent_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccContainer_defaultProfile(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16"
}
	`, name)
}

func testAccContainer_configLimits_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
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
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]

  start_container = false
}
	`, name)
}

func testAccContainer_target(name string, target string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s-1"
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
  target = "%s"
}

resource "lxd_container" "container2" {
  name = "%s-2"
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
  target = "%s"
}
	`, name, target, name, target)
}

func testAccContainer_project(projectName string, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.profiles" = false
	"features.storage.buckets" = false
  }
}
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
  project = lxd_project.project1.name
}
	`, projectName, containerName)
}

func testAccContainer_removeProject_1(projectName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.profiles" = false
	"features.storage.buckets" = false
  }
}
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
  project = lxd_project.project1.name
}
	`, projectName, containerName)
}

func testAccContainer_removeProject_2(projectName, containerName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.profiles" = false
	"features.storage.buckets" = false
  }
}
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
}
	`, projectName, containerName)
}
