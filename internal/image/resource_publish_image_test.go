package image_test

import (
	"fmt"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccPublishImage_basic(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "instance", instanceName),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "aliases.0", "test_basic"),
					resource.TestCheckResourceAttrSet("lxd_publish_image.pimg", "resource_id"),
				),
			},
		},
	})
}

func TestAccPublishImage_aliases(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	aliases := []string{"alias1", "alias2"}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_aliases(instanceName, aliases...),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "instance", instanceName),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "aliases.#", "2"),
					resource.TestCheckTypeSetElemAttr("lxd_publish_image.pimg", "aliases.*", aliases[0]),
					resource.TestCheckTypeSetElemAttr("lxd_publish_image.pimg", "aliases.*", aliases[1]),
					resource.TestCheckResourceAttrSet("lxd_publish_image.pimg", "resource_id"),
				),
			},
		},
	})
}

func TestAccPublishImage_properties(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	properties := map[string]string{"os": "Alpine", "version": "4"}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_properties(instanceName, properties),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "instance", instanceName),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "aliases.#", "0"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "properties.%", "2"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "properties.os", "Alpine"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "properties.version", "4"),
					resource.TestCheckResourceAttrSet("lxd_publish_image.pimg", "resource_id"),
				),
			},
		},
	})
}

func TestAccPublishImage_project(t *testing.T) {
	projectName := petname.Name()
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_project(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "instance", instanceName),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "aliases.#", "0"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "aliases.#", "0"),
					resource.TestCheckResourceAttr("lxd_publish_image.pimg", "project", projectName),
					resource.TestCheckResourceAttrSet("lxd_publish_image.pimg", "resource_id"),
				),
			},
		},
	})
}

func testAccPublishImage_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name            = "%s"
  image           = "%s"
  start_on_create = false
}

resource "lxd_publish_image" "pimg" {
  instance = lxd_instance.instance1.name
  aliases  = ["test_basic"]
}
	`, name, acctest.TestImage)
}

func testAccPublishImage_aliases(name string, aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name            = "%s"
  image           = "%s"
  start_on_create = false
}

resource "lxd_publish_image" "pimg" {
  instance = lxd_instance.instance1.name
  aliases  = ["%s"]
}
	`, name, acctest.TestImage, strings.Join(toStringSlice(aliases), "\",\""))
}

func testAccPublishImage_properties(name string, properties map[string]string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name            = "%s"
  image           = "%s"
  start_on_create = false
}

resource "lxd_publish_image" "pimg" {
  instance = lxd_instance.instance1.name
  properties = {
    %s
  }
}
	`, name, acctest.TestImage, strings.Join(formatProperties(properties), "\n"))
}

func testAccPublishImage_project(project, instance string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
  config = {
    "features.storage.volumes" = false
    "features.images"          = false
    "features.profiles"        = false
  }
}

resource "lxd_instance" "instance1" {
  name            = "%s"
  image           = "%s"
  project         = lxd_project.project1.name
  start_on_create = false
}

resource "lxd_publish_image" "pimg" {
  instance = lxd_instance.instance1.name
  project  = lxd_project.project1.name
}
	`, project, instance, acctest.TestImage)
}

func toStringSlice(slice []string) []string {
	new := make([]string, 0, len(slice))
	for _, v := range slice {
		new = append(new, v)
	}
	return new
}

func formatProperties(properties map[string]string) []string {
	r := make([]string, 0, len(properties))
	for k, v := range properties {
		r = append(r, fmt.Sprintf(`"%s":"%s"`, k, v))
	}
	return r
}
