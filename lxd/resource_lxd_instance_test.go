package lxd

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/canonical/lxd/shared/api"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccInstance_basic(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
				),
			},
		},
	})
}

func TestAccInstance_basicEphemeral(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_basicEphemeral(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
				),
			},
		},
	})
}

func TestAccInstance_typeContainer(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_container(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "type", "container"),
				),
			},
		},
	})
}

func TestAccInstance_typeVirtualMachine(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckVirtualization(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_virtualmachine(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "type", "virtual-machine"),
				),
			},
		},
	})
}

func TestAccInstance_remoteImage(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_remoteImage(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
				),
			},
		},
	})
}

func TestAccInstance_config(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_config(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.boot.autostart", "1"),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					testAccInstanceConfig(&instance, "boot.autostart", "1"),
				),
			},
		},
	})
}

func TestAccInstance_updateConfig(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_updateConfig1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.boot.autostart", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.user.dummy", "5"),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					testAccInstanceConfig(&instance, "boot.autostart", "1"),
					testAccInstanceConfig(&instance, "user.dummy", "5"),
					testAccInstanceConfigAbsent(&instance, "user.user-data"),
				),
			},
			{
				Config: testAccInstance_updateConfig2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.user.dummy", "5"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.user.user-data", "#cloud-config"),
					testAccInstanceConfigAbsent(&instance, "boot.autostart"),
					testAccInstanceConfig(&instance, "user.dummy", "5"),
					testAccInstanceConfig(&instance, "user.user-data", "#cloud-config"),
				),
			},
		},
	})
}

func TestAccInstance_addProfile(t *testing.T) {
	var profile api.Profile
	var instance api.Instance
	profileName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_addProfile_1(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					testAccInstanceProfile(&instance, "default"),
				),
			},
			{
				Config: testAccInstance_addProfile_2(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					testAccInstanceProfile(&instance, "default"),
					testAccInstanceProfile(&instance, profileName),
				),
			},
		},
	})
}

func TestAccInstance_removeProfile(t *testing.T) {
	var profile api.Profile
	var instance api.Instance
	profileName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_removeProfile_1(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					testAccInstanceProfile(&instance, "default"),
					testAccInstanceProfile(&instance, profileName),
				),
			},
			{
				Config: testAccInstance_removeProfile_2(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					testAccInstanceProfile(&instance, "default"),
					testAccInstanceNoProfile(&instance, profileName),
				),
			},
		},
	})
}

func TestAccInstance_device(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

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
				Config: testAccInstance_device_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_instance.instance1", "device.*", map[string]string{"properties.path": "/tmp/shared"}),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					testAccInstanceDevice(&instance, "shared", device1),
				),
			},
			{
				Config: testAccInstance_device_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_instance.instance1", "device.*", map[string]string{"properties.path": "/tmp/shared2"}),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					testAccInstanceDevice(&instance, "shared", device2),
				),
			},
		},
	})
}

func TestAccInstance_addDevice(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

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
				Config: testAccInstance_addDevice_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
				),
			},
			{
				Config: testAccInstance_addDevice_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_instance.instance1", "device.*", map[string]string{"properties.path": "/tmp/shared"}),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					testAccInstanceDevice(&instance, "shared", device),
				),
			},
		},
	})
}

func TestAccInstance_removeDevice(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

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
				Config: testAccInstance_removeDevice_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_instance.instance1", "device.*", map[string]string{"properties.path": "/tmp/shared"}),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					testAccInstanceDevice(&instance, "shared", device),
				),
			},
			{
				Config: testAccInstance_removeDevice_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					testAccInstanceNoDevice(&instance, "shared"),
				),
			},
		},
	})
}

func TestAccInstance_fileUploadContent(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_fileUploadContent_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
				),
			},
			{
				Config: testAccInstance_fileUploadContent_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
				),
			},
		},
	})
}

func TestAccInstance_fileUploadSource(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_fileUploadSource(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
				),
			},
		},
	})
}

func TestAccInstance_defaultProfile(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_defaultProfile(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
					testAccInstanceProfile(&instance, "default"),
				),
			},
		},
	})
}

func TestAccInstance_configLimits(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_configLimits_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "limits.cpu", "1"),
				),
			},
			{
				Config: testAccInstance_configLimits_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "limits.cpu", "2"),
				),
			},
		},
	})
}

func TestAccInstance_accessInterface(t *testing.T) {
	var instance api.Instance
	networkName1 := petname.Generate(1, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_accessInterface(networkName1, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "ip_address", "10.150.19.200"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "ipv4_address", "10.150.19.200"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "ipv6_address", "fd42:474b:622d:259d:216:3eff:fe39:7f36"),
				),
			},
		},
	})
}

func TestAccInstance_withDevice(t *testing.T) {
	t.Skip("Test is failing in CI but passing locally")

	var instance api.Instance
	instanceName := petname.Generate(2, "-")

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
				Config: testAccInstance_withDevice(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					testAccInstanceDevice(&instance, "foo", device),
				),
			},
		},
	})
}

func TestAccInstance_isStopped(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_isStopped(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceState(t, "lxd_instance.instance1", &instance, api.Stopped),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
				),
			},
		},
	})
}

func TestAccInstance_target(t *testing.T) {
	var instance api.Instance
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckClustering(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_target(instanceName, "node-2"),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					testAccInstanceRunning(t, "lxd_instance.instance2", &instance),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "target", "node-2"),
					resource.TestCheckResourceAttr("lxd_instance.instance2", "target", "node-2"),
				),
			},
		},
	})
}

func TestAccInstance_createProject(t *testing.T) {
	var instance api.Instance
	var project api.Project
	instanceName := petname.Generate(2, "-")
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_project(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccInstanceRunningInProject(t, "lxd_instance.instance1", &instance, projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
				),
			},
		},
	})
}

func TestAccInstance_removeProject(t *testing.T) {
	var project api.Project
	var instance api.Instance
	projectName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_removeProject_1(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccInstanceRunningInProject(t, "lxd_instance.instance1", &instance, projectName),
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "project", projectName),
				),
			},
			{
				Config: testAccInstance_removeProject_2(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
				),
			},
		},
	})
}

func testAccInstanceRunning(t *testing.T, n string, instance *api.Instance) resource.TestCheckFunc {
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

		inst, _, err := client.GetInstance(rs.Primary.ID)
		if err != nil {
			return err
		}

		if inst != nil {
			*instance = *inst
			return nil
		}

		return fmt.Errorf("Instance not found: %s", rs.Primary.ID)
	}
}

func testAccInstanceRunningInProject(t *testing.T, n string, instance *api.Instance, project string) resource.TestCheckFunc {
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
		inst, _, err := client.GetInstance(rs.Primary.ID)
		if err != nil {
			return err
		}

		if inst != nil {
			*instance = *inst
			return nil
		}

		return fmt.Errorf("Instance not found: %s", rs.Primary.ID)
	}
}

func testAccInstanceConfig(instance *api.Instance, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range instance.Config {
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

func testAccInstanceConfigAbsent(instance *api.Instance, k string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Config == nil {
			return fmt.Errorf("No config")
		}

		for key := range instance.Config {
			if k == key {
				return fmt.Errorf("Config key present, but be absent: %s", k)
			}
		}

		return nil
	}
}

func testAccInstanceState(t *testing.T, n string, instance *api.Instance, state api.StatusCode) resource.TestCheckFunc {
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

		inst, _, err := client.GetInstance(rs.Primary.ID)
		if err != nil {
			return err
		}

		if inst != nil {
			if inst.StatusCode != state {
				return fmt.Errorf("Wrong instance state. Instance has: %s", inst.StatusCode)
			}
			*instance = *inst
			return nil
		}

		return fmt.Errorf("Instance not found: %s", rs.Primary.ID)
	}
}

func testAccInstanceExpandedConfig(instance *api.Instance, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.ExpandedConfig == nil {
			return fmt.Errorf("No expanded config")
		}

		for key, value := range instance.ExpandedConfig {
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

func testAccInstanceProfile(instance *api.Instance, profile string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Profiles == nil {
			return fmt.Errorf("No profiles")
		}

		for _, v := range instance.Profiles {
			if v == profile {
				return nil
			}
		}

		return fmt.Errorf("Profile not found: %s", profile)
	}
}

func testAccInstanceNoProfile(instance *api.Instance, profileName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Profiles == nil {
			return fmt.Errorf("No profiles")
		}

		for _, v := range instance.Profiles {
			if v == profileName {
				return fmt.Errorf("Profile still attached to instance: %s", profileName)
			}
		}

		return nil
	}
}

func testAccInstanceDevice(instance *api.Instance, deviceName string, device map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Devices == nil {
			return fmt.Errorf("No devices")
		}

		if v, ok := instance.Devices[deviceName]; ok {
			if reflect.DeepEqual(v, device) {
				return nil
			}
		}

		return fmt.Errorf("Device not found: %s", deviceName)
	}
}

func testAccInstanceExpandedDevice(instance *api.Instance, deviceName string, device map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.ExpandedDevices == nil {
			return fmt.Errorf("No expanded devices")
		}

		if v, ok := instance.ExpandedDevices[deviceName]; ok {
			if reflect.DeepEqual(v, device) {
				return nil
			}
		}

		return fmt.Errorf("Expanded Device not found: %s", deviceName)
	}
}

func testAccInstanceNoDevice(instance *api.Instance, deviceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Devices == nil {
			return nil
		}

		if _, ok := instance.Devices[deviceName]; ok {
			return fmt.Errorf("Device still exists: %s", deviceName)
		}

		return nil
	}
}

func testAccInstance_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccInstance_basicEphemeral(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
  ephemeral = true
}
	`, name)
}

func testAccInstance_container(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  type = "container"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccInstance_virtualmachine(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  type = "virtual-machine"
  image = "images:alpine/3.18/amd64"
  # alpine images do not support secureboot
  config = {
    "security.secureboot" = false
  }
  profiles = ["default"]
}
	`, name)
}

func testAccInstance_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
  config = {
    "boot.autostart" = 1
  }
}
	`, name)
}

func testAccInstance_updateConfig1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default"]
  config = {
    "boot.autostart" = 1
	"user.dummy" = 5
  }
}
	`, name)
}

func testAccInstance_updateConfig2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default"]
  config = {
	"user.dummy" = 5
    "user.user-data" = "#cloud-config"
  }
}
	`, name)
}

func testAccInstance_addProfile_1(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default"]
}
	`, profileName, instanceName)
}

func testAccInstance_addProfile_2(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default", "${lxd_profile.profile1.name}"]
}
	`, profileName, instanceName)
}

func testAccInstance_removeProfile_1(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default", "${lxd_profile.profile1.name}"]
}
	`, profileName, instanceName)
}

func testAccInstance_removeProfile_2(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default"]
}
	`, profileName, instanceName)
}

func testAccInstance_device_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccInstance_device_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccInstance_addDevice_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccInstance_addDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccInstance_removeDevice_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccInstance_removeDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccInstance_fileUploadContent_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccInstance_fileUploadContent_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccInstance_fileUploadSource(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccInstance_remoteImage(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
}
	`, name)
}

func testAccInstance_defaultProfile(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
}
	`, name)
}

func testAccInstance_configLimits_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]

  limits = {
	  "cpu" = "1"
  }
}
	`, name)
}

func testAccInstance_configLimits_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]

  limits = {
	  "cpu" = "2"
  }
}
	`, name)
}

func testAccInstance_accessInterface(networkName1, instanceName string) string {
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

resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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
	`, networkName1, instanceName)
}

func testAccInstance_withDevice(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccInstance_isStopped(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]

  start_on_create = false
}
	`, name)
}

func testAccInstance_target(name string, target string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s-1"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
  target = "%s"
}

resource "lxd_instance" "instance2" {
  name = "%s-2"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
  target = "%s"
}
	`, name, target, name, target)
}

func testAccInstance_project(projectName string, instanceName string) string {
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
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  project = lxd_project.project1.name
}
	`, projectName, instanceName)
}

func testAccInstance_removeProject_1(projectName, instanceName string) string {
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
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  project = lxd_project.project1.name
}
	`, projectName, instanceName)
}

func testAccInstance_removeProject_2(projectName, instanceName string) string {
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
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
}
	`, projectName, instanceName)
}
