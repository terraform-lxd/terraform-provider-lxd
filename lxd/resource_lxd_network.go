package lxd

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceLxdNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdNetworkCreate,
		//Update: resourceLxdNetworkUpdate,
		Delete: resourceLxdNetworkDelete,
		Exists: resourceLxdNetworkExists,
		Read:   resourceLxdNetworkRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
				ForceNew: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"managed": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func resourceLxdNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client

	name := d.Get("name").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("[DEBUG] Creating network %s with config: %#v", name, config)
	if err := client.NetworkCreate(name, config); err != nil {
		return err
	}

	d.SetId(name)

	return resourceLxdNetworkRead(d, meta)
}

func resourceLxdNetworkRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client
	name := d.Id()

	network, err := client.NetworkGet(name)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved network %s: %#v", name, network)

	d.Set("config", network.Config)
	d.Set("type", network.Type)
	d.Set("managed", network.Managed)

	return nil
}

func resourceLxdNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	// Network is not able to be updated yet.
	return nil
}

func resourceLxdNetworkDelete(d *schema.ResourceData, meta interface{}) (err error) {
	client := meta.(*LxdProvider).Client
	name := d.Id()

	if err = client.NetworkDelete(name); err != nil {
		return err
	}

	return nil
}

func resourceLxdNetworkExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	client := meta.(*LxdProvider).Client
	name := d.Id()

	exists = false

	if _, err := client.NetworkGet(name); err == nil {
		exists = true
	}

	return
}
