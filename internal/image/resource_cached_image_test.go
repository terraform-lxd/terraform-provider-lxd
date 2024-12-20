package image_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

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
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_image", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "copied_aliases.#", "2"),
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
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "source_image", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "copied_aliases.#", "2"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1vm", "type", "virtual-machine"),
				),
			},
		},
	})
}

func TestAccCachedImage_alias(t *testing.T) {
	alias1 := acctest.GenerateName(2, "-")
	alias2 := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_image", acctest.TestCachedImageSourceImage),
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
	alias1 := acctest.GenerateName(2, "-")
	alias2 := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_copiedAliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "source_image", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "aliases.#", "3"),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img3", "aliases.*", acctest.TestCachedImageSourceImage),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img3", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("lxd_cached_image.img3", "aliases.*", alias2),
					resource.TestCheckResourceAttr("lxd_cached_image.img3", "copied_aliases.#", "2"),
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
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "source_image", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "aliases.0", acctest.TestCachedImageSourceImage+"/amd64"),
					resource.TestCheckResourceAttr("lxd_cached_image.img4", "copied_aliases.#", "2"),
				),
			},
		},
	})
}

func TestAccCachedImage_aliasExists(t *testing.T) {
	alias := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliasExists1(alias),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "source_image", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "aliases.0", alias),
					resource.TestCheckResourceAttr("lxd_cached_image.exists1", "copied_aliases.#", "0"),
				),
			},
			{
				Config:      testAccCachedImage_aliasExists2(alias),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Alias already exists: %s`, alias)),
			},
		},
	})
}

func TestAccCachedImage_addRemoveAlias(t *testing.T) {
	alias1 := acctest.GenerateName(2, "-")
	alias2 := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliases(alias1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_image", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "aliases.0", alias1),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "copied_aliases.#", "0"),
				),
			},
			{
				Config: testAccCachedImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_image", acctest.TestCachedImageSourceImage),
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
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.img2", "source_image", acctest.TestCachedImageSourceImage),
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
	projectName := acctest.GenerateName(2, "")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_image", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "aliases.#", "0"),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccCachedImage_instanceFromImageFingerprint(t *testing.T) {
	projectName := acctest.GenerateName(2, "")
	instanceName := acctest.GenerateName(2, "")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t) // The remote "local" does not point to clustered LXD.
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create an instance from the cached image and do not set instance
				// remote. Test will succeed only if the image is searched in the
				// remote and project where instance is created.
				Config: testAccCachedImage_instanceFromImageFingerprint(projectName, instanceName, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_image", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.inst", "project", projectName),
				),
			},
			{
				// Create an instance from the cached image and set instance's remote.
				// Test will succeed only if the image is searched in the remote and
				// project where instance is created.
				Config: testAccCachedImage_instanceFromImageFingerprint(projectName, instanceName, "local"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_image", acctest.TestCachedImageSourceImage),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "source_remote", acctest.TestCachedImageSourceRemote),
					resource.TestCheckResourceAttr("lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.inst", "project", projectName),
				),
			},
		},
	})
}

func testAccCachedImage_basic() string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img1" {
  source_remote = "%s"
  source_image  = "%s"
  copy_aliases  = true
}
	`, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage)
}

func testAccCachedImage_basicVM() string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img1vm" {
  source_remote = "%s"
  source_image  = "%s"
  type          = "virtual-machine"
  copy_aliases  = true
}
	`, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage)
}

func testAccCachedImage_aliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img2" {
  source_remote = "%s"
  source_image  = "%s"
  aliases       = ["%s"]
  copy_aliases  = false
}
	`, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage, strings.Join(aliases, `","`))
}

func testAccCachedImage_aliasExists1(alias string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "exists1" {
  source_remote = "%s"
  source_image  = "%s"
  aliases       = ["%s"]
  copy_aliases  = false
}
	`, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage, alias)
}

func testAccCachedImage_aliasExists2(alias string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "exists1" {
  source_remote = "%s"
  source_image  = "%s"
  aliases       = ["%s"]
  copy_aliases  = false
}

resource "lxd_cached_image" "exists2" {
  source_remote = "%s"
  source_image  = "%s"
  aliases       = ["%s"]
  copy_aliases  = false
}
	`, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage, alias, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage, alias)
}

func testAccCachedImage_copiedAliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img3" {
  source_remote = "%s"
  source_image  = "%s"
  aliases       = ["%s","%s"]
  copy_aliases  = true
}
	`, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage, acctest.TestCachedImageSourceImage, strings.Join(aliases, `","`))
}

func testAccCachedImage_aliasCollision() string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img4" {
  source_remote = "%s"
  source_image  = "%s"
  aliases       = ["%s/amd64"]
  copy_aliases  = true
}
	`, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage, acctest.TestCachedImageSourceImage)
}

func testAccCachedImage_project(project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
}

resource "lxd_cached_image" "img1" {
  source_remote = "%s"
  source_image  = "%s"
  project       = lxd_project.project1.name
}
	`, project, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage)
}

func testAccCachedImage_instanceFromImageFingerprint(project string, instanceName string, instanceRemote string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"

  config = {
    "features.images"   = true
    "features.profiles" = false
  }
}

resource "lxd_cached_image" "img1" {
  source_remote = "%s"
  source_image  = "%s"
  project       = lxd_project.project1.name
}

resource "lxd_instance" "inst" {
    name    = "%s"
    image   = lxd_cached_image.img1.fingerprint
    remote  = "%s"
    project = lxd_project.project1.name
    running = false
}
	`, project, acctest.TestCachedImageSourceRemote, acctest.TestCachedImageSourceImage, instanceName, instanceRemote)
}
