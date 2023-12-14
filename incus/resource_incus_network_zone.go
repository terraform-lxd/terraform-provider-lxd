package incus

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/lxc/incus/shared/api"
)

func resourceIncusNetworkZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceIncusNetworkZoneCreate,
		Update: resourceIncusNetworkZoneUpdate,
		Delete: resourceIncusNetworkZoneDelete,
		Exists: resourceIncusNetworkZoneExists,
		Read:   resourceIncusNetworkZoneRead,
		Importer: &schema.ResourceImporter{
			State: resourceIncusNetworkZoneImport,
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

func resourceIncusNetworkZoneCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
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
	config := resourceIncusConfigMap(d.Get("config"))

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

	return resourceIncusNetworkZoneRead(d, meta)
}

func resourceIncusNetworkZoneRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
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

func resourceIncusNetworkZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
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

	config := resourceIncusConfigMap(d.Get("config"))
	desc := d.Get("description").(string)

	req := api.NetworkZonePut{}
	req.Config = config
	req.Description = desc

	err = server.UpdateNetworkZone(name, req, etag)
	if err != nil {
		return err
	}

	return resourceIncusNetworkZoneRead(d, meta)
}

func resourceIncusNetworkZoneDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*incusProvider)
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

func resourceIncusNetworkZoneExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*incusProvider)
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

func resourceIncusNetworkZoneImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	p := meta.(*incusProvider)
	remote, name, err := p.IncusConfig.ParseRemote(d.Id())
	if err != nil {
		return nil, err
	}

	d.SetId(name)

	if p.IncusConfig.DefaultRemote != remote {
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
