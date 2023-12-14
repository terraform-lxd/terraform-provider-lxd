package incus

import (
	"fmt"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/lxc/incus/shared/api"
)

func TestAccPublishImage_basic(t *testing.T) {
	var instance api.Instance
	var img api.Image
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceState(t, "incus_instance.instance1", &instance, api.Stopped),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					testAccCachedImageExists(t, "incus_publish_image.test_basic", &img),
				),
			},
		},
	})
}

func TestAccPublishImage_aliases(t *testing.T) {
	var instance api.Instance
	var img api.Image
	instanceName := petname.Generate(2, "-")

	aliases := []interface{}{"alias1", "alias2"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_aliases(instanceName, aliases),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceState(t, "incus_instance.instance1", &instance, api.Stopped),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					testAccCachedImageExists(t, "incus_publish_image.test_basic", &img),
					testAccPublishImageHasAliases(&img, aliases),
				),
			},
		},
	})
}

func TestAccPublishImage_properties(t *testing.T) {
	var instance api.Instance
	var img api.Image
	instanceName := petname.Generate(2, "-")

	properties := map[string]string{"os": "Alpine", "version": "4"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_properties(instanceName, properties),
				Check: resource.ComposeTestCheckFunc(
					testAccInstanceState(t, "incus_instance.instance1", &instance, api.Stopped),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					testAccCachedImageExists(t, "incus_publish_image.test_basic", &img),
					testAccPublishImageHasProperties(&img, properties),
				),
			},
		},
	})
}

func TestAccPublishImage_project(t *testing.T) {
	var img api.Image
	var instance api.Instance
	var project api.Project
	projectName := petname.Name()
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_project(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "incus_project.project1", &project),
					testAccInstanceRunningInProject(t, "incus_instance.instance1", &instance, projectName),
					testAccPublishImageExistsInProject(t, "incus_publish_image.test_basic", &img, projectName),
				),
			},
		},
	})
}

func testAccPublishImage_basic(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]

  start_on_create = false
}

resource "incus_publish_image" "test_basic" {
  depends_on = [ incus_instance.instance1 ]

  container = "%s"
  aliases = [ "test_basic" ]
}
	`, name, name)
}

func testAccPublishImage_aliases(name string, aliases []interface{}) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]

  start_on_create = false
}

resource "incus_publish_image" "test_basic" {
  depends_on = [ incus_instance.instance1 ]

  container = "%s"
  aliases = [ "%s" ]
}
	`, name, name, strings.Join(toStringSlice(aliases), "\",\""))
}

func testAccPublishImage_properties(name string, properties map[string]string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]

  start_on_create = false
}

resource "incus_publish_image" "test_basic" {
  depends_on = [ incus_instance.instance1 ]

  container = "%s"
  properties = {
  	%s
  }
}
	`, name, name, strings.Join(formatProperties(properties), "\n"))
}

func testAccPublishImage_project(project, instance string) string {
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
  name = "%s"
  image = "images:alpine/3.18/amd64"
  profiles = ["default"]
  project = incus_project.project1.name
  start_on_create = false
}

resource "incus_publish_image" "test_basic" {
  depends_on = [ incus_instance.instance1 ]
  project = incus_project.project1.name
  container = "%s"
}
	`, project, instance, instance)
}

func testAccPublishImageExists(t *testing.T, n string, image *api.Image) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found in state: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		id := newPublishImageIDFromResourceID(rs.Primary.ID)
		client, err := testAccProvider.Meta().(*incusProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		img, _, err := client.GetImage(id.fingerprint)
		if err != nil {
			return err
		}

		if img != nil {
			*image = *img
			return nil
		}

		return fmt.Errorf("Image not found: %s", rs.Primary.ID)
	}
}

func testAccPublishImageExistsInProject(t *testing.T, n string, image *api.Image, project string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found in state: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		id := newPublishImageIDFromResourceID(rs.Primary.ID)
		client, err := testAccProvider.Meta().(*incusProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		client = client.UseProject(project)
		img, _, err := client.GetImage(id.fingerprint)
		if err != nil {
			return err
		}

		if img != nil {
			*image = *img
			return nil
		}

		return fmt.Errorf("Image not found: %s", rs.Primary.ID)
	}
}

func testAccPublishImageHasAliases(img *api.Image, aliases []interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if img.Aliases == nil || len(img.Aliases) == 0 {
			return fmt.Errorf("No aliases")
		}

		aliasSet := schema.NewSet(schema.HashString, aliases)
		found := 0

		for _, a := range img.Aliases {
			if aliasSet.Contains(a.Name) {
				found++
			}
		}

		if found != len(aliases) {
			return fmt.Errorf("The aliases doesn't match")
		}

		return nil
	}
}

func testAccPublishImageHasProperties(img *api.Image, properties map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if img.Properties == nil || len(img.Properties) == 0 {
			return fmt.Errorf("No properties")
		}

		for k, v := range img.Properties {
			if vv, ok := properties[k]; ok && v == vv {
				continue
			} else {
				return fmt.Errorf("Property %s does not match", k)
			}
		}

		return nil
	}
}

func toStringSlice(v []interface{}) []string {
	r := make([]string, len(v))
	for i, v := range v {
		r[i] = v.(string)
	}
	return r
}

func formatProperties(properties map[string]string) []string {
	r := make([]string, 0, len(properties))
	for k, v := range properties {
		r = append(r, fmt.Sprintf(`"%s":"%s"`, k, v))
	}
	return r
}
