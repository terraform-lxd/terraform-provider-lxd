package lxd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/lxc/lxd/shared/api"
)

func resourceLxdSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdSnapshotCreate,
		Delete: resourceLxdSnapshotDelete,
		Exists: resourceLxdSnapshotExists,
		Read:   resourceLxdSnapshotRead,

		Schema: map[string]*schema.Schema{
			"container_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"stateful": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"creation_date": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Use created_at instead",
			},

			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
		},
	}
}

func resourceLxdSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)

	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	ctrName := d.Get("container_name").(string)

	snapPost := api.InstanceSnapshotsPost{}
	snapPost.Name = d.Get("name").(string)
	snapPost.Stateful = d.Get("stateful").(bool)

	// stateful snapshots usually fail straight after container creation
	// add a retry loop for creating snapshots
	var i int
	for i = 0; i < 5; i++ {

		op, err := server.CreateInstanceSnapshot(ctrName, snapPost)
		if err != nil {
			return err
		}

		// Wait for snapshot operation to complete
		err = op.Wait()
		if err != nil {
			if snapPost.Stateful && strings.Contains(err.Error(), "Dumping FAILED") {
				log.Printf("[DEBUG] error creating stateful snapshot [%d]: %v", i, err)
				time.Sleep(3 * time.Second)
			} else if strings.Contains(err.Error(), "file has vanished") {
				// ignore, try again
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

	snapID := newSnapshotID(remote, ctrName, snapPost.Name)
	d.SetId(snapID.String())

	return resourceLxdSnapshotRead(d, meta)
}

func resourceLxdSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	snapID := newSnapshotIDFromResourceID(d.Id())

	snap, _, err := server.GetInstanceSnapshot(snapID.container, snapID.snapshot)
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
	d.Set("creation_date", snap.CreatedAt.String())
	d.Set("created_at", snap.CreatedAt.String())

	return nil
}

func resourceLxdSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}
	snapID := newSnapshotIDFromResourceID(d.Id())

	server.DeleteInstanceSnapshot(snapID.container, snapID.snapshot)

	return nil
}

func resourceLxdSnapshotExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}
	snapID := newSnapshotIDFromResourceID(d.Id())

	snap, _, err := server.GetInstanceSnapshot(snapID.container, snapID.snapshot)

	if err != nil && err.Error() == "not found" {
		err = nil
	}
	if err == nil && snap != nil {
		return true, nil
	}

	return false, err
}

type snapshotID struct {
	remote    string
	container string
	snapshot  string
}

func newSnapshotID(remote, container, snapshot string) snapshotID {
	return snapshotID{remote, container, snapshot}
}

func newSnapshotIDFromResourceID(id string) snapshotID {
	pieces := strings.SplitN(id, "/", 3)
	return snapshotID{pieces[0], pieces[1], pieces[2]}
}

func (s snapshotID) String() string {
	return fmt.Sprintf("%s/%s/%s", s.remote, s.container, s.snapshot)
}

func (s snapshotID) LxdID() string {
	return fmt.Sprintf("%s/%s", s.container, s.snapshot)
}
