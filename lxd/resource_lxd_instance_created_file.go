package lxd

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
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

	err = waitInstanceCreatedFile(p, server, instanceName, file.TargetFile, d)
	if err != nil {
		return err
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

func waitInstanceCreatedFile(p *lxdProvider, server lxd.InstanceServer, instanceName string, targetFile string, d *schema.ResourceData) error {
	type readerFile struct {
		reader io.ReadCloser
		file   *lxd.InstanceFileResponse
	}

	conf := &retry.StateChangeConf{
		Target: []string{"Ready"},
		Refresh: func() (interface{}, string, error) {
			reader, file, err := server.GetInstanceFile(instanceName, targetFile)
			if err != nil {
				return nil, "NotReady", nil
			}
			return &readerFile{reader, file}, "Ready", nil
		},
		Timeout:    3 * time.Minute,
		Delay:      p.RefreshInterval,
		MinTimeout: 3 * time.Second,
	}

	st, err := conf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error out waiting for file %s: %s", targetFile, err)
	}

	rf := st.(*readerFile)
	buf := &strings.Builder{}
	_, err = io.Copy(buf, rf.reader)
	if err != nil {
		return fmt.Errorf("failure reading file %s:%s: %s", instanceName, targetFile, err)
	}

	d.Set("content", buf.String())
	d.Set("mode", fmt.Sprintf("%04o", rf.file.Mode))
	d.Set("uid", int(rf.file.UID))
	d.Set("gid", int(rf.file.GID))
	rf.reader.Close()
	return nil
}
