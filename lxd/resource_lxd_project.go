package lxd

import (
	"log"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLxdProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdProjectCreate,
		Update: resourceLxdProjectUpdate,
		Delete: resourceLxdProjectDelete,
		Exists: resourceLxdProjectExists,
		Read:   resourceLxdProjectRead,

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
				Type:                  schema.TypeMap,
				Optional:              true,
				DiffSuppressOnRefresh: true,
				DiffSuppressFunc:      SuppressComputedConfigDiff(ConfigTypeProject),
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

func resourceLxdProjectCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("Attempting to create project %s", name)
	// https://github.com/canonical/lxd/blob/main/shared/api/project.go
	req := api.ProjectsPost{Name: name}
	req.Config = config
	req.Description = description

	// NOTE: https://github.com/canonical/lxd/blob/main/client/interfaces.go
	if err := server.CreateProject(req); err != nil {
		return err
	}

	d.SetId(name)

	return resourceLxdProjectRead(d, meta)
}

func resourceLxdProjectRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
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

func resourceLxdProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
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

	var changed bool
	var newProject api.ProjectPut

	if d.HasChange("description") {
		changed = true
		newDescription := d.Get("description")
		newProject.Description = newDescription.(string)
	} else {
		newProject.Description = project.Description
	}

	newConfig := resourceLxdConfigMap(d.Get("config"))

	if HasComputeConfigChanged(ConfigTypeProject, d, project.Config, newConfig) {
		changed = true
		newProject.Config = ComputeConfig(ConfigTypeProject, d, project.Config, newConfig)
	} else {
		newProject.Config = project.Config
	}

	if changed {
		err := server.UpdateProject(name, newProject, etag)
		if err != nil {
			return err
		}
	}

	return resourceLxdProjectRead(d, meta)
}

func resourceLxdProjectDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()
	server = server.UseProject(name)

	return server.DeleteProject(name)
}

func resourceLxdProjectExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
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
