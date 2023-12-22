package instance_test

import (
	"fmt"
	"regexp"
	"strings"
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
					resource.TestCheckResourceAttr("lxd_instance.instance1", "image", acctest.TestImage),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
				),
			},
		},
	})
}

func TestAccInstance_ephemeral(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_ephemeral(instanceName),
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

func TestAccInstance_ephemeralStopped(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccInstance_ephemeralStopped(instanceName),
				ExpectError: regexp.MustCompile(fmt.Sprintf("Instance %q is ephemeral and cannot be stopped", instanceName)),
			},
		},
	})
}

func TestAccInstance_container(t *testing.T) {
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
					resource.TestCheckResourceAttr("lxd_instance.instance1", "running", "false"),
				),
			},
		},
	})
}

func TestAccInstance_virtualMachine(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckVirtualization(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_virtualMachine(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "type", "virtual-machine"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
				),
			},
		},
	})
}

func TestAccInstance_virtualMachineNoDevLxd(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckVirtualization(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_virtualMachineNoDevLxd(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "type", "virtual-machine"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "config.security.devlxd", "false"),
				),
			},
		},
	})
}

func TestAccInstance_restartContainer(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	instanceType := "container"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_started(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "type", instanceType),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "running", "true"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "mac_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv6_address"),
				),
			},
			{
				Config: testAccInstance_stopped(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "running", "false"),
				),
			},
			{
				// Verifies that instance is started with network.
				Config: testAccInstance_started(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "running", "true"),
				),
			},
		},
	})
}

func TestAccInstance_restartVirtualMachine(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	instanceType := "virtual-machine"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckVirtualization(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_started(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "type", instanceType),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "running", "true"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "mac_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv6_address"),
				),
			},
			{
				Config: testAccInstance_stopped(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "running", "false"),
				),
			},
			{
				Config: testAccInstance_started(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "running", "true"),
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
					resource.TestCheckResourceAttr("lxd_instance.instance1", "image", acctest.TestImage),
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
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.target_path", "/foo/bar.txt"),
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
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.target_path", "/foo/bar.txt"),
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
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.target_path", "/foo/bar.txt"),
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
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.source_path", "../acctest/fixtures/test-file.txt"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.target_path", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.0.create_directories", "true"),
				),
			},
		},
	})
}

func TestAccInstance_execOutput(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_exec(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.exit_code", "0"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stdout", ""),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stderr", ""),
				),
			},
			{
				Config: testAccInstance_execOutput(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.exit_code", "0"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stdout", "Linux\n"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stderr", ""),
				),
			},
			{
				Config: testAccInstance_exec(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.exit_code", "0"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stdout", ""),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stderr", ""),
				),
			},
		},
	})
}

func TestAccInstance_execOnStoppedInstance(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Try to add exec blocks on instance that will not be started.
				Config:      testAccInstance_execOnStoppedInstance(instanceName),
				ExpectError: regexp.MustCompile(fmt.Sprintf("Instance %q is planned to be stopped, but exec commands need to be run", instanceName)),
			},
			{
				// Start an instance with exec command.
				Config: testAccInstance_exec(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
				),
			},
			{
				// Try to change exec block while stopping the instance.
				Config:      testAccInstance_execOnStoppedInstance(instanceName, "trigger"),
				ExpectError: regexp.MustCompile(fmt.Sprintf("Instance %q is planned to be stopped, but exec commands need to be run", instanceName)),
			},
			{
				// Stop it without changing exec blocks.
				Config: testAccInstance_execOnStoppedInstance(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
				),
			},
			{
				// Try to change exec block while instance is stopped.
				Config:      testAccInstance_execOnStoppedInstance(instanceName, "trigger"),
				ExpectError: regexp.MustCompile(fmt.Sprintf("Instance %q is planned to be stopped, but exec commands need to be run", instanceName)),
			},
		},
	})
}

func TestAccInstance_execWorkingDir(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_execWorkingDir(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.exit_code", "0"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stdout", "ID=alpine"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stderr", ""),
				),
			},
		},
	})
}

func TestAccInstance_execEnvironment(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_execEnvironment(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.exit_code", "0"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stdout", "It works."),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stderr", ""),
				),
			},
		},
	})
}

func TestAccInstance_execScript(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_execScript(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.exit_code", "0"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stdout", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stderr", ""),
				),
			},
		},
	})
}

func TestAccInstance_execError(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{

		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure command error is recorded and terraform apply is
				// not disturbed if fail_on_error is unset (or set to false).
				Config: testAccInstance_execError_1(instanceName, false /* fail on error */),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
					// resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.exit_code", "127"), // TODO: Requires LXD client 5.20.
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stdout", ""),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stderr", "Command not found"),
				),
				ExpectNonEmptyPlan: true, // timestamp() in triggers.
			},
			{
				// Ensure terraform apply fails on command error
				// if fail_on_error is set to true.
				Config:      testAccInstance_execError_1(instanceName, true /* fail on error */),
				ExpectError: regexp.MustCompile("Error: Failed to execute command"),
			},
			{
				// Ensures exit_code is set even if output is not recorded.
				Config: testAccInstance_execError_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.exit_code", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stdout", ""),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "exec.0.stderr", ""),
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

func TestAccInstance_target(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_target(instanceName, "node-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", fmt.Sprintf("%s-1", instanceName)),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "image", acctest.TestImage),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "target", "node-2"),
					resource.TestCheckResourceAttr("lxd_instance.instance2", "name", fmt.Sprintf("%s-2", instanceName)),
					resource.TestCheckResourceAttr("lxd_instance.instance2", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance2", "image", acctest.TestImage),
					resource.TestCheckResourceAttr("lxd_instance.instance2", "target", "node-2"),
				),
			},
		},
	})
}

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

func TestAccInstance_importBasic(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	resourceName := "lxd_instance.instance1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_basic(instanceName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s,image=%s", instanceName, acctest.TestImage),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func TestAccInstance_importProject(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	projectName := petname.Generate(2, "-")
	resourceName := "lxd_instance.instance1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_project(projectName, instanceName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/%s,image=%s", projectName, instanceName, acctest.TestImage),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func testAccInstance_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, name, acctest.TestImage)
}

func testAccInstance_ephemeral(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name      = "%s"
  image     = "%s"
  profiles  = ["default"]
  ephemeral = true
}
	`, name, acctest.TestImage)
}

func testAccInstance_ephemeralStopped(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name      = "%s"
  image     = "%s"
  running   = false
  ephemeral = true
}`, name, acctest.TestImage)
}

func testAccInstance_container(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  type    = "container"
  running = false
}
	`, name, acctest.TestImage)
}

func testAccInstance_virtualMachine(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
  type  = "virtual-machine"

  config = {
    # Alpine images do not support secureboot.
    "security.secureboot" = false
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_virtualMachineNoDevLxd(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
  type  = "virtual-machine"

  config = {
    # Alpine images do not support secureboot.
    "security.secureboot" = false
    "security.devlxd"     = false
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_started(name string, instanceType string) string {
	var config string
	if instanceType == "virtual-machine" {
		config = `"security.secureboot" = false`
	}

	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  type    = "%s"
  running = true

  config = {
    %s
  }
}
	`, name, acctest.TestImage, instanceType, config)
}

func testAccInstance_stopped(name string, instanceType string) string {
	var config string
	if instanceType == "virtual-machine" {
		config = `"security.secureboot" = false`
	}

	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  type    = "%s"
  running = false

  config = {
    %s
  }
}
	`, name, acctest.TestImage, instanceType, config)
}

func testAccInstance_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
  config = {
    "boot.autostart" = 1
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_updateConfig1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
  config = {
    "user.dummy"     = 5
    "boot.autostart" = 1
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_updateConfig2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
  config = {
    "user.dummy"     = 5
    "user.user-data" = "#cloud-config"
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_updateConfig3(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
  config = {
    "user.dummy"             = 5
    "user.user-data"         = "#cloud-config"
    "cloud-init.vendor-data" = "#cloud-config"
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_addProfile_1(instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, instanceName, acctest.TestImage)
}

func testAccInstance_addProfile_2(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default", lxd_profile.profile1.name]
}
	`, profileName, instanceName, acctest.TestImage)
}

func testAccInstance_removeProfile_1(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default", lxd_profile.profile1.name]
}
	`, profileName, instanceName, acctest.TestImage)
}

func testAccInstance_removeProfile_2(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default"]
}
	`, profileName, instanceName, acctest.TestImage)
}

func testAccInstance_noProfile(name string) string {
	return fmt.Sprintf(`
resource "lxd_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "zfs"
}

resource "lxd_instance" "instance1" {
  name             = "%[1]s"
  image            = "%s"
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
	`, name, acctest.TestImage)
}

func testAccInstance_device_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared"
    }
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_device_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared2"
    }
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_addDevice_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, name, acctest.TestImage)
}

func testAccInstance_addDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared"
    }
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_removeDevice_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared"
    }
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_removeDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, name, acctest.TestImage)
}

func testAccInstance_fileUploadContent_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  file {
    content            = "Hello, World!\n"
    target_path        = "/foo/bar.txt"
    mode               = "0644"
    create_directories = true
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_fileUploadContent_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  file {
    content            = "Hello, World!\n"
    target_path        = "/foo/bar.txt"
    mode               = "0777"
    create_directories = true
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_fileUploadContent_3(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  file {
    content            = "Goodbye, World!\n"
    target_path        = "/foo/bar.txt"
    mode               = "0777"
    create_directories = false
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_fileUploadSource(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  file {
    source_path        = "../acctest/fixtures/test-file.txt"
    target_path        = "/foo/bar.txt"
    mode               = "0644"
    create_directories = true
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_exec(instanceName string, triggers ...string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  exec {
    command       = ["uname"]
    triggers      = ["%s"]
    record_output = false
  }
}
	`, instanceName, acctest.TestImage, strings.Join(triggers, "\", \""))
}

func testAccInstance_execOnStoppedInstance(instanceName string, triggers ...string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  running = false

  exec {
    command       = ["uname"]
    triggers      = ["%s"]
    record_output = false
  }
}
	`, instanceName, acctest.TestImage, strings.Join(triggers, "\", \""))
}

func testAccInstance_execOutput(instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  exec {
    command       = ["uname"]
    triggers      = ["rerun"]
    record_output = true
  }
}
	`, instanceName, acctest.TestImage)
}

func testAccInstance_execWorkingDir(instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  exec {
    command = [
      "/bin/sh", "-c",
      "cat os-release | grep '^ID' | tr -d '\n'"
    ]
    working_dir   = "/etc"
    record_output = true
  }
}
	`, instanceName, acctest.TestImage)
}

func testAccInstance_execEnvironment(instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  exec {
    command       = ["/bin/sh", "-c", "echo -n $ENV_TEST"]
    record_output = true

    environment = {
      "ENV_TEST" = "It works."
    }
  }
}
	`, instanceName, acctest.TestImage)
}

func testAccInstance_execScript(instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  file {
    source_path = "../acctest/fixtures/test-script.sh"
    target_path = "/root/test-script.sh"
    mode        = "0700"
  }

  exec {
    command       = ["/bin/sh", "test-script.sh"]
    record_output = true
  }
}
	`, instanceName, acctest.TestImage)
}

func testAccInstance_execError_1(instanceName string, failOnError bool) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  exec {
    command       = ["invalid"]
    triggers      = [timestamp()]
    record_output = true
    fail_on_error = "%v"
  }
}
	`, instanceName, acctest.TestImage, failOnError)
}

func testAccInstance_execError_2(instanceName string, triggers ...string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  exec {
    command = ["/bin/sh", "-c", "ls / | grep 'nothing'"]
  }
}
	`, instanceName, acctest.TestImage)
}

func testAccInstance_remoteImage(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, name, acctest.TestImage)
}

func testAccInstance_configLimits_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  limits = {
    "cpu" = 1
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_configLimits_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

  limits = {
    "cpu"    = 2
    "memory" = "128MiB"
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_accessInterface(networkName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network1" {
  name = "%s"

  config = {
    "ipv4.address" = "10.150.19.1/24"
  }
}

resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"

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
	`, networkName, instanceName, acctest.TestImage)
}

func testAccInstance_target(name string, target string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name   = "%[1]s-1"
  image  = "%[3]s"
  target = "%[2]s"
}

resource "lxd_instance" "instance2" {
  name   = "%[1]s-2"
  image  = "%[3]s"
  target = "%[2]s"
}
	`, name, target, acctest.TestImage)
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
  name  = "%s"
  image = "%s"
  project = lxd_project.project1.name
}
	`, projectName, instanceName, acctest.TestImage)
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
  name    = "%s"
  image   = "%s"
  project = lxd_project.project1.name
}
	`, projectName, instanceName, acctest.TestImage)
}

func testAccInstance_removeProject_2(projectName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
}

resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, projectName, instanceName, acctest.TestImage)
}
