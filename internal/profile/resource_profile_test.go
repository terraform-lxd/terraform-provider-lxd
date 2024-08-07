package profile_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccProfile_basic(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_basic(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "description", ""),
				),
			},
		},
	})
}

func TestAccProfile_config(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "description", "My profile"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "config.limits.cpu", "2"),
				),
			},
		},
	})
}

func TestAccProfile_device(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_device_1(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.source", "/tmp"),
				),
			},
			{
				Config: testAccProfile_device_2(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared2"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.source", "/tmp2"),
				),
			},
		},
	})
}

func TestAccProfile_addDevice(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_addDevice_1(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "0"),
				),
			},
			{
				Config: testAccProfile_addDevice_2(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "shared1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.source", "/tmp"),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_profile.profile1", "device.*", map[string]string{"properties.path": "/tmp/shared1"}),
				),
			},
			{
				Config: testAccProfile_addDevice_3(profileName),
				Check: resource.ComposeTestCheckFunc(
					// Here we are naivly assuming devices are added to the
					// state in the same order they were added. If any test
					// fails because of this approach,
					// "resource.TestCheckTypeSetElemNestedAttrs" should be used.
					//
					// resource.TestCheckTypeSetElemNestedAttrs("lxd_profile.profile1", "device.*", map[string]string{"properties.path": "/tmp/shared2"}),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "2"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "shared1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.source", "/tmp"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.1.name", "shared2"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.1.type", "disk"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.1.properties.path", "/tmp/shared2"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.1.properties.source", "/tmp"),
				),
			},
		},
	})
}

func TestAccProfile_removeDevice(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_removeDevice_1(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.source", "/tmp"),
				),
			},
			{
				Config: testAccProfile_removeDevice_2(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "0"),
				),
			},
		},
	})
}

func TestAccProfile_instanceConfig(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_instanceConfig(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "config.%", "1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "config.limits.cpu", "2"),
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

func TestAccProfile_instanceDevice(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_instanceDevice(profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "2"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.1", profileName),
					// Instance should not track expanded config/devices (populated from profiles).
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "0"),
				),
			},
		},
	})
}

func TestAccProfile_project(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_project(projectName, profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "foo"),
				),
			},
		},
	})
}

func TestAccProfile_defaultProfile(t *testing.T) {
	projectName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure default profile cannot be managed in default project.
				Config:      testAccProfile_basic("default"),
				ExpectError: regexp.MustCompile("Cannot import existing profile"),
			},
			{
				// Ensure default profile cannot be managed in project with "features.profiles".
				Config:      testAccProfile_defaultProfile(projectName, false),
				ExpectError: regexp.MustCompile("Cannot import existing profile"),
			},
			{
				// Ensure default profile can be managed in non-default project that
				// have "features.profiles" enabled.
				Config: testAccProfile_defaultProfile(projectName, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_profile.default", "name", "default"),
					resource.TestCheckResourceAttr("lxd_profile.default", "project", projectName),
					resource.TestCheckResourceAttr("lxd_profile.default", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_profile.default", "device.0.name", "foo"),
				),
			},
		},
	})
}

func TestAccProfile_importBasic(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")
	resourceName := "lxd_profile.profile1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_basic(profileName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        profileName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccProfile_importConfig(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")
	resourceName := "lxd_profile.profile1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_config(profileName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        profileName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccProfile_importDevice(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")
	resourceName := "lxd_profile.profile1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_device_1(profileName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        profileName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccProfile_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}
	`, name)
}

func testAccProfile_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name        = "%s"
  description = "My profile"
  config = {
    "limits.cpu" = 2
  }
}
	`, name)
}

func testAccProfile_device_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"

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

func testAccProfile_device_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"

  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp2"
      path   = "/tmp/shared2"
    }
  }
}
	`, name)
}

func testAccProfile_addDevice_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}
`, name)
}

func testAccProfile_addDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"

  device {
    name = "shared1"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared1"
    }
  }
}
	`, name)
}

func testAccProfile_addDevice_3(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"

  device {
    name = "shared2"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared2"
    }
  }

  device {
    name = "shared1"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared1"
    }
  }
}
	`, name)
}

func testAccProfile_removeDevice_1(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"

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

func testAccProfile_removeDevice_2(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
}
`, name)
}

func testAccProfile_instanceConfig(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name   = "%s"
  config = {
    "limits.cpu" = 2
  }
}

resource "lxd_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default", lxd_profile.profile1.name]
}
	`, profileName, instanceName, acctest.TestImage)
}

func testAccProfile_instanceDevice(profileName, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile1" {
  name = "%s"
  device {
    name = "shared"
    type = "disk"
    properties = {
      source = "/tmp"
      path   = "/tmp/shared"
    }
  }
}

resource "lxd_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default", lxd_profile.profile1.name]
}
	`, profileName, instanceName, acctest.TestImage)
}

func testAccProfile_project(projectName string, profileName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
  config = {
	"features.images"   = false
	"features.profiles" = false
  }
}

resource "lxd_profile" "profile1" {
  name    = "%s"
  project = lxd_project.project1.name

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
	`, projectName, profileName)
}

func testAccProfile_defaultProfile(projectName string, enableProfileFeature bool) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
  config = {
	"features.images"   = false
	"features.profiles" = %v
  }
}

resource "lxd_profile" "default" {
  name    = "default"
  project = lxd_project.project1.name

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
	`, projectName, enableProfileFeature)
}
