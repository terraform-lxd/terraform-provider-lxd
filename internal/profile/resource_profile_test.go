package profile_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/canonical/lxd/shared/api"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccProfile_basic(t *testing.T) {
	// var profile api.Profile
	profileName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_basic(profileName),
				Check: resource.ComposeTestCheckFunc(
					// testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "description", ""),
				),
			},
		},
	})
}

func TestAccProfile_config(t *testing.T) {
	// var profile api.Profile
	profileName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_config(profileName),
				Check: resource.ComposeTestCheckFunc(
					// testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					// testAccProfileConfig(&profile, "limits.cpu", "2"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "description", "My profile"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "config.limits.cpu", "2"),
				),
			},
		},
	})
}

// TODO:
//   - Verify both devices exist
func TestAccProfile_device(t *testing.T) {
	// var profile api.Profile
	profileName := petname.Generate(2, "-")

	// device1 := map[string]string{
	// 	"type":   "disk",
	// 	"source": "/tmp",
	// 	"path":   "/tmp/shared",
	// }

	// device2 := map[string]string{
	// 	"type":   "disk",
	// 	"source": "/tmp2",
	// 	"path":   "/tmp/shared2",
	// }

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_device_1(profileName),
				Check: resource.ComposeTestCheckFunc(
					// testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					// testAccProfileDevice(&profile, "shared", device1),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.path", "/tmp/shared"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.source", "/tmp"),
				),
			},
			{
				Config: testAccProfile_device_2(profileName),
				Check: resource.ComposeTestCheckFunc(
					// testAccProfileRunning(t, "lxd_profile.profile1", &profile),
					// testAccProfileDevice(&profile, "shared", device2),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
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
	profileName := petname.Generate(2, "-")

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
	profileName := petname.Generate(2, "-")

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

// TODO:
// - requires instance resource
// func TestAccProfile_instanceConfig(t *testing.T) {
// 	// var profile api.Profile
// 	// var instance api.Instance
// 	profileName := petname.Generate(2, "-")
// 	instanceName := petname.Generate(2, "-")

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:                 func() { acctest.PreCheck(t) },
// 		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccProfile_instanceConfig(profileName, instanceName),
// 				Check: resource.ComposeTestCheckFunc(
// 					// testAccProfileRunning(t, "lxd_profile.profile1", &profile),
// 					// testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
// 					// testAccProfileConfig(&profile, "limits.cpu", "2"),
// 					// testAccInstanceExpandedConfig(&instance, "limits.cpu", "2"),
// 					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
// 					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
// 					resource.TestCheckResourceAttr("lxd_profile.profile1", "config.limits.cpu", "2"),
// 				),
// 			},
// 		},
// 	})
// }

// TODO:
// - requires instance resource
// func TestAccProfile_instanceDevice(t *testing.T) {
// 	// var profile api.Profile
// 	// var instance api.Instance
// 	profileName := petname.Generate(2, "-")
// 	instanceName := petname.Generate(2, "-")

// 	// device := map[string]string{
// 	// 	"type":   "disk",
// 	// 	"source": "/tmp",
// 	// 	"path":   "/tmp/shared",
// 	// }

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:                 func() { acctest.PreCheck(t) },
// 		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccProfile_instanceDevice(profileName, instanceName),
// 				Check: resource.ComposeTestCheckFunc(
// 					// testAccProfileRunning(t, "lxd_profile.profile1", &profile),
// 					// testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
// 					// testAccProfileDevice(&profile, "shared", device),
// 					// testAccInstanceExpandedDevice(&instance, "shared", device),
// 					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
// 					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
// 					resource.TestCheckTypeSetElemNestedAttrs("lxd_profile.profile1", "device.*", map[string]string{"properties.path": "/tmp/shared"}),
// 				),
// 			},
// 		},
// 	})
// }

// TODO:
// - requires instance resource
// func TestAccProfile_instanceDevice_2(t *testing.T) {
// 	t.Skip("Test is failing in CI but passing locally")

// 	// var profile api.Profile
// 	// var instance api.Instance
// 	profileName := petname.Generate(2, "-")
// 	instanceName := petname.Generate(2, "-")

// 	// device := map[string]string{
// 	// 	"type":    "nic",
// 	// 	"name":    "bar",
// 	// 	"nictype": "bridged",
// 	// 	"parent":  "lxdbr0",
// 	// }

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:                 func() { acctest.PreCheck(t) },
// 		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccProfile_instanceDevice_2(profileName, instanceName),
// 				Check: resource.ComposeTestCheckFunc(
// 					// testAccProfileRunning(t, "lxd_profile.profile1", &profile),
// 					// testAccInstanceRunning(t, "lxd_instance.instance1", &instance),
// 					// testAccProfileDevice(&profile, "foo", device),
// 					// testAccInstanceExpandedDevice(&instance, "foo", device),
// 					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
// 					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
// 				),
// 			},
// 		},
// 	})
// }

func TestAccProfile_project(t *testing.T) {
	// var profile api.Profile
	// var project api.Project
	profileName := petname.Generate(2, "-")
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_project(projectName, profileName),
				Check:  resource.ComposeTestCheckFunc(
				// testAccProjectRunning(t, "lxd_project.project1", &project),
				// testAccProfileRunningInProject(t, "lxd_profile.profile1", &profile, projectName),
				),
			},
		},
	})
}

// func testAccProfileRunning(t *testing.T, n string, profile *api.Profile) resource.TestCheckFunc {
// 	return func(s *terraform.State) error {
// 		rs, ok := s.RootModule().Resources[n]
// 		if !ok {
// 			return fmt.Errorf("Not found: %s", n)
// 		}

// 		if rs.Primary.ID == "" {
// 			return fmt.Errorf("No ID is set")
// 		}

// 		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
// 		if err != nil {
// 			return err
// 		}
// 		p, _, err := client.GetProfile(rs.Primary.ID)
// 		if err != nil {
// 			return err
// 		}

// 		if p != nil {
// 			*profile = *p
// 			return nil
// 		}

// 		return fmt.Errorf("Profile not found: %s", rs.Primary.ID)
// 	}
// }

// func testAccProfileRunningInProject(t *testing.T, n string, profile *api.Profile, projectName string) resource.TestCheckFunc {
// 	return func(s *terraform.State) error {
// 		rs, ok := s.RootModule().Resources[n]
// 		if !ok {
// 			return fmt.Errorf("Not found: %s", n)
// 		}

// 		if rs.Primary.ID == "" {
// 			return fmt.Errorf("No ID is set")
// 		}

// 		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
// 		if err != nil {
// 			return err
// 		}
// 		client = client.UseProject(projectName)
// 		p, _, err := client.GetProfile(rs.Primary.ID)
// 		if err != nil {
// 			return err
// 		}

// 		if p != nil {
// 			*profile = *p
// 			return nil
// 		}

// 		return fmt.Errorf("Profile not found: %s", rs.Primary.ID)
// 	}
// }

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
  name = "%s"
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
      path = "/tmp/shared"
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
      path = "/tmp/shared2"
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
      path = "/tmp/shared1"
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
      path = "/tmp/shared2"
    }
  }

  device {
    name = "shared1"
    type = "disk"
    properties = {
      source = "/tmp"
      path = "/tmp/shared1"
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
      path = "/tmp/shared"
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

// func testAccProfile_instanceConfig(profileName, instanceName string) string {
// 	return fmt.Sprintf(`
// resource "lxd_profile" "profile1" {
//   name = "%s"
//   config = {
//     "limits.cpu" = 2
//   }
// }

// resource "lxd_instance" "instance1" {
//   name = "%s"
//   image = "images:alpine/3.18"
//   profiles = ["default", "${lxd_profile.profile1.name}"]
// }
// 	`, profileName, instanceName)
// }

// func testAccProfile_instanceDevice(profileName, instanceName string) string {
// 	return fmt.Sprintf(`
// resource "lxd_profile" "profile1" {
//   name = "%s"
//   device {
//     name = "shared"
//     type = "disk"
//     properties = {
//       source = "/tmp"
//       path = "/tmp/shared"
//     }
//   }
// }

// resource "lxd_instance" "instance1" {
//   name = "%s"
//   image = "images:alpine/3.18"
//   profiles = ["default", "${lxd_profile.profile1.name}"]
// }
// 	`, profileName, instanceName)
// }

// func testAccProfile_instanceDevice_2(profileName, instanceName string) string {
// 	return fmt.Sprintf(`
// resource "lxd_profile" "profile1" {
//   name = "%s"

//   device {
//     name = "foo"
//     type = "nic"
//     properties = {
//       name    = "bar"
//       nictype = "bridged"
//       parent  = "lxdbr0"
//     }
//   }
// }

// resource "lxd_instance" "instance1" {
//   name = "%s"
//   image = "images:alpine/3.18"
//   profiles = ["default", "${lxd_profile.profile1.name}"]
// }
// 	`, profileName, instanceName)
// }

func testAccProfile_project(projectName, profileName string) string {
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
resource "lxd_profile" "profile1" {
  name = "%s"
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
