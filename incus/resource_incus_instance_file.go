package incus

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceIncusInstanceFile() *schema.Resource {
	return &schema.Resource{
		Create: resourceIncusInstanceFileCreate,
		Exists: resourceIncusInstanceFileExists,
		Read:   resourceIncusInstanceFileRead,
		Delete: resourceIncusInstanceFileDelete,

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
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"source"},
			},

			"source": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"content"},
			},

			"uid": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
				Default:  0,
			},

			"gid": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
				Default:  0,
			},

			"mode": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "0755",
			},

			"create_directories": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
			},

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"append": {
				Type:     schema.TypeBool,
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

func resourceIncusInstanceFileCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
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
		RemoteName:        remote,
		InstanceName:      instanceName,
		TargetFile:        d.Get("target_file").(string),
		Content:           d.Get("content").(string),
		Source:            d.Get("source").(string),
		UID:               d.Get("uid").(int),
		GID:               d.Get("gid").(int),
		Mode:              d.Get("mode").(string),
		CreateDirectories: d.Get("create_directories").(bool),
		Append:            d.Get("append").(bool),
	}

	err = instanceUploadFile(server, instanceName, file)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %s", file.TargetFile, err)
	}

	d.SetId(file.String())
	return nil
}

func resourceIncusInstanceFileRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	v, targetFile := newFileIDFromResourceID(d.Id())
	remote, instanceName, err := p.IncusConfig.ParseRemote(v)

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

	_, file, err := server.GetInstanceFile(instanceName, targetFile)
	if err != nil {
		return fmt.Errorf("unable to retrieve file %s:%s: %s", instanceName, targetFile, err)
	}

	log.Printf("[DEBUG] Retrieved file: %#v", file)

	d.Set("instance_name", instanceName)
	d.Set("target_file", targetFile)
	d.Set("uid", file.UID)
	d.Set("gid", file.GID)
	d.Set("mode", fmt.Sprintf("%04o", file.Mode))

	return nil
}

func resourceIncusInstanceFileDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	v, targetFile := newFileIDFromResourceID(d.Id())
	remote, instanceName, err := p.IncusConfig.ParseRemote(v)
	if err != nil {
		return err
	}

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

	err = instanceDeleteFile(server, instanceName, targetFile)
	if err != nil {
		return fmt.Errorf("unable to delete file %s:%s: %s", instanceName, targetFile, err)
	}

	return nil
}

func resourceIncusInstanceFileExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*incusProvider)
	v, targetFile := newFileIDFromResourceID(d.Id())
	remote, instanceName, err := p.IncusConfig.ParseRemote(v)
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
		if isNotFoundError(err) {
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
		if isNotFoundError(err) {
			err = nil
			return
		}

		return
	}

	exists = true

	return
}
