package project_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccProject_DS_basic(t *testing.T) {
	projectName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProject_DS_basic(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_project.proj", "name", projectName),
					resource.TestCheckResourceAttr("data.lxd_project.proj", "description", "Terraform provider test project"),
					resource.TestCheckResourceAttr("data.lxd_project.proj", "config.user.key", "value"),
				),
			},
		},
	})
}

func testAccProject_DS_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_project" "proj" {
  name        = %[1]q
  description = "Terraform provider test project"

  config = {
    "user.key" = "value"
  }
}

data "lxd_project" "proj" {
  name = lxd_project.proj.name
}
  `, name)
}
