package lxd

import (
	"fmt"
	"log"
	"strings"

	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lxc/lxd/shared/api"
)

func resourceLxdSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdSnapshotCreate,
		Delete: resourceLxdSnapshotDelete,
		Exists: resourceLxdSnapshotExists,
		Read:   resourceLxdSnapshotRead,

		Schema: map[string]*schema.Schema{
			"container_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"stateful": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"creation_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceLxdSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client
	remote := meta.(*LxdProvider).Remote

	ctrName := d.Get("container_name").(string)
	snapName := d.Get("name").(string)
	stateful := d.Get("stateful").(bool)

	// stateful snapshots usually fail straight after container creation
	// add a retry loop for creating snapshots
	var err error
	var i int
	for i = 0; i < 5; i++ {
		var resp *api.Response
		resp, err = client.Snapshot(ctrName, snapName, stateful)
		if err != nil {
			return err
		}

		// Wait for snapshot operation to complete
		err = client.WaitForSuccess(resp.Operation)
		if err != nil {
			if stateful && strings.Contains(err.Error(), "Dumping FAILED") {
				log.Printf("[DEBUG] error creating stateful snapshot [%d]: %v", i, err)
				time.Sleep(3 * time.Second)
			} else {
				return err
			}
		} else {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("Failed to create snapshot after %d attempts, last error: %v", i, err)
	}

	snapId := NewSnapshotId(remote, ctrName, snapName)
	d.SetId(snapId.String())

	return resourceLxdSnapshotRead(d, meta)
}

func resourceLxdSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client
	snapId := NewSnapshotIdFromResourceId(d.Id())

	snap, err := client.SnapshotInfo(snapId.LxdId())
	if err != nil {
		if err.Error() == "not found" {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("container_name", snapId.container)
	d.Set("name", snapId.snapshot)
	d.Set("stateful", snap.Stateful)
	d.Set("creation_date", snap.CreationDate.String())

	return nil
}

func resourceLxdSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client
	name := d.Id()

	client.Delete(name)

	return nil
}

func resourceLxdSnapshotExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*LxdProvider).Client
	snapId := NewSnapshotIdFromResourceId(d.Id())

	snap, err := client.SnapshotInfo(snapId.LxdId())

	if err != nil && err.Error() == "not found" {
		err = nil
	}
	if err == nil && snap != nil {
		return true, nil
	}

	return false, err
}

type snapshotId struct {
	remote    string
	container string
	snapshot  string
}

func NewSnapshotId(remote, container, snapshot string) snapshotId {
	return snapshotId{remote, container, snapshot}
}

func NewSnapshotIdFromResourceId(id string) snapshotId {
	pieces := strings.SplitN(id, "/", 3)
	return snapshotId{pieces[0], pieces[1], pieces[2]}
}

func (s snapshotId) String() string {
	return fmt.Sprintf("%s/%s/%s", s.remote, s.container, s.snapshot)
}

func (s snapshotId) LxdId() string {
	return fmt.Sprintf("%s/%s", s.container, s.snapshot)
}
