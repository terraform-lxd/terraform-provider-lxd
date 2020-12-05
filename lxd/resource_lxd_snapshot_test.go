package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccSnapshot_stateless(t *testing.T) {
	containerName := strings.ToLower(petname.Generate(2, "-"))
	snapshotName := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshot_basic(containerName, snapshotName, false),
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
// 	containerName := strings.ToLower(petname.Generate(2, "-"))
// 	snapshotName := strings.ToLower(petname.Generate(2, "-"))

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:  func() { testAccPreCheck(t) },
// 		Providers: testAccProviders,
// 		Steps: []resource.TestStep{
// 			resource.TestStep{
// 				Config: testAccSnapshot_basic(containerName, snapshotName, true),
// 				Check: resource.ComposeTestCheckFunc(
// 					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snapshotName),
// 					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "stateful", "true"),
// 				),
// 			},
// 		},
// 	})
// }

func TestAccSnapshot_multiple(t *testing.T) {

	containerName := strings.ToLower(petname.Generate(2, "-"))
	snap1Name := strings.ToLower(petname.Generate(2, "-"))
	snap2Name := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSnapshot_multiple1(containerName, snap1Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snap1Name),
				),
			},
			{
				Config: testAccSnapshot_multiple2(containerName, snap1Name, snap2Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot1", "name", snap1Name),
					resource.TestCheckResourceAttr("lxd_snapshot.snapshot2", "name", snap2Name),
				),
			},
		},
	})
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
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.12"
  profiles = ["default"]
}

resource "lxd_snapshot" "snapshot1" {
  container_name = "${lxd_container.container1.name}"
  name = "%s"
  stateful = "%v"
}
	`, cName, sName, stateful)
}

func testAccSnapshot_multiple1(cName, sName string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.12"
  profiles = ["default"]
}

resource "lxd_snapshot" "snapshot1" {
  container_name = "${lxd_container.container1.name}"
  name = "%s"
  stateful = "false"
}
	`, cName, sName)
}

func testAccSnapshot_multiple2(cName, sName1, sName2 string) string {
	return fmt.Sprintf(`
resource "lxd_container" "container1" {
  name = "%s"
  image = "images:alpine/3.12"
  profiles = ["default"]
}

resource "lxd_snapshot" "snapshot1" {
  container_name = "${lxd_container.container1.name}"
  name = "%s"
  stateful = "false"
}

resource "lxd_snapshot" "snapshot2" {
  container_name = "${lxd_container.container1.name}"
  name = "%s"
  stateful = "false"
}
	`, cName, sName1, sName2)
}
