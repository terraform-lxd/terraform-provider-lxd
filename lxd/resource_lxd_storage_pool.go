package lxd

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceLxdStoragePool() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdStoragePoolCreate,
		Update: resourceLxdStoragePoolUpdate,
		Delete: resourceLxdStoragePoolDelete,
		Exists: resourceLxdStoragePoolExists,
		Read:   resourceLxdStoragePoolRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"driver": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "dir" && value != "lvm" && value != "btrfs" && value != "zfs" {
						errors = append(errors, fmt.Errorf(
							"Only dir, lvm, btrfs, and zfs are supported values for 'driver'"))
					}
					return
				},
			},

			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
				ForceNew: false,
			},

			"remote": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
		},
	}
}

func resourceLxdStoragePoolCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	driver := d.Get("driver").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("Attempting to create storage pool %s", name)
	if err := client.StoragePoolCreate(name, driver, config); err != nil {
		return err
	}

	d.SetId(name)

	return resourceLxdStoragePoolRead(d, meta)
}

func resourceLxdStoragePoolRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	pool, err := client.StoragePoolGet(name)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved storage pool %s: %#v", name, pool)

	d.Set("config", pool.Config)

	return nil
}

func resourceLxdStoragePoolUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	if d.HasChange("config") {
		pool, err := client.StoragePoolGet(name)
		if err != nil {
			return err
		}

		config := resourceLxdConfigMap(d.Get("config"))
		pool.Config = config

		log.Printf("[DEBUG] Updated storage pool %s config: %#v", name, pool)

		if err := client.StoragePoolPut(name, pool); err != nil {
			return err
		}
	}

	return nil
}

func resourceLxdStoragePoolDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	if err = client.StoragePoolDelete(name); err != nil {
		return err
	}

	return nil
}

func resourceLxdStoragePoolExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	name := d.Id()
	exists = false

	_, err = client.StoragePoolGet(name)
	if err == nil {
		exists = true
	}

	return
}
