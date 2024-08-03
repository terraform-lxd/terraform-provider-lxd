package image_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img1", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("incus_image.img1", "copied_aliases.#", "4"),
				),
			},
		},
	})
}

func TestAccImage_basicVM(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_basicVM(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img1vm", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1vm", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1vm", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("incus_image.img1vm", "type", "virtual-machine"),
					resource.TestCheckResourceAttr("incus_image.img1vm", "copied_aliases.#", "4"),
				),
			},
		},
	})
}

func TestAccImage_alias(t *testing.T) {
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img2", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img2", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("incus_image.img2", "aliases.#", "2"),
					resource.TestCheckTypeSetElemAttr("incus_image.img2", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("incus_image.img2", "aliases.*", alias2),
					resource.TestCheckResourceAttr("incus_image.img2", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_copiedAliases(t *testing.T) {
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_copiedAliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img3", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img3", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img3", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("incus_image.img3", "aliases.#", "3"),
					resource.TestCheckTypeSetElemAttr("incus_image.img3", "aliases.*", "alpine/edge"),
					resource.TestCheckTypeSetElemAttr("incus_image.img3", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("incus_image.img3", "aliases.*", alias2),
					resource.TestCheckResourceAttr("incus_image.img3", "copied_aliases.#", "4"),
				),
			},
		},
	})
}

func TestAccImage_aliasCollision(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_aliasCollision(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img4", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img4", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img4", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("incus_image.img4", "aliases.#", "1"),
					resource.TestCheckResourceAttr("incus_image.img4", "aliases.0", "alpine/edge/amd64"),
					resource.TestCheckResourceAttr("incus_image.img4", "copied_aliases.#", "4"),
				),
			},
		},
	})
}

func TestAccImage_aliasExists(t *testing.T) {
	alias := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_aliasExists1(alias),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.exists1", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.exists1", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.exists1", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("incus_image.exists1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("incus_image.exists1", "aliases.0", alias),
					resource.TestCheckResourceAttr("incus_image.exists1", "copied_aliases.#", "0"),
				),
			},
			{
				Config:      testAccImage_aliasExists2(alias),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Image alias %q already exists`, alias)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.exists1", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.exists1", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.exists1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("incus_image.exists1", "aliases.0", alias),
				),
			},
		},
	})
}

func TestAccImage_addRemoveAlias(t *testing.T) {
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_aliases(alias1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img2", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img2", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("incus_image.img2", "aliases.#", "1"),
					resource.TestCheckResourceAttr("incus_image.img2", "aliases.0", alias1),
					resource.TestCheckResourceAttr("incus_image.img2", "copied_aliases.#", "0"),
				),
			},
			{
				Config: testAccImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img2", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img2", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("incus_image.img2", "aliases.#", "2"),
					resource.TestCheckTypeSetElemAttr("incus_image.img2", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("incus_image.img2", "aliases.*", alias2),
					resource.TestCheckResourceAttr("incus_image.img2", "copied_aliases.#", "0"),
				),
			},
			{
				Config: testAccImage_aliases(alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img2", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img2", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("incus_image.img2", "aliases.#", "1"),
					resource.TestCheckResourceAttr("incus_image.img2", "aliases.0", alias2),
					resource.TestCheckResourceAttr("incus_image.img2", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_project(t *testing.T) {
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1", "project", projectName),
					resource.TestCheckNoResourceAttr("incus_image.img1", "aliases"),
					resource.TestCheckResourceAttr("incus_image.img1", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_instanceFromImageFingerprint(t *testing.T) {
	projectName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_instanceFromImageFingerprint(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.inst", "project", projectName),
				),
			},
		},
	})
}

func TestAccImage_architecture(t *testing.T) {
	projectName := petname.Name()
	architecture := "aarch64"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_architecture(projectName, architecture),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "source_remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1", "project", projectName),
					resource.TestCheckNoResourceAttr("incus_image.img1", "aliases"),
					resource.TestCheckResourceAttr("incus_image.img1", "copied_aliases.#", "0"),
					resource.TestCheckResourceAttr("incus_image.img1", "architecture", architecture),
				),
			},
		},
	})
}

func testAccImage_basic() string {
	return `
resource "incus_image" "img1" {
  source_remote = "images"
  source_image  = "alpine/edge"
  copy_aliases  = true
}
	`
}

func testAccImage_basicVM() string {
	return `
resource "incus_image" "img1vm" {
  source_remote = "images"
  source_image  = "alpine/edge"
  type          = "virtual-machine"
  copy_aliases  = true
}
	`
}

func testAccImage_aliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "incus_image" "img2" {
  source_remote = "images"
  source_image  = "alpine/edge"
  aliases       = ["%s"]
  copy_aliases  = false
}
	`, strings.Join(aliases, `","`))
}

func testAccImage_aliasExists1(alias string) string {
	return fmt.Sprintf(`
resource "incus_image" "exists1" {
  source_remote = "images"
  source_image  = "alpine/edge"
  aliases       = ["%s"]
  copy_aliases  = false
}
	`, alias)
}

func testAccImage_aliasExists2(alias string) string {
	return fmt.Sprintf(`
resource "incus_image" "exists1" {
  source_remote = "images"
  source_image  = "alpine/edge"
  aliases       = ["%s"]
  copy_aliases  = false
}

resource "incus_image" "exists2" {
  source_remote = "images"
  source_image  = "alpine/edge"
  aliases       = ["%s"]
  copy_aliases  = false
}
	`, alias, alias)
}

func testAccImage_copiedAliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "incus_image" "img3" {
  source_remote = "images"
  source_image  = "alpine/edge"
  aliases       = ["alpine/edge","%s"]
  copy_aliases  = true
}
	`, strings.Join(aliases, `","`))
}

func testAccImage_aliasCollision() string {
	return `
resource "incus_image" "img4" {
  source_remote = "images"
  source_image  = "alpine/edge"
  aliases       = ["alpine/edge/amd64"]
  copy_aliases  = true
}
	`
}

func testAccImage_project(project string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
}
resource "incus_image" "img1" {
  source_remote = "images"
  source_image  = "alpine/edge"
  project       = incus_project.project1.name
}
	`, project)
}

func testAccImage_instanceFromImageFingerprint(project string, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
}

resource "incus_image" "img1" {
  source_remote = "images"
  source_image  = "alpine/edge"
  project       = incus_project.project1.name
}

resource "incus_instance" "inst" {
  name    = "%s"
  project = incus_project.project1.name
  image   = incus_image.img1.fingerprint
  running = false

  device {
    name = "root"
    type = "disk"
    properties = {
	  pool = "default"
	  path = "/"
    }
  }
}
	`, project, instanceName)
}

func testAccImage_architecture(project string, architecture string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
}
resource "incus_image" "img1" {
  source_remote = "images"
  source_image  = "alpine/edge"
  project       = incus_project.project1.name
  architecture  = "%s"
}
	`, project, architecture)
}
