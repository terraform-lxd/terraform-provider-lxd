package instance_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccLxdInstanceSnapshot_stateless(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	snapshotName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceSnapshot_basic(instanceName, snapshotName, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snapshotName),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "instance", instanceName),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "stateful", "false"),
					resource.TestCheckResourceAttrSet("lxd_snapshot.snapshot1", "created_at"),
				),
			},
		},
	})
}

/*
	Disabling this test until:

- travis test environment updated with CRIU
- some LXD stateful snapshot bugs are isolated and resolved / worked around
e.g.
(00.758590) Error (criu/parasite-syscall.c:532): Unable to connect a transport socket: Permission denied
(00.758600) Error (criu/parasite-syscall.c:134): Can't block signals for 5087: No such process
(00.758607) Error (criu/cr-dump.c:1244): Can't infect (pid: 5087) with parasite
(00.761999) Error (criu/ptrace.c:54): Unable to detach from 5087: No such process
(00.762251) Error (criu/cr-dump.c:1628): Dumping FAILED.
*/
func TestAccLxdInstanceSnapshot_stateful(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	snapshotName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceSnapshot_basic(instanceName, snapshotName, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snapshotName),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "stateful", "true"),
					resource.TestCheckResourceAttrSet("lxd_snapshot.snapshot1", "created_at"),
				),
			},
		},
	})
}

func TestAccLxdInstanceSnapshot_multiple(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	snapshotName1 := petname.Generate(2, "-")
	snapshotName2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceSnapshot_multiple1(instanceName, snapshotName1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snapshotName1),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "instance", instanceName),
					resource.TestCheckResourceAttrSet("lxd_snapshot.snapshot1", "created_at"),
				),
			},
			{
				Config: testAccInstanceSnapshot_multiple2(instanceName, snapshotName1, snapshotName2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snapshotName1),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "instance", instanceName),
					resource.TestCheckResourceAttrSet("lxd_snapshot.snapshot1", "created_at"),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot2", "name", snapshotName2),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot2", "instance", instanceName),
					resource.TestCheckResourceAttrSet("lxd_snapshot.snapshot2", "created_at"),
				),
			},
		},
	})
}

func TestAccLxdInstanceSnapshot_project(t *testing.T) {
	projectName := petname.Name()
	instanceName := petname.Generate(2, "-")
	snapshotName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceSnapshot_project(projectName, instanceName, snapshotName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "project", projectName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snapshotName),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "instance", instanceName),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "project", projectName),
					resource.TestCheckResourceAttrSet("lxd_snapshot.snapshot2", "created_at"),
				),
			},
		},
	})
}

func testAccInstanceSnapshot_basic(cName, sName string, stateful bool) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18"
}

resource "lxd_snapshot" "snapshot1" {
  instance = lxd_instance.instance1.name
  name     = "%s"
  stateful = "%v"
}
	`, cName, sName, stateful)
}

func testAccInstanceSnapshot_multiple1(cName, sName string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name  = "%s"
  image = "images:alpine/3.18"
}

resource "lxd_snapshot" "snapshot1" {
  name     = "%s"
  instance = lxd_instance.instance1.name
  stateful = false
}
	`, cName, sName)
}

func testAccInstanceSnapshot_multiple2(cName, sName1, sName2 string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
}

resource "lxd_snapshot" "snapshot1" {
  name     = "%s"
  instance = lxd_instance.instance1.name
  stateful = "false"
}

resource "lxd_snapshot" "snapshot2" {
  name     = "%s"
  instance = lxd_instance.instance1.name
  stateful = "false"
}
	`, cName, sName1, sName2)
}
func testAccInstanceSnapshot_project(project, instance, snapshot string) string {
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
  name    = "%s"
  image   = "images:alpine/3.18"
  project = lxd_project.project1.name
}

resource "lxd_snapshot" "snapshot1" {
  instance = lxd_instance.instance1.name
  name     = "%s"
  stateful = false
  project  = lxd_project.project1.name
}
	`, project, instance, snapshot)
}
