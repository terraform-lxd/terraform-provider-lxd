package lxd

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/lxc/lxd/shared/api"
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

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},

			"target": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"config": {
				Type:     schema.TypeMap,
				Optional: true,
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

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	name := d.Get("name").(string)
	desc := d.Get("description").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("Attempting to create project %s", name)
	project := api.ProjectsPost{}
	project.Description = desc
	project.Config = config
	project.Name = name

	server.UseProject(name)
	if err := server.CreateProject(project); err != nil {
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

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	name := d.Id()

	server.UseProject(name)
	project, _, err := server.GetProject(name)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved  %s: %#v", name, project)

	d.Set("name", project.Name)
	d.Set("description", project.Description)
	d.Set("config", project.Config)
	d.Set("expanded_config", project.Config)

	return nil
}

func resourceLxdProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	project, etag, err := server.GetProject(d.Id())
	if err != nil {
		return err
	}

	config := resourceLxdConfigMap(d.Get("config"))
	project.Config = config

	log.Printf("[DEBUG] Updated project config: %#v", project)

	put := api.ProjectPut{}
	put.Config = config
	put.Description = d.Get("description").(string)

	server.UseProject(d.Id())
	if err := server.UpdateProject(d.Id(), put, etag); err != nil {
		return err
	}

	return nil
}

func resourceLxdProjectDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	return server.DeleteProject(d.Id())
}

func resourceLxdProjectExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	exists = false

	_, _, err = server.GetProject(d.Id())
	if err == nil {
		exists = true
	}

	return
}
