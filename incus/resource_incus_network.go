package incus

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/lxc/incus/shared/api"
)

func resourceIncusNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceIncusNetworkCreate,
		Update: resourceIncusNetworkUpdate,
		Delete: resourceIncusNetworkDelete,
		Exists: resourceIncusNetworkExists,
		Read:   resourceIncusNetworkRead,
		Importer: &schema.ResourceImporter{
			State: resourceIncusNetworkImport,
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
				ValidateFunc: resourceIncusValidateNetworkType,
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

func resourceIncusNetworkCreate(d *schema.ResourceData, meta interface{}) error {
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
		return err
	}

	d.SetId(name)

	return resourceIncusNetworkRead(d, meta)
}

func resourceIncusNetworkRead(d *schema.ResourceData, meta interface{}) error {
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

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	network, _, err := server.GetNetwork(name)
	if err != nil {
		if isNotFoundError(err) {
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
	// of reconcilling data defined in Terraform versus what Incus
	// is returning.
	if v := d.Get("target"); v == "" {
		d.Set("config", network.Config)
	}

	d.Set("description", network.Description)
	d.Set("type", network.Type)
	d.Set("managed", network.Managed)

	return nil
}

func resourceIncusNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
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

	config := resourceIncusConfigMap(d.Get("config"))
	desc := d.Get("description").(string)

	req := api.NetworkPut{
		Config:      config,
		Description: desc,
	}

	err = server.UpdateNetwork(name, req, etag)
	if err != nil {
		return err
	}

	return resourceIncusNetworkRead(d, meta)
}

func resourceIncusNetworkDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*incusProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	name := d.Id()

	err = server.DeleteNetwork(name)
	if err != nil && isNotFoundError(err) {
		err = nil
	}

	return err
}

func resourceIncusNetworkExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
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

	v, _, err := server.GetNetwork(name)
	if err != nil && isNotFoundError(err) {
		err = nil
	}
	if err == nil && v != nil {
		exists = true
	}

	return
}

func resourceIncusNetworkImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
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

	network, _, err := server.GetNetwork(name)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Import Retrieved network %s: %#v", name, network)

	d.Set("name", name)
	return []*schema.ResourceData{d}, nil
}
