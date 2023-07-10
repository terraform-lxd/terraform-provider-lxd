package lxd

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLxdContainerFile() *schema.Resource {
	return &schema.Resource{
		Create:             resourceLxdContainerFileCreate,
		Exists:             resourceLxdContainerFileExists,
		Read:               resourceLxdContainerFileRead,
		Delete:             resourceLxdContainerFileDelete,
		DeprecationMessage: "lxd_container_file has been deprecated and will be removed. Please use lxd_instance_file instead.",

		Schema: map[string]*schema.Schema{
			"container_name": {
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

func resourceLxdContainerFileCreate(d *schema.ResourceData, meta interface{}) error {
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

	containerName := d.Get("container_name").(string)
	_, _, err = server.GetContainer(containerName)
	if err != nil {
		return fmt.Errorf("unable to retrieve container %s: %s", containerName, err)
	}

	file := File{
		RemoteName:        remote,
		InstanceName:      containerName,
		TargetFile:        d.Get("target_file").(string),
		Content:           d.Get("content").(string),
		Source:            d.Get("source").(string),
		UID:               d.Get("uid").(int),
		GID:               d.Get("gid").(int),
		Mode:              d.Get("mode").(string),
		CreateDirectories: d.Get("create_directories").(bool),
		Append:            d.Get("append").(bool),
	}

	err = instanceUploadFile(server, containerName, file)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %s", file.TargetFile, err)
	}

	d.SetId(file.String())
	return nil
}

func resourceLxdContainerFileRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	v, targetFile := newFileIDFromResourceID(d.Id())
	remote, containerName, err := p.LXDConfig.ParseRemote(v)

	remote = p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	_, _, err = server.GetContainer(containerName)
	if err != nil {
		return fmt.Errorf("unable to retrieve container %s: %s", containerName, err)
	}

	_, file, err := server.GetContainerFile(containerName, targetFile)
	if err != nil {
		return fmt.Errorf("unable to retrieve file %s:%s: %s", containerName, targetFile, err)
	}

	log.Printf("[DEBUG] Retrieved file: %#v", file)

	d.Set("container_name", containerName)
	d.Set("target_file", targetFile)
	d.Set("uid", file.UID)
	d.Set("gid", file.GID)
	d.Set("mode", fmt.Sprintf("%04o", file.Mode))

	return nil
}

func resourceLxdContainerFileDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	v, targetFile := newFileIDFromResourceID(d.Id())
	remote, containerName, err := p.LXDConfig.ParseRemote(v)
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

	_, _, err = server.GetContainer(containerName)
	if err != nil {
		return fmt.Errorf("unable to retrieve container %s: %s", containerName, err)
	}

	err = instanceDeleteFile(server, containerName, targetFile)
	if err != nil {
		return fmt.Errorf("unable to delete file %s:%s: %s", containerName, targetFile, err)
	}

	return nil
}

func resourceLxdContainerFileExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	v, targetFile := newFileIDFromResourceID(d.Id())
	remote, containerName, err := p.LXDConfig.ParseRemote(v)
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

	_, _, err = server.GetContainer(containerName)
	if err != nil {
		// If the container could not be found, then the file
		// can't exist. Ignore the error and return with exists
		// set to false.
		if err.Error() == "not found" {
			err = nil
			return
		}

		err = fmt.Errorf("unable to retrieve container %s: %s", containerName, err)
		return
	}

	_, _, err = server.GetContainerFile(containerName, targetFile)
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
