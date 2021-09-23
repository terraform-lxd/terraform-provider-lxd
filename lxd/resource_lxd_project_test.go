package lxd

import (
	"fmt"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/lxc/lxd/shared/api"
)

func TestAccProject_basic(t *testing.T) {
	var project api.Project
	projectName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccProject_basic(projectName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectExists(t, "lxd_project.project0", &project),
					resource.TestCheckResourceAttr("lxd_project.project0", "name", projectName),
					resource.TestCheckResourceAttr("lxd_project.project0", "id", projectName),
					resource.TestCheckResourceAttr("lxd_project.project0", "description", "Terraform provider test project"),
				),
			},
		},
	})
}

func TestAccProject_config(t *testing.T) {
	var project api.Project
	projectName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccProject_config(projectName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectExists(t, "lxd_project.project1", &project),
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_project.project1", "config.features.images", "false"),
					resource.TestCheckResourceAttr("lxd_project.project1", "config.features.profiles", "false"),
					resource.TestCheckResourceAttr("lxd_project.project1", "config.features.storage.volumes", "false"),
					testAccProjectConfig(&project, "features.images", "false"),
					testAccProjectConfig(&project, "features.profiles", "false"),
					testAccProjectConfig(&project, "features.storage.volumes", "false"),
				),
			},
		},
	})
}

func testAccProject_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project0" {
	name        = "%s"
	description = "Terraform provider test project"
	config = {
		"features.images" = true
		"features.profiles" = true
		"features.storage.volumes" = true
	}
}
	`, name)
}

func testAccProject_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
	name        = "%s"
	description = "Terraform provider test project"
	config = {
		"features.images" = false
		"features.profiles" = false
		"features.storage.volumes" = false
	}
}
	`, name)
}

func testAccProjectExists(t *testing.T, n string, project *api.Project) resource.TestCheckFunc {
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
		p, _, err := client.GetProject(rs.Primary.ID)
		if err != nil {
			return err
		}

		if p != nil {
			*project = *p
			return nil
		}

		return fmt.Errorf("Profile not found: %s", rs.Primary.ID)
	}
}

func testAccProjectConfig(project *api.Project, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if project.ProjectPut.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range project.ProjectPut.Config {
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
