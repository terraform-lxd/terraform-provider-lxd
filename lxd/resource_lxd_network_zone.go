package lxd

import (
	"log"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLxdNetworkZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdNetworkZoneCreate,
		Update: resourceLxdNetworkZoneUpdate,
		Delete: resourceLxdNetworkZoneDelete,
		Exists: resourceLxdNetworkZoneExists,
		Read:   resourceLxdNetworkZoneRead,
		Importer: &schema.ResourceImporter{
			State: resourceLxdNetworkZoneImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"remote": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"config": {
				Type:     schema.TypeMap,
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

func resourceLxdNetworkZoneCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Get("name").(string)
	desc := d.Get("description").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("[DEBUG] Creating network zone %s with config: %#v", name, config)

	req := api.NetworkZonesPost{}
	req.Name = name
	req.Config = config
	req.Description = desc

	mutex.Lock()
	err = server.CreateNetworkZone(req)
	mutex.Unlock()

	if err != nil {
		return err
	}

	d.SetId(name)

	return resourceLxdNetworkZoneRead(d, meta)
}

func resourceLxdNetworkZoneRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Id()

	zone, _, err := server.GetNetworkZone(name)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}

		return err
	}

	log.Printf("[DEBUG] Retrieved network zone %s: %#v", name, zone)

	d.Set("config", zone.Config)
	d.Set("description", zone.Description)

	return nil
}

func resourceLxdNetworkZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Id()

	_, etag, err := server.GetNetworkZone(name)
	if err != nil {
		return err
	}

	config := resourceLxdConfigMap(d.Get("config"))
	desc := d.Get("description").(string)

	req := api.NetworkZonePut{}
	req.Config = config
	req.Description = desc

	err = server.UpdateNetworkZone(name, req, etag)
	if err != nil {
		return err
	}

	return resourceLxdNetworkZoneRead(d, meta)
}

func resourceLxdNetworkZoneDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Id()

	err = server.DeleteNetworkZone(name)
	if err != nil && isNotFoundError(err) {
		err = nil
	}

	return err
}

func resourceLxdNetworkZoneExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Id()

	exists = false

	v, _, err := server.GetNetworkZone(name)
	if err != nil && isNotFoundError(err) {
		err = nil
	}

	if err == nil && v != nil {
		exists = true
	}

	return
}

func resourceLxdNetworkZoneImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	p := meta.(*lxdProvider)
	remote, name, err := p.LXDConfig.ParseRemote(d.Id())
	if err != nil {
		return nil, err
	}

	d.SetId(name)

	if p.LXDConfig.DefaultRemote != remote {
		d.Set("remote", remote)
	}

	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return nil, err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	zone, _, err := server.GetNetworkZone(name)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Import Retrieved network zone %s: %#v", name, zone)

	d.Set("name", name)
	return []*schema.ResourceData{d}, nil
}
