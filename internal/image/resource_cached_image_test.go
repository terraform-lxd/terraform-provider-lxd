package image_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccCachedImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "copied_aliases.#", "10"),
				),
			},
		},
	})
}

func TestAccCachedImage_basicVM(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_basicVM(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "copied_aliases.#", "10"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "type", "virtual-machine"),
				),
			},
		},
	})
}

func TestAccCachedImage_alias(t *testing.T) {
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "aliases.#", "2"),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img2", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img2", "aliases.*", alias2),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccCachedImage_copiedAliases(t *testing.T) {
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_copiedAliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "aliases.#", "3"),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img3", "aliases.*", "22.04"),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img3", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img3", "aliases.*", alias2),
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "copied_aliases.#", "10"),
				),
			},
		},
	})
}

func TestAccCachedImage_aliasCollision(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliasCollision(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "aliases.0", "22.04/amd64"),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "copied_aliases.#", "10"),
				),
			},
		},
	})
}

func TestAccCachedImage_aliasExists(t *testing.T) {
	alias := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliasExists1(alias),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "aliases.0", alias),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "copied_aliases.#", "0"),
				),
			},
			{
				Config:      testAccCachedImage_aliasExists2(alias),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Image alias %q already exists`, alias)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "aliases.0", alias),
				),
			},
		},
	})
}

func TestAccCachedImage_addRemoveAlias(t *testing.T) {
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliases(alias1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "aliases.0", alias1),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copied_aliases.#", "0"),
				),
			},
			{
				Config: testAccCachedImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "aliases.#", "2"),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img2", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img2", "aliases.*", alias2),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copied_aliases.#", "0"),
				),
			},
			{
				Config: testAccCachedImage_aliases(alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "aliases.0", alias2),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccCachedImage_project(t *testing.T) {
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_remote", "ubuntu-minimal-daily"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_image", "22.04"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "project", projectName),
					resource.TestCheckNoResourceAttr("lxd_cached_image.img1", "aliases"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func testAccCachedImage_basic() string {
	return `
resource "lxd_cached_image" "img1" {
  source_remote = "ubuntu-minimal-daily"
  source_image  = "22.04"
  copy_aliases  = true
}
	`
}

func testAccCachedImage_basicVM() string {
	return `
resource "lxd_cached_image" "img1vm" {
  source_remote = "ubuntu-minimal-daily"
  source_image  = "22.04"
  type          = "virtual-machine"
  copy_aliases  = true
}
	`
}

func testAccCachedImage_aliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img2" {
  source_remote = "ubuntu-minimal-daily"
  source_image  = "22.04"
  aliases       = ["%s"]
  copy_aliases  = false
}
	`, strings.Join(aliases, `","`))
}

func testAccCachedImage_aliasExists1(alias string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "exists1" {
  source_remote = "ubuntu-minimal-daily"
  source_image  = "22.04"
  aliases       = ["%s"]
  copy_aliases  = false
}
	`, alias)
}

func testAccCachedImage_aliasExists2(alias string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "exists1" {
  source_remote = "ubuntu-minimal-daily"
  source_image  = "22.04"
  aliases       = ["%s"]
  copy_aliases  = false
}

resource "lxd_cached_image" "exists2" {
  source_remote = "ubuntu-minimal-daily"
  source_image  = "22.04"
  aliases       = ["%s"]
  copy_aliases  = false
}
	`, alias, alias)
}

func testAccCachedImage_copiedAliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img3" {
  source_remote = "ubuntu-minimal-daily"
  source_image  = "22.04"
  aliases       = ["22.04","%s"]
  copy_aliases  = true
}
	`, strings.Join(aliases, `","`))
}

func testAccCachedImage_aliasCollision() string {
	return `
resource "lxd_cached_image" "img4" {
  source_remote = "ubuntu-minimal-daily"
  source_image  = "22.04"
  aliases       = ["22.04/amd64"]
  copy_aliases  = true
}
	`
}

func testAccCachedImage_project(project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
}
resource "lxd_cached_image" "img1" {
  source_remote = "ubuntu-minimal-daily"
  source_image  = "22.04"
  project       = lxd_project.project1.name
}
	`, project)
}
