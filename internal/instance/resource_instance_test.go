package instance_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccInstance_basic(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_basic(instanceName, acctest.TestImage),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "ephemeral", "false"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "image", acctest.TestImage),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.0", "default"),
				),
			},
		},
	})
}

func TestAccInstance_noImage(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_noImage(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "ephemeral", "false"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.0", "default"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "ephemeral", "true"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.0", "default"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "type", "container"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "running", "false"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "type", "virtual-machine"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
				),
			},
		},
	})
}

func TestAccInstance_virtualMachineNoDevIncus(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckVirtualization(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_virtualMachineNoDevIncus(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "type", "virtual-machine"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.security.guestapi", "false"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "type", instanceType),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "running", "true"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "mac_address"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "ipv6_address"),
				),
			},
			{
				Config: testAccInstance_stopped(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "running", "false"),
				),
			},
			{
				// Verifies that instance is started with network.
				Config: testAccInstance_started(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "running", "true"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "type", instanceType),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "running", "true"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "mac_address"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "ipv6_address"),
				),
			},
			{
				Config: testAccInstance_stopped(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "running", "false"),
				),
			},
			{
				Config: testAccInstance_started(instanceName, instanceType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "running", "true"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "image", acctest.TestImage),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.boot.autostart", "1"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.boot.autostart", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.user.dummy", "5"),
				),
			},
			{
				Config: testAccInstance_updateConfig2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.user.dummy", "5"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.user.user-data", "#cloud-config"),
					resource.TestCheckNoResourceAttr("incus_instance.instance1", "config.boot.autostart"),
				),
			},
			{
				Config: testAccInstance_updateConfig3(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.user.dummy", "5"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.user.user-data", "#cloud-config"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.cloud-init.vendor-data", "#cloud-config"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.0", "default"),
				),
			},
			{
				Config: testAccInstance_addProfile_2(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "2"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.0", "default"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.1", profileName),
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
					resource.TestCheckResourceAttr("incus_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "2"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.0", "default"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.1", profileName),
				),
			},
			{
				Config: testAccInstance_removeProfile_2(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.0", "default"),
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
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", name),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", name),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "0"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.pool", name),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.path", "/"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.source", "/tmp"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.path", "/tmp/shared"),
				),
			},
			{
				Config: testAccInstance_device_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.source", "/tmp"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.path", "/tmp/shared2"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "0"),
				),
			},
			{
				Config: testAccInstance_addDevice_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.source", "/tmp"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.path", "/tmp/shared"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.source", "/tmp"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.path", "/tmp/shared"),
				),
			},
			{
				Config: testAccInstance_removeDevice_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "0"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.mode", "0644"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.content", "Hello, World!\n"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.target_path", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.create_directories", "true"),
				),
			},
			{
				Config: testAccInstance_fileUploadContent_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.mode", "0777"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.content", "Hello, World!\n"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.target_path", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.create_directories", "true"),
				),
			},
			{
				Config: testAccInstance_fileUploadContent_3(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.mode", "0777"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.content", "Goodbye, World!\n"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.target_path", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.create_directories", "false"),
				),
			},
		},
	})
}

func TestAccInstance_fileUploadContent_VM(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_fileUploadContent_VM(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "type", "virtual-machine"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.mode", "0777"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.content", "Hello from VM!\n"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.target_path", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.create_directories", "true"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.mode", "0644"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.source_path", "../acctest/fixtures/test-file.txt"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.target_path", "/foo/bar.txt"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "file.0.create_directories", "true"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.%", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.limits.cpu", "1"),
				),
			},
			{
				Config: testAccInstance_configLimits_2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.%", "2"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.limits.cpu", "2"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.limits.memory", "128MiB"),
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
					resource.TestCheckResourceAttr("incus_network.network1", "name", networkName1),
					resource.TestCheckResourceAttr("incus_network.network1", "config.ipv4.address", "10.150.19.1/24"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.%", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "config.user.access_interface", "eth0"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.name", "eth0"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.type", "nic"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.nictype", "bridged"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.parent", networkName1),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.hwaddr", "00:16:3e:39:7f:36"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.ipv4.address", "10.150.19.200"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "mac_address", "00:16:3e:39:7f:36"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "ipv4_address", "10.150.19.200"),
					resource.TestCheckResourceAttrSet("incus_instance.instance1", "ipv6_address"),
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
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", fmt.Sprintf("%s-1", instanceName)),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "image", acctest.TestImage),
					resource.TestCheckResourceAttr("incus_instance.instance1", "target", "node-2"),
					resource.TestCheckResourceAttr("incus_instance.instance2", "name", fmt.Sprintf("%s-2", instanceName)),
					resource.TestCheckResourceAttr("incus_instance.instance2", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance2", "image", acctest.TestImage),
					resource.TestCheckResourceAttr("incus_instance.instance2", "target", "node-2"),
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
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "project", projectName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
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
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "project", projectName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
				),
			},
			{
				Config: testAccInstance_removeProject_2(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckNoResourceAttr("incus_instance.instance1", "project"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
				),
			},
		},
	})
}

func TestAccInstance_importBasic(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	resourceName := "incus_instance.instance1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_basic(instanceName, acctest.TestImage),
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
	resourceName := "incus_instance.instance1"

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

func TestAccInstance_oci(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	ociImage := "docker:alpine:latest"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_basic(instanceName, ociImage),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "ephemeral", "false"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "image", ociImage),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "profiles.0", "default"),
				),
			},
		},
	})
}

func TestAccInstance_sourceInstance(t *testing.T) {
	projectName := petname.Name()
	sourceInstanceName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_sourceInstance(projectName, sourceInstanceName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance2", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance2", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance2", "ephemeral", "false"),
					resource.TestCheckResourceAttr("incus_instance.instance2", "source_instance.name", sourceInstanceName),
					resource.TestCheckResourceAttr("incus_instance.instance2", "profiles.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance2", "profiles.0", "default"),
				),
			},
		},
	})
}

func TestAccInstance_sourceInstanceWithSnapshot(t *testing.T) {
	projectName := petname.Name()
	sourceInstanceName := petname.Generate(2, "-")
	snapshotName := "snap0"
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstance_sourceInstanceWithSnapshot(projectName, sourceInstanceName, snapshotName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance2", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance2", "status", "Running"),
					resource.TestCheckResourceAttr("incus_instance.instance2", "ephemeral", "false"),
					resource.TestCheckResourceAttr("incus_instance.instance2", "source_instance.name", sourceInstanceName),
					resource.TestCheckResourceAttr("incus_instance.instance2", "source_instance.snapshot", snapshotName),
					resource.TestCheckResourceAttr("incus_instance.instance2", "profiles.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance2", "profiles.0", "default"),
				),
			},
		},
	})
}

func testAccInstance_basic(name string, image string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, name, image)
}

func testAccInstance_noImage(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name  = "%s"
  running = false
}
	`, name)
}

func testAccInstance_ephemeral(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name      = "%s"
  image     = "%s"
  profiles  = ["default"]
  ephemeral = true
}
	`, name, acctest.TestImage)
}

func testAccInstance_ephemeralStopped(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name      = "%s"
  image     = "%s"
  running   = false
  ephemeral = true
}`, name, acctest.TestImage)
}

func testAccInstance_container(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  type    = "container"
  running = false
}
	`, name, acctest.TestImage)
}

func testAccInstance_virtualMachine(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
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

func testAccInstance_virtualMachineNoDevIncus(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"
  type  = "virtual-machine"

  config = {
    # Alpine images do not support secureboot.
    "security.secureboot" = false
    "security.guestapi"   = false
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
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, instanceName, acctest.TestImage)
}

func testAccInstance_addProfile_2(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_profile" "profile1" {
  name = "%s"
}

resource "incus_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default", incus_profile.profile1.name]
}
	`, profileName, instanceName, acctest.TestImage)
}

func testAccInstance_removeProfile_1(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_profile" "profile1" {
  name = "%s"
}

resource "incus_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default", incus_profile.profile1.name]
}
	`, profileName, instanceName, acctest.TestImage)
}

func testAccInstance_removeProfile_2(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_profile" "profile1" {
  name = "%s"
}

resource "incus_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default"]
}
	`, profileName, instanceName, acctest.TestImage)
}

func testAccInstance_noProfile(name string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "zfs"
}

resource "incus_instance" "instance1" {
  name             = "%[1]s"
  image            = "%s"
  profiles         = []
  wait_for_network = false

  device {
    name = "root"
    type = "disk"
    properties = {
	path = "/"
	pool = incus_storage_pool.pool1.name
    }
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_device_1(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, name, acctest.TestImage)
}

func testAccInstance_addDevice_2(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, name, acctest.TestImage)
}

func testAccInstance_fileUploadContent_1(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
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
resource "incus_instance" "instance1" {
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

func testAccInstance_fileUploadContent_VM(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"
  type  = "virtual-machine"

  config = {
    # Alpine images do not support secureboot.
    "security.secureboot" = false
  }

  file {
    content            = "Hello from VM!\n"
    target_path        = "/foo/bar.txt"
    mode               = "0777"
    create_directories = true
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_fileUploadSource(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
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

func testAccInstance_remoteImage(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, name, acctest.TestImage)
}

func testAccInstance_configLimits_1(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"

  config = {
    "limits.cpu" = 1
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_configLimits_2(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"

  config = {
    "limits.cpu"    = 2
    "limits.memory" = "128MiB"
  }
}
	`, name, acctest.TestImage)
}

func testAccInstance_accessInterface(networkName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_network" "network1" {
  name = "%s"

  config = {
    "ipv4.address" = "10.150.19.1/24"
  }
}

resource "incus_instance" "instance1" {
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
      parent         = "${incus_network.network1.name}"
      hwaddr         = "00:16:3e:39:7f:36"
      "ipv4.address" = "10.150.19.200"
    }
  }
}
	`, networkName, instanceName, acctest.TestImage)
}

func testAccInstance_target(name string, target string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name   = "%[1]s-1"
  image  = "%[3]s"
  target = "%[2]s"
}

resource "incus_instance" "instance2" {
  name   = "%[1]s-2"
  image  = "%[3]s"
  target = "%[2]s"
}
	`, name, target, acctest.TestImage)
}

func testAccInstance_project(projectName string, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name   = "%s"
  config = {
    "features.images"   = false
    "features.profiles" = false
  }
}

resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"
  project = incus_project.project1.name
}
	`, projectName, instanceName, acctest.TestImage)
}

func testAccInstance_removeProject_1(projectName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
  config = {
	"features.images"   = false
	"features.profiles" = false
  }
}

resource "incus_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  project = incus_project.project1.name
}
	`, projectName, instanceName, acctest.TestImage)
}

func testAccInstance_removeProject_2(projectName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
}

resource "incus_instance" "instance1" {
  name  = "%s"
  image = "%s"
}
	`, projectName, instanceName, acctest.TestImage)
}

func testAccInstance_sourceInstance(projectName, sourceInstanceName string, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%[1]s"
  config = {
	"features.images"   = false
	"features.profiles" = false
  }
}

resource "incus_instance" "instance1" {
  project = incus_project.project1.name
  name  = "%[2]s"
  image = "%[4]s"
}

resource "incus_instance" "instance2" {
  project = incus_project.project1.name
  name  = "%[3]s"

  source_instance = {
    project = incus_project.project1.name
    name    = incus_instance.instance1.name
  }
}
	`, projectName, sourceInstanceName, instanceName, acctest.TestImage)
}

func testAccInstance_sourceInstanceWithSnapshot(projectName, sourceInstanceName string, snapshotName string, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%[1]s"
  config = {
	"features.images"   = false
	"features.profiles" = false
  }
}

resource "incus_instance" "instance1" {
  project = incus_project.project1.name
  name  = "%[2]s"
  image = "%[5]s"
}

resource "incus_instance_snapshot" "snapshot1" {
  name     = "%[3]s"
  instance = incus_instance.instance1.name
  stateful = false
  project  = incus_project.project1.name
}

resource "incus_instance" "instance2" {
  project = incus_project.project1.name
  name  = "%[4]s"

  source_instance = {
    project  = incus_project.project1.name
    name     = incus_instance.instance1.name
    snapshot = incus_instance_snapshot.snapshot1.name
  }
}
	`, projectName, sourceInstanceName, snapshotName, instanceName, acctest.TestImage)
}
