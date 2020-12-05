package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/lxc/lxd/shared/api"
)

func TestAccPublishImage_basic(t *testing.T) {
	var container api.Container
	var img api.Image
	containerName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_basic(containerName),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerState(t, "lxd_container.container1", &container, api.Stopped),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccCachedImageExists(t, "lxd_publish_image.test_basic", &img),
				),
			},
		},
	})
}

func TestAccPublishImage_aliases(t *testing.T) {
	var container api.Container
	var img api.Image
	containerName := strings.ToLower(petname.Generate(2, "-"))

	aliases := []interface{}{"alias1", "alias2"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_aliases(containerName, aliases),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerState(t, "lxd_container.container1", &container, api.Stopped),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccCachedImageExists(t, "lxd_publish_image.test_basic", &img),
					testAccPublishImageHasAliases(&img, aliases),
				),
			},
		},
	})
}

func TestAccPublishImage_properties(t *testing.T) {
	var container api.Container
	var img api.Image
	containerName := strings.ToLower(petname.Generate(2, "-"))

	properties := map[string]string{"os": "Alpine", "version": "4"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPublishImage_properties(containerName, properties),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerState(t, "lxd_container.container1", &container, api.Stopped),
					resource.TestCheckResourceAttr("lxd_container.container1", "name", containerName),
					testAccCachedImageExists(t, "lxd_publish_image.test_basic", &img),
					testAccPublishImageHasProperties(&img, properties),
				),
			},
		},
	})
}

func testAccPublishImage_basic(name string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.12/amd64"
  profiles = ["default"]

  start_container = false
}

resource "lxd_publish_image" "test_basic" {
  depends_on = [ lxd_container.container1 ]

  container = "%s"
  aliases = [ "test_basic" ]
}
	`, name, name)
}

func testAccPublishImage_aliases(name string, aliases []interface{}) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.12/amd64"
  profiles = ["default"]

  start_container = false
}

resource "lxd_publish_image" "test_basic" {
  depends_on = [ lxd_container.container1 ]

  container = "%s"
  aliases = [ "%s" ]
}
	`, name, name, strings.Join(toStringSlice(aliases), "\",\""))
}

func testAccPublishImage_properties(name string, properties map[string]string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.12/amd64"
  profiles = ["default"]

  start_container = false
}

resource "lxd_publish_image" "test_basic" {
  depends_on = [ lxd_container.container1 ]

  container = "%s"
  properties = {
  	%s
  }
}
	`, name, name, strings.Join(formatProperties(properties), "\n"))
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
		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
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
