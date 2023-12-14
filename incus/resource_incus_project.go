package incus

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/lxc/incus/shared/api"
)

func resourceIncusProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceIncusProjectCreate,
		Update: resourceIncusProjectUpdate,
		Delete: resourceIncusProjectDelete,
		Exists: resourceIncusProjectExists,
		Read:   resourceIncusProjectRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},

			"target": {
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   false,
				Deprecated: "This attribute is ignored",
			},
		},
	}
}

func resourceIncusProjectCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	config := resourceIncusConfigMap(d.Get("config"))

	log.Printf("Attempting to create project %s", name)
	// https://github.com/lxc/incus/blob/main/shared/api/project.go
	req := api.ProjectsPost{Name: name}
	req.Config = config
	req.Description = description

	// NOTE: https://github.com/lxc/incus/blob/main/client/interfaces.go
	if err := server.CreateProject(req); err != nil {
		return err
	}

	d.SetId(name)

	return resourceIncusProjectRead(d, meta)
}

func resourceIncusProjectRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()
	server = server.UseProject(name)
	project, _, err := server.GetProject(name)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved project %s: %#v", name, project)

	d.Set("description", project.Description)
	d.Set("config", project.Config)
	d.Set("name", project.Name)

	return nil
}

func resourceIncusProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()
	server = server.UseProject(name)
	project, etag, err := server.GetProject(name)
	if err != nil {
		return err
	}

	// Copy the current project config into the updatable project struct.
	newProject := api.ProjectPut{
		Config:      project.Config,
		Description: project.Description,
	}

	var changed bool

	if d.HasChange("description") {
		changed = true
		_, newDescription := d.GetChange("description")
		newProject.Description = newDescription.(string)
	}

	if d.HasChange("config") {
		changed = true
		_, newConfig := d.GetChange("config")
		newProject.Config = resourceIncusConfigMap(newConfig)
	}

	if changed {
		err := server.UpdateProject(name, newProject, etag)
		if err != nil {
			return err
		}
	}

	return resourceIncusProjectRead(d, meta)
}

func resourceIncusProjectDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*incusProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()
	server = server.UseProject(name)

	return server.DeleteProject(name)
}

func resourceIncusProjectExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*incusProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	name := d.Id()
	server = server.UseProject(name)

	exists = false

	project, _, err := server.GetProject(name)
	if err == nil && project != nil {
		exists = true
	}

	return
}
