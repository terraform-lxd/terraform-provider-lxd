package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/lxc/lxd/client"
)

func TestAccContainerFileContentConfig(t *testing.T) {
	var file lxd.ContainerFileResponse

	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerFileContentConfig(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerFileExists(t, "lxd_container_file.file1", &file),
					resource.TestCheckResourceAttr("lxd_container_file.file1", "create_directories", "true"),
				),
			},
		},
	})
}

func TestAccContainerFileSourceConfig(t *testing.T) {
	var file lxd.ContainerFileResponse

	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainerFileSourceConfig(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerFileExists(t, "lxd_container_file.file1", &file),
					resource.TestCheckResourceAttr("lxd_container_file.file1", "create_directories", "true"),
				),
			},
		},
	})
}

func testAccContainerFileExists(t *testing.T, n string, file *lxd.ContainerFileResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := testAccProvider.Meta().(*lxdProvider)
		v, targetFile := newFileIDFromResourceID(rs.Primary.ID)
		remote, containerName, err := p.LXDConfig.ParseRemote(v)

		client, err := p.GetContainerServer(remote)
		if err != nil {
			return err
		}

		_, f, err := client.GetContainerFile(containerName, targetFile)
		if err != nil {
			return err
		}

		if f != nil {
			*file = *f
			return nil
		}

		return fmt.Errorf("Container file not found: %s", rs.Primary.ID)
	}
}

func testAccContainerFileContentConfig(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}

resource "lxd_container_file" "file1" {
  container_name = "${lxd_container.container1.name}"
  target_file = "/foo/bar.txt"
  content = "Hello, World!\n"
  create_directories = true
}
	`, name)
}

func testAccContainerFileSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.9/amd64"
  profiles = ["default"]
}

resource "lxd_container_file" "file1" {
  container_name = "${lxd_container.container1.name}"
  target_file = "/foo/bar.txt"
	source = "test-fixtures/test-file.txt"
  create_directories = true
}
	`, name)
}
