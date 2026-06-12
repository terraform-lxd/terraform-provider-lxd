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
				Config: acctest.Provider() + testAccImage_DS_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_image.img", "image", acctest.TestCachedImage),
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
				Config: acctest.Provider() + testAccImage_DS_basicVM(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_image.img", "image", acctest.TestCachedImage),
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
				Config: acctest.Provider() + testAccImage_DS_cached("custom-alias1", "custom-alias2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_image.img", "type", "container"),
					resource.TestCheckResourceAttr("data.lxd_image.img", "aliases.0", "custom-alias1"),
					resource.TestCheckResourceAttr("data.lxd_image.img", "aliases.1", "custom-alias2"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "architecture"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "fingerprint"),
				),
			},
		},
	})
}

func TestAccImage_DS_fingerprintWithArchitecture(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_DS_fingerprintWithArchitecture(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_image.img", "type", "container"),
					resource.TestCheckResourceAttrPair("data.lxd_image.img", "fingerprint", "lxd_image.img", "fingerprint"),
					resource.TestCheckResourceAttrPair("data.lxd_image.img", "architecture", "lxd_image.img", "source_image.architecture"),
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
				Config: acctest.Provider() + testAccImage_DS_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_image.img", "type", "container"),
					resource.TestCheckResourceAttr("data.lxd_image.img", "project", projectName),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "architecture"),
					resource.TestCheckResourceAttrSet("data.lxd_image.img", "fingerprint"),
				),
			},
		},
	})
}

func testAccImage_DS_basic() string {
	return fmt.Sprintf(`
data "lxd_image" "img" {
  image = %q
}
	`, acctest.TestCachedImage)
}

func testAccImage_DS_basicVM() string {
	return fmt.Sprintf(`
data "lxd_image" "img" {
  image = %q
  type  = "virtual-machine"
}
	`, acctest.TestCachedImage)
}

func testAccImage_DS_cached(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_image" "img" {
  aliases = ["%s"]

  source_image = {
    image        = %q
    copy_aliases = false
  }
}

data "lxd_image" "img" {
  image = lxd_image.img.fingerprint
}
	`, strings.Join(aliases, `","`), acctest.TestCachedImage)
}

func testAccImage_DS_fingerprintWithArchitecture() string {
	return fmt.Sprintf(`
resource "lxd_image" "img" {
  source_image = {
    image = %q
  }
}

data "lxd_image" "img" {
  image        = lxd_image.img.fingerprint
  architecture = lxd_image.img.source_image.architecture
}
	`, acctest.TestCachedImage)
}

func testAccImage_DS_project(project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "proj" {
  name = %q
}

resource "lxd_image" "img" {
  project = lxd_project.proj.name

  source_image = {
    image = %q
  }
}

data "lxd_image" "img" {
  image   = lxd_image.img.fingerprint
  project = lxd_project.proj.name
}
	`, project, acctest.TestCachedImage)
}
