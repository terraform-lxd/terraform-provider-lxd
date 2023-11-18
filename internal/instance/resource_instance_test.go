package instance_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccInstance_basic(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "ephemeral", "false"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "image", "images:alpine/3.18/amd64"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
				),
			},
		},
	})
}

func TestAccInstance_basicEphemeral(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_basicEphemeral(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "ephemeral", "true"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
				),
			},
		},
	})
}

func TestAccInstance_typeContainer(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_container(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "type", "container"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "start_on_create", "false"),
				),
			},
		},
	})
}

func TestAccInstance_typeVirtualMachine(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_virtualmachine(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "type", "virtual-machine"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
				),
			},
		},
	})
}

func TestAccInstance_remoteImage(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_remoteImage(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "image", "images:alpine/3.18/amd64"),
				),
			},
		},
	})
}

func TestAccInstance_config(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_config(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.boot.autostart", "1"),
				),
			},
		},
	})
}

func TestAccInstance_updateConfig(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_updateConfig1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.boot.autostart", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.user.dummy", "5"),
				),
			},
			{
				Config: testAccInstance_updateConfig2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.user.dummy", "5"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.user.user-data", "#cloud-config"),
					resource.TestCheckNoResourceAttr("lxd_instance.instance1", "config.boot.autostart"),
				),
			},
			{
				Config: testAccInstance_updateConfig3(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.user.dummy", "5"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.user.user-data", "#cloud-config"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.cloud-init.vendor-data", "#cloud-config"),
				),
			},
		},
	})
}

func TestAccInstance_addProfile(t *testing.T) {
	profileName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_addProfile_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
				),
			},
			{
				Config: testAccInstance_addProfile_2(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "2"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.1", profileName),
				),
			},
		},
	})
}

func TestAccInstance_removeProfile(t *testing.T) {
	profileName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_removeProfile_1(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "2"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.1", profileName),
				),
			},
			{
				Config: testAccInstance_removeProfile_2(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
				),
			},
		},
	})
}

func TestAccInstance_noProfile(t *testing.T) {
	name := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_noProfile(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "name", name),
					resource.TestCheckResourceAttr("lxd_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", name),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "0"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.pool", name),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.path", "/"),
				),
			},
		},
	})
}

func TestAccInstance_device(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_device_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.source", "/tmp"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.path", "/tmp/shared"),
				),
			},
			{
				Config: testAccInstance_device_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.source", "/tmp"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.path", "/tmp/shared2"),
				),
			},
		},
	})
}

func TestAccInstance_addDevice(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_addDevice_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "0"),
				),
			},
			{
				Config: testAccInstance_addDevice_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.source", "/tmp"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.path", "/tmp/shared"),
				),
			},
		},
	})
}

func TestAccInstance_removeDevice(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_removeDevice_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.source", "/tmp"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.path", "/tmp/shared"),
				),
			},
			{
				Config: testAccInstance_removeDevice_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "0"),
				),
			},
		},
	})
}

func TestAccInstance_fileUploadContent(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_fileUploadContent_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.mode", "0644"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.content", "Hello, World!\n"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.target_file", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.create_directories", "true"),
				),
			},
			{
				Config: testAccInstance_fileUploadContent_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.mode", "0777"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.content", "Hello, World!\n"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.target_file", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.create_directories", "true"),
				),
			},
			{
				Config: testAccInstance_fileUploadContent_3(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.mode", "0777"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.content", "Goodbye, World!\n"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.target_file", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.create_directories", "false"),
				),
			},
		},
	})
}

func TestAccInstance_fileUploadSource(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_fileUploadSource(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.mode", "0644"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.source", "../acctest/test-fixtures/test-file.txt"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.target_file", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.create_directories", "true"),
				),
			},
		},
	})
}

func TestAccInstance_configLimits(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_configLimits_1(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "limits.%", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "limits.cpu", "1"),
				),
			},
			{
				Config: testAccInstance_configLimits_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "limits.%", "2"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "limits.cpu", "2"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "limits.memory", "128MiB"),
				),
			},
		},
	})
}

func TestAccInstance_accessInterface(t *testing.T) {
	networkName1 := petname.Generate(1, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_accessInterface(networkName1, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network1", "name", networkName1),
					resource.TestCheckResourceAttr("lxd_network.network1", "config.ipv4.address", "10.150.19.1/24"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.user.access_interface", "eth0"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.name", "eth0"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.type", "nic"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.nictype", "bridged"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.parent", networkName1),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.hwaddr", "00:16:3e:39:7f:36"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.ipv4.address", "10.150.19.200"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "mac_address", "00:16:3e:39:7f:36"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "ipv4_address", "10.150.19.200"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv6_address"),
				),
			},
		},
	})
}

// TODO:
// - Precheck clustering.
// func TestAccInstance_target(t *testing.T) {
// 	instanceName := petname.Generate(2, "-")

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:                 func() { acctest.PreCheck(t) },
// 		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccInstance_target(instanceName, "node-2"),
// 				Check: resource.ComposeTestCheckFunc(
// 					resource.TestCheckResourceAttr("lxd_instance.instance1", "target", "node-2"),
// 					resource.TestCheckResourceAttr("lxd_instance.instance2", "target", "node-2"),
// 				),
// 			},
// 		},
// 	})
// }

func TestAccInstance_createProject(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_project(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
				),
			},
		},
	})
}

func TestAccInstance_removeProject(t *testing.T) {
	projectName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_removeProject_1(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
				),
			},
			{
				Config: testAccInstance_removeProject_2(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckNoResourceAttr("lxd_instance.instance1", "project"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
				),
			},
		},
	})
}

func testAccInstance_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name     = "%s"
  image    = "images:alpine/3.18/amd64"
}
	`, name)
}

func testAccInstance_basicEphemeral(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name      = "%s"
  image     = "images:alpine/3.18/amd64"
  profiles  = ["default"]
  ephemeral = true
}
	`, name)
}

func testAccInstance_container(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name            = "%s"
  type            = "container"
  image           = "images:alpine/3.18/amd64"
  start_on_create = false
}
	`, name)
}

func testAccInstance_virtualmachine(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  type  = "virtual-machine"
  image = "images:alpine/3.18/amd64"

  # Alpine images do not support secureboot
  config = {
    "security.secureboot" = false
  }
}
	`, name)
}

func testAccInstance_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"
  config = {
    "boot.autostart" = 1
  }
}
	`, name)
}

func testAccInstance_updateConfig1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18"
  config = {
    "user.dummy"     = 5
    "boot.autostart" = 1
  }
}
	`, name)
}

func testAccInstance_updateConfig2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18"
  config = {
    "user.dummy"     = 5
    "user.user-data" = "#cloud-config"
  }
}
	`, name)
}

func testAccInstance_updateConfig3(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18"
  config = {
    "user.dummy"             = 5
    "user.user-data"         = "#cloud-config"
    "cloud-init.vendor-data" = "#cloud-config"
  }
}
	`, name)
}

func testAccInstance_addProfile_1(instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18"
}
	`, instanceName)
}

func testAccInstance_addProfile_2(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name     = "%s"
  image    = "images:alpine/3.18"
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
  name     = "%s"
  image    = "images:alpine/3.18"
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
  name     = "%s"
  image    = "images:alpine/3.18"
  profiles = ["default"]
}
	`, profileName, instanceName)
}

func testAccInstance_noProfile(name string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "zfs"
}

resource "lxd_instance" "instance1" {
  name             = "%[1]s"
  image            = "images:alpine/3.18/amd64"
  profiles         = []
  wait_for_network = false

  device {
    name = "root"
    type = "disk"
    properties = {
	path = "/"
	pool = lxd_storage_pool.pool1.name
    }
  }
}
	`, name)
}

func testAccInstance_device_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared"
    }
  }
}
	`, name)
}

func testAccInstance_device_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared2"
    }
  }
}
	`, name)
}

func testAccInstance_addDevice_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"
}
	`, name)
}

func testAccInstance_addDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared"
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

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared"
    }
  }
}
	`, name)
}

func testAccInstance_removeDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"
}
	`, name)
}

func testAccInstance_fileUploadContent_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  file {
    content            = "Hello, World!\n"
    target_file        = "/foo/bar.txt"
    mode               = "0644"
    create_directories = true
  }
}
	`, name)
}

func testAccInstance_fileUploadContent_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  file {
    content            = "Hello, World!\n"
    target_file        = "/foo/bar.txt"
    mode               = "0777"
    create_directories = true
  }
}
	`, name)
}

func testAccInstance_fileUploadContent_3(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  file {
    content            = "Goodbye, World!\n"
    target_file        = "/foo/bar.txt"
    mode               = "0777"
    create_directories = false
  }
}
	`, name)
}

func testAccInstance_fileUploadSource(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  file {
    source             = "../acctest/test-fixtures/test-file.txt"
    target_file        = "/foo/bar.txt"
    mode               = "0644"
    create_directories = true
  }
}
	`, name)
}

func testAccInstance_remoteImage(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"
}
	`, name)
}

func testAccInstance_configLimits_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  limits = {
    "cpu" = 1
  }
}
	`, name)
}

func testAccInstance_configLimits_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  limits = {
    "cpu"    = 2
    "memory" = "128MiB"
  }
}
	`, name)
}

func testAccInstance_accessInterface(networkName1, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network1" {
  name = "%s"

  config = {
    "ipv4.address" = "10.150.19.1/24"
  }
}

resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"

  config = {
    "user.access_interface" = "eth0"
  }

  device {
    name = "eth0"
    type = "nic"

    properties = {
      nictype        = "bridged"
      parent         = "${lxd_network.network1.name}"
      hwaddr         = "00:16:3e:39:7f:36"
      "ipv4.address" = "10.150.19.200"
    }
  }
}
	`, networkName1, instanceName)
}

func testAccInstance_target(name string, target string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name   = "%s-1"
  image  = "images:alpine/3.18/amd64"
  target = "%s"
}

resource "lxd_instance" "instance2" {
  name   = "%s-2"
  image  = "images:alpine/3.18/amd64"
  target = "%s"
}
	`, name, target, name, target)
}

func testAccInstance_project(projectName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name   = "%s"
  config = {
    "features.images"   = false
    "features.profiles" = false
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
  name = "%s"
  config = {
	"features.images"   = false
	"features.profiles" = false
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
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18/amd64"
}
	`, projectName, instanceName)
}
