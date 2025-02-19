package image_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccImage_DS_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_DS_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_image.img", "name", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("data.lxd_image.img", "type", "container"),
					resource.TestCheckResourceAttr("data.lxd_image.img", "aliases.#", "2"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "architecture"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "fingerprint"),
				),
			},
		},
	})
}

func TestAccImage_DS_basicVM(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_DS_basicVM(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_image.img", "name", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("data.lxd_image.img", "type", "virtual-machine"),
					resource.TestCheckResourceAttr("data.lxd_image.img", "aliases.#", "2"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "architecture"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "fingerprint"),
				),
			},
		},
	})
}

func TestAccImage_DS_cached(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_DS_cached("custom-alias1", "custom-alias2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_image.img", "type", "container"),
					resource.TestCheckResourceAttr("data.lxd_image.img", "aliases.0", "custom-alias1"),
					resource.TestCheckResourceAttr("data.lxd_image.img", "aliases.1", "custom-alias2"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "architecture"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "fingerprint"),
					resource.TestCheckNoResourceAttr("data.lxd_image.img", "name"),
				),
			},
		},
	})
}

func TestAccImage_DS_project(t *testing.T) {
	projectName := acctest.GenerateName(2, "")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_DS_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_image.img", "type", "container"),
					resource.TestCheckResourceAttr("data.lxd_image.img", "project", projectName),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "architecture"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "fingerprint"),
					resource.TestCheckNoResourceAttr("data.lxd_image.img", "name"),
				),
			},
		},
	})
}

func testAccImage_DS_basic() string {
	return fmt.Sprintf(`
data "lxd_image" "img" {
  name   = %q
  remote = %q
}
	`, acctest.TestCachedImageSourceImage, acctest.TestCachedImageSourceRemote)
}

func testAccImage_DS_basicVM() string {
	return fmt.Sprintf(`
data "lxd_image" "img" {
  name   = %q
  type   = "virtual-machine"
  remote = %q
}
	`, acctest.TestCachedImageSourceImage, acctest.TestCachedImageSourceRemote)
}

func testAccImage_DS_cached(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img" {
  source_remote = %q
  source_image  = %q
  copy_aliases  = false
  aliases       = ["%s"]
}

data "lxd_image" "img" {
  fingerprint = lxd_cached_image.img.fingerprint
}
	`, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage, strings.Join(aliases, `","`))
}

func testAccImage_DS_project(project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "proj" {
  name = %q
}

resource "lxd_cached_image" "img" {
  source_remote = %q
  source_image  = %q
  project       = lxd_project.proj.name
}

data "lxd_image" "img" {
  fingerprint = lxd_cached_image.img.fingerprint
  project     = lxd_project.proj.name
}
	`, project, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage)
}
