package profile_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccProfile_DS_basic(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_DS_basic(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "name", profileName),
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "description", "Terraform provider test profile"),
				),
			},
		},
	})
}

func TestAccProfile_DS_config(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_DS_config(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "name", profileName),
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "config.limits.cpu", "2"),
				),
			},
		},
	})
}

func TestAccProfile_DS_device(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_DS_device(profileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "name", profileName),
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "device.0.name", "shared"),
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "device.0.properties.path", "/tmp/shared"),
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "device.0.properties.source", "/tmp"),
				),
			},
		},
	})
}

func TestAccProfile_DS_project(t *testing.T) {
	profileName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_DS_project(profileName, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "name", profileName),
					resource.TestCheckResourceAttr("data.lxd_profile.profile", "project", projectName),
				),
			},
		},
	})
}

func testAccProfile_DS_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile" {
  name        = %q
  description = "Terraform provider test profile"
}

data "lxd_profile" "profile" {
  name = lxd_profile.profile.name
}
	`, name)
}

func testAccProfile_DS_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile" {
  name = %q

  config = {
    "limits.cpu" = 2
  }
}

data "lxd_profile" "profile" {
  name = lxd_profile.profile.name
}
	`, name)
}

func testAccProfile_DS_device(name string) string {
	return fmt.Sprintf(`
resource "lxd_profile" "profile" {
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

data "lxd_profile" "profile" {
  name = lxd_profile.profile.name
}
	`, name)
}

func testAccProfile_DS_project(name string, project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "proj" {
  name        = %[2]q
}

resource "lxd_profile" "profile" {
  name        = %[1]q
  project     = lxd_project.proj.name
  description = "Terraform provider test profile"
}

data "lxd_profile" "profile" {
  name    = lxd_profile.profile.name
  project = lxd_profile.profile.project
}
	`, name, project)
}
