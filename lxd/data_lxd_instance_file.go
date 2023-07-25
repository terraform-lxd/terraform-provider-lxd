package lxd

import (
	"fmt"
	"io"
	"strings"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataLxdInstanceFile() *schema.Resource {
	return &schema.Resource{
		Read: dataLxdInstanceFileRead,

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

func dataLxdInstanceFileRead(d *schema.ResourceData, meta interface{}) error {
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

func waitInstanceCreatedFile(p *lxdProvider, server lxd.InstanceServer, instanceName string, targetFile string, d *schema.ResourceData) error {
	type readerFile struct {
		reader io.ReadCloser
		file   *lxd.InstanceFileResponse
	}

	reader, file, err := server.GetInstanceFile(instanceName, targetFile)
	if err != nil {
		// not available now, use WaitForState to poll.
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
		reader = rf.reader
		file = rf.file
	}

	buf := &strings.Builder{}
	_, err = io.Copy(buf, reader)
	if err != nil {
		return fmt.Errorf("failure reading file %s:%s: %s", instanceName, targetFile, err)
	}

	d.Set("content", buf.String())
	d.Set("mode", fmt.Sprintf("%04o", file.Mode))
	d.Set("uid", int(file.UID))
	d.Set("gid", int(file.GID))
	reader.Close()
	return nil
}
