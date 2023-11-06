package lxd

import (
	"fmt"
	"testing"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccContainerFile_content(t *testing.T) {
	var file lxd.ContainerFileResponse

	containerName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainerFile_content(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerFileExists(t, "lxd_container_file.file1", &file),
					resource.TestCheckResourceAttr("lxd_container_file.file1", "create_directories", "true"),
				),
			},
		},
	})
}

func TestAccContainerFile_source(t *testing.T) {
	var file lxd.ContainerFileResponse

	containerName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainerFile_source(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerFileExists(t, "lxd_container_file.file1", &file),
					resource.TestCheckResourceAttr("lxd_container_file.file1", "create_directories", "true"),
				),
			},
		},
	})
}

func TestAccContainerFile_project(t *testing.T) {
	var file lxd.ContainerFileResponse
	var project api.Project
	var container api.Container
	projectName := petname.Name()
	containerName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainerFile_project(projectName, containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccContainerRunningInProject(t, "lxd_container.container1", &container, projectName),
					testAccContainerFileExistsInProject(t, "lxd_container_file.file1", &file, projectName),
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

		client, err := p.GetInstanceServer(remote)
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

func testAccContainerFileExistsInProject(t *testing.T, n string, file *lxd.ContainerFileResponse, project string) resource.TestCheckFunc {
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

		client, err := p.GetInstanceServer(remote)
		if err != nil {
			return err
		}
		client = client.UseProject(project)

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

func testAccContainerFile_content(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccContainerFile_source(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
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

func testAccContainerFile_project(project, container string) string {
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

resource "lxd_container" "container1" {
  name      = "%s"
  image     = "images:alpine/3.18/amd64"
  project   = lxd_project.project1.name
}

resource "lxd_container_file" "file1" {
  container_name     = lxd_container.container1.name
  target_file        = "/foo/bar.txt"
  source   	     = "test-fixtures/test-file.txt"
  create_directories = true
  project   	     = lxd_project.project1.name
}
	`, project, container)
}
