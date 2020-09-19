package lxd

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/lxc/lxd/shared/api"
)

func resourceLxdNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdNetworkCreate,
		//Update: resourceLxdNetworkUpdate,
		Delete: resourceLxdNetworkDelete,
		Exists: resourceLxdNetworkExists,
		Read:   resourceLxdNetworkRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"config": {
				Type:     schema.TypeMap,
				Required: true,
				ForceNew: true,
			},

			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"managed": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"remote": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
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

	name := d.Get("name").(string)
	desc := d.Get("description").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("[DEBUG] Creating network %s with config: %#v", name, config)
	req := api.NetworksPost{Name: name}
	req.Config = config
	req.Description = desc

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
	name := d.Id()

	network, _, err := server.GetNetwork(name)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved network %s: %#v", name, network)

	d.Set("config", network.Config)
	d.Set("description", network.Description)
	d.Set("type", network.Type)
	d.Set("managed", network.Managed)

	return nil
}

func resourceLxdNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	// Network is not able to be updated yet.
	return nil
}

func resourceLxdNetworkDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	return server.DeleteNetwork(name)
}

func resourceLxdNetworkExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	name := d.Id()

	exists = false

	if _, _, err := server.GetNetwork(name); err == nil {
		exists = true
	}

	return
}
