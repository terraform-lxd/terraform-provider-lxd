package lxd

import (
	"fmt"
	"testing"

	"github.com/canonical/lxd/shared/api"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSnapshot_stateless(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	snapshotName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshot_basic(instanceName, snapshotName, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snapshotName),
				),
			},
		},
	})
}

/* Disabling this test until:
- travis test environment updated with CRIU
- some LXD stateful snapshot bugs are isolated and resolved / worked around
e.g.
(00.758590) Error (criu/parasite-syscall.c:532): Unable to connect a transport socket: Permission denied
(00.758600) Error (criu/parasite-syscall.c:134): Can't block signals for 5087: No such process
(00.758607) Error (criu/cr-dump.c:1244): Can't infect (pid: 5087) with parasite
(00.761999) Error (criu/ptrace.c:54): Unable to detach from 5087: No such process
(00.762251) Error (criu/cr-dump.c:1628): Dumping FAILED.
*/
// func TestAccSnapshot_stateful(t *testing.T) {
// 	instanceName := petname.Generate(2, "-")
// 	snapshotName := petname.Generate(2, "-")

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:  func() { testAccPreCheck(t) },
// 		Providers: testAccProviders,
// 		Steps: []resource.TestStep{
// 			resource.TestStep{
// 				Config: testAccSnapshot_basic(instanceName, snapshotName, true),
// 				Check: resource.ComposeTestCheckFunc(
// 					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snapshotName),
// 					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "stateful", "true"),
// 				),
// 			},
// 		},
// 	})
// }

func TestAccSnapshot_multiple(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	snap1Name := petname.Generate(2, "-")
	snap2Name := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshot_multiple1(instanceName, snap1Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snap1Name),
				),
			},
			{
				Config: testAccSnapshot_multiple2(instanceName, snap1Name, snap2Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snap1Name),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot2", "name", snap2Name),
				),
			},
		},
	})
}

func TestAccSnapshot_project(t *testing.T) {
	var project api.Project
	var instance api.Instance
	var snap api.InstanceSnapshot
	projectName := petname.Name()
	instanceName := petname.Generate(2, "-")
	snapName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshot_project(projectName, instanceName, snapName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "lxd_project.project1", &project),
					testAccInstanceRunningInProject(t, "lxd_instance.instance1", &instance, projectName),
					testAccSnapshotExistsInProject(t, "lxd_snapshot.snapshot1", &snap, projectName),
				),
			},
		},
	})
}

func testAccSnapshotExistsInProject(t *testing.T, n string, snap *api.InstanceSnapshot, project string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		client = client.UseProject(project)
		snapID := newSnapshotIDFromResourceID(rs.Primary.ID)
		sn, _, err := client.GetInstanceSnapshot(snapID.container, snapID.snapshot)
		if err != nil {
			return err
		}

		*snap = *sn

		return nil
	}
}

func TestSnapshotId_String(t *testing.T) {
	type fields struct {
		remote    string
		container string
		snapshot  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"basic",
			fields{remote: "local", container: "c1", snapshot: "snap1"},
			"local/c1/snap1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := snapshotID{
				remote:    tt.fields.remote,
				container: tt.fields.container,
				snapshot:  tt.fields.snapshot,
			}
			if got := s.String(); got != tt.want {
				t.Errorf("snapshotId.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshotId_LxdID(t *testing.T) {
	type fields struct {
		remote    string
		container string
		snapshot  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"basic",
			fields{remote: "local", container: "c1", snapshot: "snap1"},
			"c1/snap1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := snapshotID{
				remote:    tt.fields.remote,
				container: tt.fields.container,
				snapshot:  tt.fields.snapshot,
			}
			if got := s.LxdID(); got != tt.want {
				t.Errorf("snapshotId.LxdID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testAccSnapshot_basic(cName, sName string, stateful bool) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default"]
}

resource "lxd_snapshot" "snapshot1" {
  container_name = "${lxd_instance.instance1.name}"
  name = "%s"
  stateful = "%v"
}
	`, cName, sName, stateful)
}

func testAccSnapshot_multiple1(cName, sName string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default"]
}

resource "lxd_snapshot" "snapshot1" {
  container_name = "${lxd_instance.instance1.name}"
  name = "%s"
  stateful = "false"
}
	`, cName, sName)
}

func testAccSnapshot_multiple2(cName, sName1, sName2 string) string {
	return fmt.Sprintf(`
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  profiles = ["default"]
}

resource "lxd_snapshot" "snapshot1" {
  container_name = "${lxd_instance.instance1.name}"
  name = "%s"
  stateful = "false"
}

resource "lxd_snapshot" "snapshot2" {
  container_name = "${lxd_instance.instance1.name}"
  name = "%s"
  stateful = "false"
}
	`, cName, sName1, sName2)
}
func testAccSnapshot_project(project, instance, snapshot string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.profiles" = false
	"features.storage.buckets" = false
  }
}
resource "lxd_instance" "instance1" {
  name = "%s"
  image = "images:alpine/3.18"
  project = lxd_project.project1.name
}

resource "lxd_snapshot" "snapshot1" {
  container_name = "${lxd_instance.instance1.name}"
  name = "%s"
  stateful = "false"
  project = lxd_project.project1.name
}
	`, project, instance, snapshot)
}
