package lxd

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lxc/lxd/shared/api"
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

			"remote": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
		},
	}
}

func resourceLxdVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	pool := d.Get("pool").(string)
	volType := d.Get("type").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("Attempting to create volume %s", name)
	volume := api.StorageVolumesPost{}
	volume.Name = name
	volume.Type = volType
	volume.Config = config
	if err := client.CreateStoragePoolVolume(pool, volume); err != nil {
		return err
	}

	v := NewVolumeId(pool, name, volType)
	d.SetId(v.String())

	return resourceLxdVolumeRead(d, meta)
}

func resourceLxdVolumeRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	v := NewVolumeIdFromResourceId(d.Id())
	volume, _, err := client.GetStoragePoolVolume(v.pool, v.volType, v.name)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved volume %s: %#v", v.name, volume)

	d.Set("config", volume.Config)

	return nil
}

func resourceLxdVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	if d.HasChange("config") {
		v := NewVolumeIdFromResourceId(d.Id())
		volume, etag, err := client.GetStoragePoolVolume(v.pool, v.volType, v.name)
		if err != nil {
			return err
		}

		config := resourceLxdConfigMap(d.Get("config"))
		volume.Config = config

		log.Printf("[DEBUG] Updated volume config: %#v", volume)

		post := api.StorageVolumePut{}
		post.Config = config
		if err := client.UpdateStoragePoolVolume(v.pool, v.volType, v.name, post, etag); err != nil {
			return err
		}
	}

	return nil
}

func resourceLxdVolumeDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	v := NewVolumeIdFromResourceId(d.Id())

	if err = client.DeleteStoragePoolVolume(v.pool, v.volType, v.name); err != nil {
		return err
	}

	return nil
}

func resourceLxdVolumeExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	exists = false

	v := NewVolumeIdFromResourceId(d.Id())
	_, _, err = client.GetStoragePoolVolume(v.pool, v.volType, v.name)
	if err == nil {
		exists = true
	}

	return
}
