package image_test

import (
	"fmt"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccImagePublish_basic(t *testing.T) {
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImagePublish_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "instance", instanceName),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "aliases.#", "1"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "aliases.0", "test_basic"),
					resource.TestCheckResourceAttrSet("incus_image_publish.pimg", "resource_id"),
				),
			},
		},
	})
}

func TestAccImagePublish_aliases(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	aliases := []string{"alias1", "alias2"}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImagePublish_aliases(instanceName, aliases...),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "instance", instanceName),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "aliases.#", "2"),
					resource.TestCheckTypeSetElemAttr("incus_image_publish.pimg", "aliases.*", aliases[0]),
					resource.TestCheckTypeSetElemAttr("incus_image_publish.pimg", "aliases.*", aliases[1]),
					resource.TestCheckResourceAttrSet("incus_image_publish.pimg", "resource_id"),
				),
			},
		},
	})
}

func TestAccImagePublish_properties(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	properties := map[string]string{"os": "Alpine", "version": "4"}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImagePublish_properties(instanceName, properties),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "instance", instanceName),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "aliases.#", "0"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "properties.%", "2"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "properties.os", "Alpine"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "properties.version", "4"),
					resource.TestCheckResourceAttrSet("incus_image_publish.pimg", "resource_id"),
				),
			},
		},
	})
}

func TestAccImagePublish_project(t *testing.T) {
	projectName := petname.Name()
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImagePublish_project(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "instance", instanceName),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "aliases.#", "0"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "aliases.#", "0"),
					resource.TestCheckResourceAttr("incus_image_publish.pimg", "project", projectName),
					resource.TestCheckResourceAttrSet("incus_image_publish.pimg", "resource_id"),
				),
			},
		},
	})
}

func testAccImagePublish_basic(name string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  running = false
}

resource "incus_image_publish" "pimg" {
  instance = incus_instance.instance1.name
  aliases  = ["test_basic"]
}
	`, name, acctest.TestImage)
}

func testAccImagePublish_aliases(name string, aliases ...string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  running = false
}

resource "incus_image_publish" "pimg" {
  instance = incus_instance.instance1.name
  aliases  = ["%s"]
}
	`, name, acctest.TestImage, strings.Join(toStringSlice(aliases), "\",\""))
}

func testAccImagePublish_properties(name string, properties map[string]string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  running = false
}

resource "incus_image_publish" "pimg" {
  instance = incus_instance.instance1.name
  properties = {
    %s
  }
}
	`, name, acctest.TestImage, strings.Join(formatProperties(properties), "\n"))
}

func testAccImagePublish_project(project, instance string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
  config = {
    "features.storage.volumes" = false
    "features.images"          = false
    "features.profiles"        = false
  }
}

resource "incus_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  project = incus_project.project1.name
  running = false
}

resource "incus_image_publish" "pimg" {
  instance = incus_instance.instance1.name
  project  = incus_project.project1.name
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
