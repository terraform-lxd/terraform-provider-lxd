package incus

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
)

func TestAccInstanceFile_content(t *testing.T) {
	var file incus.InstanceFileResponse

	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceFile_content(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceFileExists(t, "incus_instance_file.file1", &file),
					resource.TestCheckResourceAttr("incus_instance_file.file1", "create_directories", "true"),
				),
			},
		},
	})
}

func TestAccInstanceFile_source(t *testing.T) {
	var file incus.InstanceFileResponse

	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceFile_source(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceFileExists(t, "incus_instance_file.file1", &file),
					resource.TestCheckResourceAttr("incus_instance_file.file1", "create_directories", "true"),
				),
			},
		},
	})
}

func TestAccInstanceFile_project(t *testing.T) {
	var file incus.InstanceFileResponse
	var project api.Project
	var instance api.Instance
	projectName := petname.Name()
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceFile_project(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "incus_project.project1", &project),
					testAccInstanceRunningInProject(t, "incus_instance.instance1", &instance, projectName),
					testAccInstanceFileExistsInProject(t, "incus_instance_file.file1", &file, projectName),
				),
			},
		},
	})
}

func testAccInstanceFileExists(t *testing.T, n string, file *incus.InstanceFileResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := testAccProvider.Meta().(*incusProvider)
		v, targetFile := newFileIDFromResourceID(rs.Primary.ID)
		remote, instanceName, err := p.IncusConfig.ParseRemote(v)

		client, err := p.GetInstanceServer(remote)
		if err != nil {
			return err
		}

		_, f, err := client.GetInstanceFile(instanceName, targetFile)
		if err != nil {
			return err
		}

		if f != nil {
			*file = *f
			return nil
		}

		return fmt.Errorf("Instance file not found: %s", rs.Primary.ID)
	}
}

func testAccInstanceFileExistsInProject(t *testing.T, n string, file *incus.InstanceFileResponse, project string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := testAccProvider.Meta().(*incusProvider)
		v, targetFile := newFileIDFromResourceID(rs.Primary.ID)
		remote, instanceName, err := p.IncusConfig.ParseRemote(v)

		client, err := p.GetInstanceServer(remote)
		if err != nil {
			return err
		}
		client = client.UseProject(project)

		_, f, err := client.GetInstanceFile(instanceName, targetFile)
		if err != nil {
			return err
		}

		if f != nil {
			*file = *f
			return nil
		}

		return fmt.Errorf("Instance file not found: %s", rs.Primary.ID)
	}
}

func testAccInstanceFile_content(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
}

resource "incus_instance_file" "file1" {
  instance_name = "${incus_instance.instance1.name}"
  target_file = "/foo/bar.txt"
  content = "Hello, World!\n"
  create_directories = true
}
	`, name)
}

func testAccInstanceFile_source(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
}

resource "incus_instance_file" "file1" {
  instance_name = "${incus_instance.instance1.name}"
  target_file = "/foo/bar.txt"
  source = "test-fixtures/test-file.txt"
  create_directories = true
}
	`, name)
}

func testAccInstanceFile_project(project, instance string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.profiles" = false
	"features.storage.buckets" = false
  }
}

resource "incus_instance" "instance1" {
  name      = "%s"
  image     = "images:alpine/3.18/amd64"
  project   = incus_project.project1.name
}

resource "incus_instance_file" "file1" {
  instance_name     = incus_instance.instance1.name
  target_file        = "/foo/bar.txt"
  source   	     = "test-fixtures/test-file.txt"
  create_directories = true
  project   	     = incus_project.project1.name
}
	`, project, instance)
}
