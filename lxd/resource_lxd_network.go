package lxd

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/lxc/lxd/shared/api"
)

func resourceLxdNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdNetworkCreate,
		Update: resourceLxdNetworkUpdate,
		Delete: resourceLxdNetworkDelete,
		Exists: resourceLxdNetworkExists,
		Read:   resourceLxdNetworkRead,
		Importer: &schema.ResourceImporter{
			State: resourceLxdNetworkImport,
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

			"target": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"type": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				ValidateFunc: resourceLxdValidateNetworkType,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"managed": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceLxdNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}
	project := d.Get("project").(string)
	if project != "" {
		server = server.UseProject(project)
	}

	name := d.Get("name").(string)
	desc := d.Get("description").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("[DEBUG] Creating network %s with config: %#v", name, config)
	req := api.NetworksPost{Name: name}
	req.Config = config
	req.Description = desc

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	if v, ok := d.GetOk("type"); ok && v != "" {
		networkType := v.(string)
		req.Type = networkType
	}

	mutex.Lock()
	err = server.CreateNetwork(req)
	mutex.Unlock()

	if err != nil {
		if err.Error() == "not implemented" {
			err = errNetworksNotImplemented
		}

		return err
	}

	d.SetId(name)

	return resourceLxdNetworkRead(d, meta)
}

func resourceLxdNetworkRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}
	project := d.Get("project").(string)
	if project != "" {
		server = server.UseProject(project)
	}

	name := d.Id()

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	network, _, err := server.GetNetwork(name)
	if err != nil {
		if err.Error() == "not found" {
			d.SetId("")
			return nil
		}

		return err
	}

	log.Printf("[DEBUG] Retrieved network %s: %#v", name, network)

	// Only set config values if we're not running in
	// clustered mode. This is because the cluster shares data
	// with other resources of the same name and will cause
	// config keys to be added and removed without a good way
	// of reconcilling data defined in Terraform versus what LXD
	// is returning.
	if v := d.Get("target"); v == "" {
		d.Set("config", network.Config)
	}

	d.Set("description", network.Description)
	d.Set("type", network.Type)
	d.Set("managed", network.Managed)

	return nil
}

func resourceLxdNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}
	project := d.Get("project").(string)
	if project != "" {
		server = server.UseProject(project)
	}

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	name := d.Id()
	_, etag, err := server.GetNetwork(name)
	if err != nil {
		return err
	}

	config := resourceLxdConfigMap(d.Get("config"))
	desc := d.Get("description").(string)

	req := api.NetworkPut{
		Config:      config,
		Description: desc,
	}

	err = server.UpdateNetwork(name, req, etag)
	if err != nil {
		return err
	}

	return resourceLxdNetworkRead(d, meta)
}

func resourceLxdNetworkDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}
	project := d.Get("project").(string)
	if project != "" {
		server = server.UseProject(project)
	}

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	name := d.Id()

	err = server.DeleteNetwork(name)
	if err != nil && err.Error() == "not found" {
		err = nil
	}

	return err
}

func resourceLxdNetworkExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}
	project := d.Get("project").(string)
	if project != "" {
		server = server.UseProject(project)
	}

	name := d.Id()

	exists = false

	v, _, err := server.GetNetwork(name)
	if err != nil && err.Error() == "not found" {
		err = nil
	}
	if err == nil && v != nil {
		exists = true
	}

	return
}

func resourceLxdNetworkImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
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
	project := d.Get("project").(string)
	if project != "" {
		server = server.UseProject(project)
	}

	network, _, err := server.GetNetwork(name)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Import Retrieved network %s: %#v", name, network)

	d.Set("name", name)
	return []*schema.ResourceData{d}, nil
}
