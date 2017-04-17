package lxd

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceLxdVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdVolumeCreate,
		Update: resourceLxdVolumeUpdate,
		Delete: resourceLxdVolumeDelete,
		Exists: resourceLxdVolumeExists,
		Read:   resourceLxdVolumeRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"pool": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "custom",
			},

			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
			},
		},
	}
}

func resourceLxdVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client
	name := d.Get("name").(string)
	pool := d.Get("pool").(string)
	volType := d.Get("type").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("Attempting to create volume %s", name)
	if err := client.StoragePoolVolumeTypeCreate(pool, name, volType, config); err != nil {
		return err
	}

	v := NewVolumeId(pool, name, volType)
	d.SetId(v.String())

	return resourceLxdVolumeRead(d, meta)
}

func resourceLxdVolumeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client

	v := NewVolumeIdFromResourceId(d.Id())
	volume, err := client.StoragePoolVolumeTypeGet(v.pool, v.name, v.volType)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved volume %s: %#v", v.name, volume)

	d.Set("config", volume.Config)

	return nil
}

func resourceLxdVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client

	if d.HasChange("config") {
		v := NewVolumeIdFromResourceId(d.Id())
		volume, err := client.StoragePoolVolumeTypeGet(v.pool, v.name, v.volType)
		if err != nil {
			return err
		}

		config := resourceLxdConfigMap(d.Get("config"))
		volume.Config = config

		log.Printf("[DEBUG] Updated volume config: %#v", volume)

		if err := client.StoragePoolVolumeTypePut(v.pool, v.name, v.volType, volume); err != nil {
			return err
		}
	}

	return nil
}

func resourceLxdVolumeDelete(d *schema.ResourceData, meta interface{}) (err error) {
	client := meta.(*LxdProvider).Client
	v := NewVolumeIdFromResourceId(d.Id())

	if err = client.StoragePoolVolumeTypeDelete(v.pool, v.name, v.volType); err != nil {
		return err
	}

	return nil
}

func resourceLxdVolumeExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	client := meta.(*LxdProvider).Client
	exists = false

	v := NewVolumeIdFromResourceId(d.Id())
	_, err = client.StoragePoolVolumeTypeGet(v.pool, v.name, v.volType)
	if err == nil {
		exists = true
	}

	return
}
