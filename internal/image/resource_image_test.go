package image_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

func TestAccImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img1", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img1", "source_image.copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_image.img1", "copied_aliases.#", "2"),
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
				Config: acctest.Provider() + testAccImage_basicVM(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img1vm", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img1vm", "source_image.copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_image.img1vm", "source_image.type", "virtual-machine"),
					resource.TestCheckResourceAttr("lxd_image.img1vm", "copied_aliases.#", "2"),
				),
			},
		},
	})
}

func TestAccImage_alias(t *testing.T) {
	alias1 := acctest.GenerateName(2, "-")
	alias2 := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img2", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img2", "source_image.copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_image.img2", "aliases.#", "2"),
					resource.TestCheckTypeSetElemAttr("lxd_image.img2", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("lxd_image.img2", "aliases.*", alias2),
					resource.TestCheckResourceAttr("lxd_image.img2", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_copiedAliases(t *testing.T) {
	alias1 := acctest.GenerateName(2, "-")
	alias2 := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_copiedAliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img3", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img3", "source_image.copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_image.img3", "aliases.#", "3"),
					resource.TestCheckTypeSetElemAttr("lxd_image.img3", "aliases.*", acctest.TestCachedImageSourceImage),
					resource.TestCheckTypeSetElemAttr("lxd_image.img3", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("lxd_image.img3", "aliases.*", alias2),
					resource.TestCheckResourceAttr("lxd_image.img3", "copied_aliases.#", "2"),
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
				Config: acctest.Provider() + testAccImage_aliasCollision(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img4", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img4", "source_image.copy_aliases", "true"),
					resource.TestCheckResourceAttr("lxd_image.img4", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_image.img4", "aliases.0", acctest.TestCachedImageSourceImage+"/amd64"),
					resource.TestCheckResourceAttr("lxd_image.img4", "copied_aliases.#", "2"),
				),
			},
		},
	})
}

func TestAccImage_aliasExists(t *testing.T) {
	alias := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_aliasExists1(alias),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.exists1", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.exists1", "source_image.copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_image.exists1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_image.exists1", "aliases.0", alias),
					resource.TestCheckResourceAttr("lxd_image.exists1", "copied_aliases.#", "0"),
				),
			},
			{
				Config:      acctest.Provider() + testAccImage_aliasExists2(alias),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Image alias %q already exists`, alias)),
			},
		},
	})
}

func TestAccImage_addRemoveAlias(t *testing.T) {
	alias1 := acctest.GenerateName(2, "-")
	alias2 := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_aliases(alias1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img2", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img2", "source_image.copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_image.img2", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_image.img2", "aliases.0", alias1),
					resource.TestCheckResourceAttr("lxd_image.img2", "copied_aliases.#", "0"),
				),
			},
			{
				Config: acctest.Provider() + testAccImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img2", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img2", "source_image.copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_image.img2", "aliases.#", "2"),
					resource.TestCheckTypeSetElemAttr("lxd_image.img2", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("lxd_image.img2", "aliases.*", alias2),
					resource.TestCheckResourceAttr("lxd_image.img2", "copied_aliases.#", "0"),
				),
			},
			{
				Config: acctest.Provider() + testAccImage_aliases(alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img2", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img2", "source_image.copy_aliases", "false"),
					resource.TestCheckResourceAttr("lxd_image.img2", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_image.img2", "aliases.0", alias2),
					resource.TestCheckResourceAttr("lxd_image.img2", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_project(t *testing.T) {
	projectName := acctest.GenerateName(2, "")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_image.img1", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.#", "0"),
					resource.TestCheckResourceAttr("lxd_image.img1", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_instanceFromImageFingerprint(t *testing.T) {
	projectName := acctest.GenerateName(2, "")
	instanceName := acctest.GenerateName(2, "")

	provider := acctest.ProviderWithRemotes(map[string]provider_config.LxdRemote{
		"local": {
			Address: "unix://",
		},
	})

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
				Config: provider + testAccImage_instanceFromImageFingerprint(projectName, instanceName, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_image.img1", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.inst", "project", projectName),
				),
			},
			{
				// Create an instance from the cached image and set instance's remote.
				// Test will succeed only if the image is searched in the remote and
				// project where instance is created.
				Config: provider + testAccImage_instanceFromImageFingerprint(projectName, instanceName, "local"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_image.img1", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.inst", "project", projectName),
				),
			},
		},
	})
}

func TestAccImage_architecture(t *testing.T) {
	projectName := acctest.GenerateName(2, "")
	architecture := "aarch64"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_architecture(projectName, architecture),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_image.img1", "source_image.image", acctest.TestCachedImage),
					resource.TestCheckResourceAttr("lxd_image.img1", "source_image.architecture", architecture),
					resource.TestCheckResourceAttr("lxd_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.#", "0"),
					resource.TestCheckResourceAttr("lxd_image.img1", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_sourceInstance(t *testing.T) {
	projectName := acctest.GenerateName(2, "")
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_sourceInstance(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img1", "source_instance.name", instanceName),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.0", instanceName),
					resource.TestCheckResourceAttrSet("lxd_image.img1", "fingerprint"),
				),
			},
		},
	})
}

func TestAccImage_sourceInstanceAddRemoveAlias(t *testing.T) {
	projectName := acctest.GenerateName(2, "")
	instanceName := acctest.GenerateName(2, "-")
	alias1 := acctest.GenerateName(2, "-")
	alias2 := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_sourceInstanceAliases(projectName, instanceName, alias1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img1", "source_instance.name", instanceName),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.0", alias1),
					resource.TestCheckResourceAttrSet("lxd_image.img1", "fingerprint"),
				),
			},
			{
				// Add a second alias.
				Config: acctest.Provider() + testAccImage_sourceInstanceAliases(projectName, instanceName, alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img1", "source_instance.name", instanceName),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.#", "2"),
					resource.TestCheckTypeSetElemAttr("lxd_image.img1", "aliases.*", alias1),
					resource.TestCheckTypeSetElemAttr("lxd_image.img1", "aliases.*", alias2),
				),
			},
			{
				// Remove the first alias.
				Config: acctest.Provider() + testAccImage_sourceInstanceAliases(projectName, instanceName, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img1", "source_instance.name", instanceName),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.0", alias2),
				),
			},
		},
	})
}

func TestAccImage_sourceInstanceWithSnapshot(t *testing.T) {
	projectName := acctest.GenerateName(2, "")
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccImage_sourceInstanceWithSnapshot(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_image.img1", "source_instance.name", instanceName),
					resource.TestCheckResourceAttr("lxd_image.img1", "source_instance.snapshot", "snap0"),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.#", "1"),
					resource.TestCheckResourceAttr("lxd_image.img1", "aliases.0", instanceName),
					resource.TestCheckResourceAttrSet("lxd_image.img1", "fingerprint"),
				),
			},
		},
	})
}

func testAccImage_basic() string {
	return fmt.Sprintf(`
resource "lxd_image" "img1" {
  source_image = {
    image        = "%s"
    copy_aliases = true
  }
}
	`, acctest.TestCachedImage)
}

func testAccImage_basicVM() string {
	return fmt.Sprintf(`
resource "lxd_image" "img1vm" {
  source_image = {
    image        = "%s"
    type         = "virtual-machine"
    copy_aliases = true
  }
}
	`, acctest.TestCachedImage)
}

func testAccImage_aliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_image" "img2" {
  aliases = ["%s"]

  source_image = {
    image        = "%s"
    copy_aliases = false
  }
}
	`, strings.Join(aliases, `","`), acctest.TestCachedImage)
}

func testAccImage_aliasExists1(alias string) string {
	return fmt.Sprintf(`
resource "lxd_image" "exists1" {
  aliases = ["%s"]

  source_image = {
    image        = "%s"
    copy_aliases = false
  }
}
	`, alias, acctest.TestCachedImage)
}

func testAccImage_aliasExists2(alias string) string {
	return fmt.Sprintf(`
resource "lxd_image" "exists1" {
  aliases = ["%s"]

  source_image = {
    image        = "%s"
    copy_aliases = false
  }
}

resource "lxd_image" "exists2" {
  aliases = ["%s"]

  source_image = {
    image        = "%s"
    copy_aliases = false
  }
}
	`, alias, acctest.TestCachedImage, alias, acctest.TestCachedImage)
}

func testAccImage_copiedAliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_image" "img3" {
  aliases = ["%s","%s"]

  source_image = {
    image        = "%s"
    copy_aliases = true
  }
}
	`, acctest.TestCachedImageSourceImage, strings.Join(aliases, `","`), acctest.TestCachedImage)
}

func testAccImage_aliasCollision() string {
	return fmt.Sprintf(`
resource "lxd_image" "img4" {
  aliases = ["%s/amd64"]

  source_image = {
    image        = "%s"
    copy_aliases = true
  }
}
	`, acctest.TestCachedImageSourceImage, acctest.TestCachedImage)
}

func testAccImage_project(project string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
}

resource "lxd_image" "img1" {
  project = lxd_project.project1.name

  source_image = {
    image = "%s"
  }
}
	`, project, acctest.TestCachedImage)
}

func testAccImage_instanceFromImageFingerprint(project string, instanceName string, instanceRemote string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"

  config = {
    "features.images"   = true
    "features.profiles" = false
  }
}

resource "lxd_image" "img1" {
  project = lxd_project.project1.name

  source_image = {
    image = "%s"
  }
}

resource "lxd_instance" "inst" {
    name    = "%s"
    image   = lxd_image.img1.fingerprint
    remote  = "%s"
    project = lxd_project.project1.name
    running = false
}
	`, project, acctest.TestCachedImage, instanceName, instanceRemote)
}

func testAccImage_architecture(project string, architecture string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%s"
}

resource "lxd_image" "img1" {
  project = lxd_project.project1.name

  source_image = {
    image        = "%s"
    architecture = "%s"
  }
}
	`, project, acctest.TestCachedImage, architecture)
}

func testAccImage_sourceInstance(projectName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%[1]s"

  config = {
    "features.images"   = false
    "features.profiles" = false
  }
}

resource "lxd_instance" "instance1" {
  project = lxd_project.project1.name
  name    = "%[2]s"
  image   = "%[3]s"
  running = false
}

resource "lxd_image" "img1" {
  project = lxd_project.project1.name

  aliases = [lxd_instance.instance1.name]

  source_instance = {
    name = lxd_instance.instance1.name
  }
}
	`, projectName, instanceName, acctest.TestImage)
}

func testAccImage_sourceInstanceAliases(projectName string, instanceName string, aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%[1]s"

  config = {
    "features.images"   = false
    "features.profiles" = false
  }
}

resource "lxd_instance" "instance1" {
  project = lxd_project.project1.name
  name    = "%[2]s"
  image   = "%[3]s"
  running = false
}

resource "lxd_image" "img1" {
  project = lxd_project.project1.name

  aliases = ["%[4]s"]

  source_instance = {
    name = lxd_instance.instance1.name
  }
}
	`, projectName, instanceName, acctest.TestImage, strings.Join(aliases, `","`))
}

func testAccImage_sourceInstanceWithSnapshot(projectName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name = "%[1]s"

  config = {
    "features.images"   = false
    "features.profiles" = false
  }
}

resource "lxd_instance" "instance1" {
  project = lxd_project.project1.name
  name    = "%[2]s"
  image   = "%[3]s"
}

resource "lxd_instance_snapshot" "snapshot1" {
  project  = lxd_project.project1.name
  name     = "snap0"
  instance = lxd_instance.instance1.name
  stateful = false
}

resource "lxd_image" "img1" {
  project = lxd_project.project1.name

  aliases = [lxd_instance.instance1.name]

  source_instance = {
    name     = lxd_instance.instance1.name
    snapshot = lxd_instance_snapshot.snapshot1.name
  }
}
	`, projectName, instanceName, acctest.TestImage)
}
