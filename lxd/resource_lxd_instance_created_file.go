package lxd

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLxdInstanceCreatedFile() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdInstanceCreatedFileCreate,
		Exists: resourceLxdInstanceCreatedFileExists,
		Read:   resourceLxdInstanceCreatedFileRead,
		Delete: resourceLxdInstanceCreatedFileDelete,

		Schema: map[string]*schema.Schema{
			"instance_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"target_file": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"content": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: false,
			},

			"uid": {
				Type:     schema.TypeInt,
				Computed: true,
				ForceNew: false,
			},

			"gid": {
				Type:     schema.TypeInt,
				Computed: true,
				ForceNew: false,
			},

			"mode": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: false,
			},

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceLxdInstanceCreatedFileCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	instanceName := d.Get("instance_name").(string)
	_, _, err = server.GetInstance(instanceName)
	if err != nil {
		return fmt.Errorf("unable to retrieve instance %s: %s", instanceName, err)
	}

	file := File{
		RemoteName:   remote,
		InstanceName: instanceName,
		TargetFile:   d.Get("target_file").(string),
	}

	err = waitInstanceCreatedFile(server, instanceName, file.TargetFile, d)
	if err != nil {
		return fmt.Errorf("timed out waiting for file %s: %s", file.TargetFile, err)
	}

	d.SetId(file.String())
	return nil
}

func resourceLxdInstanceCreatedFileRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	v, targetFile := newFileIDFromResourceID(d.Id())
	remote, instanceName, err := p.LXDConfig.ParseRemote(v)

	remote = p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	_, _, err = server.GetInstance(instanceName)
	if err != nil {
		return fmt.Errorf("unable to retrieve instance %s: %s", instanceName, err)
	}

	reader, file, err := server.GetInstanceFile(instanceName, targetFile)
	if err != nil {
		return fmt.Errorf("unable to retrieve file %s:%s: %s", instanceName, targetFile, err)
	}

	log.Printf("[DEBUG] Retrieved file: %#v", file)

	buf := &strings.Builder{}
	_, err = io.Copy(buf, reader)
	if err != nil {
		return fmt.Errorf("failure reading file %s:%s: %s", instanceName, targetFile, err)
	}
	d.Set("content", buf.String())
	reader.Close()

	d.Set("instance_name", instanceName)
	d.Set("target_file", targetFile)
	d.Set("uid", file.UID)
	d.Set("gid", file.GID)
	d.Set("mode", fmt.Sprintf("%04o", file.Mode))

	return nil
}

func resourceLxdInstanceCreatedFileDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceLxdInstanceCreatedFileExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	v, targetFile := newFileIDFromResourceID(d.Id())
	remote, instanceName, err := p.LXDConfig.ParseRemote(v)
	if err != nil {
		err = fmt.Errorf("unable to determine remote: %s", err)
		return
	}

	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	_, _, err = server.GetInstance(instanceName)
	if err != nil {
		// If the instance could not be found, then the file
		// can't exist. Ignore the error and return with exists
		// set to false.
		if err.Error() == "not found" {
			err = nil
			return
		}

		err = fmt.Errorf("unable to retrieve instance %s: %s", instanceName, err)
		return
	}

	_, _, err = server.GetInstanceFile(instanceName, targetFile)
	if err != nil {
		// If the file could not be found, then it doesn't exist.
		// Ignore the error and return with exists set to false.
		if err.Error() == "not found" {
			err = nil
			return
		}

		return
	}

	exists = true

	return
}

func waitInstanceCreatedFile(server lxd.InstanceServer, instanceName string, targetFile string, d *schema.ResourceData) error {
	targetFile, err := filepath.Abs(targetFile)
	if err != nil {
		return fmt.Errorf("Could not determine destination target %s", targetFile)
	}

	targetIsDir := strings.HasSuffix(targetFile, "/")
	if targetIsDir {
		return fmt.Errorf("Target must be an absolute path with filename")
	}

	remainingTries := 60 * 3 // wait approximately 3 minutes
	for {
		if remainingTries == 0 {
			return fmt.Errorf("timed out waiting for file %s:%s: %s", instanceName, targetFile, err)
		}

		reader, f, err := server.GetInstanceFile(instanceName, targetFile)
		if err != nil {
			<-time.After(time.Second)
			remainingTries--
			continue
		}

		buf := &strings.Builder{}
		_, err = io.Copy(buf, reader)
		if err != nil {
			return fmt.Errorf("failure reading file %s:%s: %s", instanceName, targetFile, err)
		}

		d.Set("content", buf.String())
		d.Set("mode", fmt.Sprintf("%04o", f.Mode))
		d.Set("uid", int(f.UID))
		d.Set("gid", int(f.GID))
		reader.Close()
		return nil
	}
}
