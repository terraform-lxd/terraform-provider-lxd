package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/lxc/lxd/shared/api"
)

//At a high level, the first basic test for a resource should establish the following:
//- Terraform can plan and apply a common resource configuration without error.
//- Verify the expected attributes are saved to state, and contain the values expected.
//- Verify the values in the remote API/Service for the resource match what is stored in state.
//- Verify that a subsequent terraform plan does not produce a diff/change.

func TestAccProject_basic(t *testing.T) {
	var project api.Project
	projectName := strings.ToLower(petname.Generate(2, "-"))

	// https://github.com/hashicorp/terraform-plugin-sdk/blob/main/helper/resource/testing.go
	// https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/acceptance-tests/testcase
	resource.Test(t, resource.TestCase{
		// PreCheck, if non-nil, will be called before any test steps are
		// executed. It will only be executed in the case that the steps
		// would run, so it can be used for some validation before running
		// acceptance tests, such as verifying that keys are setup.
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		// Terraform runs plan, apply, refresh, and then final plan for each TestStep in the TestCase
		// If the last plan results in a non-empty plan, Terraform will exit with an error.
		Steps: []resource.TestStep{
			{
				Config: testAccProject_basic(projectName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project0", &project),
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
					testAccProjectRunning(t, "lxd_project.project1", &project),
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

func testAccProjectRunning(t *testing.T, n string, project *api.Project) resource.TestCheckFunc {
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

		return fmt.Errorf("Project not found: %s", rs.Primary.ID)
	}
}

func testAccProjectConfig(project *api.Project, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if project.Config == nil {
			return fmt.Errorf("No config")
		}

		for key, value := range project.Config {
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

func testAccProject_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project0" {
  name = "%s"
  description = "Terraform provider test project"
  config = {
	"features.images" = true
	"features.profiles" = true
	"features.storage.volumes" = true
	"features.storage.buckets" = true
  }
}`, name)
}

func testAccProject_config(name string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
  description = "Terraform provider test project"
  config = {
	"features.images" = false
	"features.profiles" = false
	"features.storage.volumes" = false
	"features.storage.buckets" = false
  }
}`, name)
}
