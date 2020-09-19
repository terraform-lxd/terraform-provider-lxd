package lxd

import (
	"log"
	"os"
	"strings"
	"testing"

	"fmt"

	"io/ioutil"

	"path/filepath"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	lxd "github.com/lxc/lxd/lxc/config"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProvider.ResourcesMap["lxd_noop"] = resourceLxdNoOp()

	testAccProviders = map[string]terraform.ResourceProvider{
		"lxd": testAccProvider,
	}

	testAccProvider.ConfigureFunc = testProviderConfigureWrapper
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testProviderConfigureWrapper(d *schema.ResourceData) (interface{}, error) {
	d.Set("refresh_interval", "5s")
	return providerConfigure(d)
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func TestAccLxdProvider_envRemote(t *testing.T) {
	envName := os.Getenv("LXD_REMOTE")

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLxdProvider_basic(envName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", envName),
				),
			},
		},
	})
}

func TestAccLxdProvider_providerRemote(t *testing.T) {
	envName := strings.ToLower(petname.Generate(2, "-"))
	envPort := os.Getenv("LXD_PORT")
	envAddr := os.Getenv("LXD_ADDR")
	envPassword := os.Getenv("LXD_PASSWORD")

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLxdProvider_remote(envName, envAddr, envPort, envPassword),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", envName),
				),
			},
		},
	})
}

func TestAccLxdProvider_imageRemotes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLxdProvider_basic("ubuntu"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", "ubuntu"),
				),
			},
			{
				Config: testAccLxdProvider_basic("ubuntu-daily"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", "ubuntu-daily"),
				),
			},
			{
				Config: testAccLxdProvider_basic("images"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", "images"),
				),
			},
		},
	})
}

func TestAccLxdProvider_socketRemote(t *testing.T) {
	remoteName := strings.ToLower(petname.Generate(2, "-"))
	socketAddr := "/var/snap/lxd/common/lxd/unix.socket"
	addr := fmt.Sprintf("unix:%s", socketAddr)

	envName := os.Getenv("LXD_REMOTE")
	os.Unsetenv("LXD_REMOTE")
	defer os.Setenv("LXD_REMOTE", envName)

	tmpDirName := petname.Generate(1, "")
	tmpDir, err := ioutil.TempDir(os.TempDir(), tmpDirName)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // clean up

	conf := &lxd.Config{}
	conf.Remotes = map[string]lxd.Remote{
		remoteName: {
			Addr: addr,
		},
	}
	conf.DefaultRemote = remoteName
	conf.SaveConfig(filepath.Join(tmpDir, "config.yml"))

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLxdProvider_socketRemote(remoteName, socketAddr),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", remoteName),
				),
			},
		},
	})
}

func TestAccLxdProvider_lxcConfigRemotes(t *testing.T) {
	remoteName := strings.ToLower(petname.Generate(2, "-"))
	remoteAddr := os.Getenv("LXD_ADDR")
	remotePort := os.Getenv("LXD_PORT")
	remotePassword := os.Getenv("LXD_PASSWORD")

	envName := os.Getenv("LXD_REMOTE")
	os.Unsetenv("LXD_REMOTE")
	defer os.Setenv("LXD_REMOTE", envName)

	tmpDirName := petname.Generate(1, "")
	tmpDir, err := ioutil.TempDir(os.TempDir(), tmpDirName)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // clean up

	conf := &lxd.Config{}
	conf.Remotes = map[string]lxd.Remote{
		remoteName: {
			Addr: fmt.Sprintf("%s://%s:%s", os.Getenv("LXD_SCHEME"), remoteAddr, os.Getenv("LXD_PORT")),
		},
	}
	conf.DefaultRemote = remoteName
	conf.SaveConfig(filepath.Join(tmpDir, "config.yml"))

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLxdProvider_lxcConfig1(tmpDir, remoteName, remoteAddr, remotePort, remotePassword),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", remoteName),
					resource.TestCheckResourceAttr("lxd_noop.noop1", "client_name", remoteName),
				),
			},
			{
				Config: testAccLxdProvider_lxcConfig2(tmpDir, remoteName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", remoteName),
					resource.TestCheckResourceAttr("lxd_noop.noop1", "client_name", remoteName),
				),
			},
			{
				Config: testAccLxdProvider_lxcConfig3(tmpDir),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop2", "remote", ""),
					resource.TestCheckResourceAttr("lxd_noop.noop2", "client_name", remoteName),
				),
			},
		},
	})

}

func TestAccLxdProvider_noConfigFile(t *testing.T) {
	envName := strings.ToLower(petname.Generate(2, "-"))
	envPort := os.Getenv("LXD_PORT")
	envAddr := os.Getenv("LXD_ADDR")
	envPassword := os.Getenv("LXD_PASSWORD")

	tmpDirName := petname.Generate(1, "")
	tmpDir, err := ioutil.TempDir(os.TempDir(), tmpDirName)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLxdProvider_configDir(envName, envAddr, envPort, envPassword, tmpDir),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop1", "remote", envName),
				),
			},
		},
	})
}

func testAccPreCheck(t *testing.T) {
	// NoOp
}

func testAccLxdProvider_basic(remote string) string {
	return fmt.Sprintf(`
provider "lxd" {
}

resource "lxd_noop" "noop1" {
	name = "noop1"
	remote = "%s"
}
`, remote)
}

func testAccLxdProvider_remote(remote, addr, port, password string) string {
	return fmt.Sprintf(`
provider "lxd" {
	accept_remote_certificate    = true
	generate_client_certificates = true
	lxd_remote {
		name     = "%s"
		address  = "%s"
		port     = "%s"
		password = "%s"
	}
}

resource "lxd_noop" "noop1" {
	name = "noop1"
	remote = "%s"
}
`, remote, addr, port, password, remote)
}

func testAccLxdProvider_socketRemote(remote, socketAddr string) string {
	return fmt.Sprintf(`
provider "lxd" {
	accept_remote_certificate    = true
	generate_client_certificates = true
	lxd_remote {
		name     = "%s"
		address  = "%s"
		scheme   = "unix"
	}
}

resource "lxd_noop" "noop1" {
	name = "noop1"
	remote = "%s"
}
`, remote, socketAddr, remote)
}

func testAccLxdProvider_lxcConfig1(confDir, remote, addr, port, password string) string {
	return fmt.Sprintf(`
provider "lxd" {
	config_dir                   = "%s"
	accept_remote_certificate    = true
	generate_client_certificates = true
	lxd_remote {
		name     = "%s"
		address  = "%s"
		port     = "%s"
		password = "%s"
	}
}

resource "lxd_noop" "noop1" {
	name = "noop1"
	remote = "%s"
}
`, confDir, remote, addr, port, password, remote)
}

func testAccLxdProvider_lxcConfig2(confDir, remote string) string {

	return fmt.Sprintf(`
provider "lxd" {
	config_dir = "%s"
	accept_remote_certificate = true
	generate_client_certificates = true
}

resource "lxd_noop" "noop1" {
	name = "noop1"
	remote = "%s"
}
`, confDir, remote)
}

// Config that does not set remote name, forcing use of default
func testAccLxdProvider_lxcConfig3(confDir string) string {
	return fmt.Sprintf(`
provider "lxd" {
	config_dir = "%s"
	accept_remote_certificate = true
	generate_client_certificates = true
}

resource "lxd_noop" "noop2" {
	name = "noop2"
}
`, confDir)
}

func testAccLxdProvider_configDir(remote, addr, port, password, confDir string) string {
	return fmt.Sprintf(`
provider "lxd" {
	accept_remote_certificate    = true
	generate_client_certificates = true
	config_dir                   = "%s"
	lxd_remote {
		name     = "%s"
		address  = "%s"
		port     = "%s"
		password = "%s"
	}
}

resource "lxd_noop" "noop1" {
	name = "noop1"
	remote = "%s"
}
`, confDir, remote, addr, port, password, remote)
}

// this NoOp resource allows us to invoke the Terraform testing framework to test the Provider
// without actually calling out to any LXD server's to create or destroy resources.
func resourceLxdNoOp() *schema.Resource {
	return &schema.Resource{
		Create: func(d *schema.ResourceData, meta interface{}) error {
			p := meta.(*lxdProvider)
			remote := p.selectRemote(d)
			_, err := meta.(*lxdProvider).GetServer(remote)
			if err != nil {
				return err
			}

			d.SetId(d.Get("name").(string))
			d.Set("name", d.Get("name"))
			d.Set("client_name", remote)
			d.Set("remote", d.Get("remote"))
			return nil
		},

		Delete: schema.RemoveFromState,

		Read: func(d *schema.ResourceData, meta interface{}) error {
			p := meta.(*lxdProvider)
			remote := p.selectRemote(d)
			_, err := meta.(*lxdProvider).GetServer(remote)
			if err != nil {
				return err
			}

			d.Set("name", d.Get("name"))
			d.Set("remote", d.Get("remote"))
			d.Set("client_name", remote)
			return nil
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			"client_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
