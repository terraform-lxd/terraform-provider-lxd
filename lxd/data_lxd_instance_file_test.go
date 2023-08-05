package lxd

import (
	"fmt"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccInstanceFileDataSource(t *testing.T) {
	content := "Welcome to Alpine Linux 3.16\nKernel \\r on an \\m (\\l)\n\n"

	instanceName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataInstanceFile(instanceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_instance_file.file1", "content", content),
				),
			},
		},
	})
}

func testAccDataInstanceFile(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.16/amd64"
  profiles = ["default"]
}

data "lxd_instance_file" "file1" {
	instance_name = "${lxd_instance.instance1.name}"
	target_file = "/etc/issue"
	timeout = 30
}
	`, name)
}
