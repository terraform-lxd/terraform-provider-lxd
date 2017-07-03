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

			"remote": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
		},
	}
}

func resourceLxdSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)

	remote := p.selectRemote(d)
	client, err := p.GetClient(remote)
	if err != nil {
		return err
	}

	ctrName := d.Get("container_name").(string)

	snapPost := api.ContainerSnapshotsPost{}
	snapPost.Name = d.Get("name").(string)
	snapPost.Stateful = d.Get("stateful").(bool)

	// stateful snapshots usually fail straight after container creation
	// add a retry loop for creating snapshots
	var i int
	for i = 0; i < 5; i++ {

		op, err := client.CreateContainerSnapshot(ctrName, snapPost)
		if err != nil {
			return err
		}

		// Wait for snapshot operation to complete
		err = op.Wait()
		if err != nil {
			if snapPost.Stateful && strings.Contains(err.Error(), "Dumping FAILED") {
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

	snapID := NewSnapshotId(remote, ctrName, snapPost.Name)
	d.SetId(snapID.String())

	return resourceLxdSnapshotRead(d, meta)
}

func resourceLxdSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	snapID := NewSnapshotIdFromResourceId(d.Id())

	snap, _, err := client.GetContainerSnapshot(snapID.container, snapID.snapshot)
	if err != nil {
		if err.Error() == "not found" {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("container_name", snapID.container)
	d.Set("name", snapID.snapshot)
	d.Set("stateful", snap.Stateful)
	d.Set("creation_date", snap.CreationDate.String())

	return nil
}

func resourceLxdSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}
	snapID := NewSnapshotIdFromResourceId(d.Id())

	client.DeleteContainerSnapshot(snapID.container, snapID.snapshot)

	return nil
}

func resourceLxdSnapshotExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return false, err
	}
	snapID := NewSnapshotIdFromResourceId(d.Id())

	snap, _, err := client.GetContainerSnapshot(snapID.container, snapID.snapshot)

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
