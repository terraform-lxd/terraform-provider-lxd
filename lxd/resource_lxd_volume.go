package lxd

import (
	"fmt"
	"log"
	"strings"

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

	volId := fmt.Sprintf("%s/%s/%s", name, pool, volType)
	d.SetId(volId)

	return resourceLxdVolumeRead(d, meta)
}

func resourceLxdVolumeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client

	volName, volPool, volType, err := resourceLxdVolumeParseID(d.Id())
	if err != nil {
		return err
	}

	volume, err := client.StoragePoolVolumeTypeGet(volPool, volName, volType)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved volume %s: %#v", volName, volume)

	d.Set("config", volume.Config)

	return nil
}

func resourceLxdVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client

	volName, volPool, volType, err := resourceLxdVolumeParseID(d.Id())
	if err != nil {
		return err
	}

	if d.HasChange("config") {
		volume, err := client.StoragePoolVolumeTypeGet(volPool, volName, volType)
		if err != nil {
			return err
		}

		config := resourceLxdConfigMap(d.Get("config"))
		volume.Config = config

		log.Printf("[DEBUG] Updated volume config: %#v", volume)

		if err := client.StoragePoolVolumeTypePut(volPool, volName, volType, volume); err != nil {
			return err
		}
	}

	return nil
}

func resourceLxdVolumeDelete(d *schema.ResourceData, meta interface{}) (err error) {
	client := meta.(*LxdProvider).Client
	volName, volPool, volType, err := resourceLxdVolumeParseID(d.Id())
	if err != nil {
		return err
	}

	if err = client.StoragePoolVolumeTypeDelete(volPool, volName, volType); err != nil {
		return err
	}

	return nil
}

func resourceLxdVolumeExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	client := meta.(*LxdProvider).Client
	exists = false

	volName, volPool, volType, err := resourceLxdVolumeParseID(d.Id())
	if err != nil {
		return
	}

	_, err = client.StoragePoolVolumeTypeGet(volPool, volName, volType)
	if err == nil {
		exists = true
	}

	return
}

func resourceLxdVolumeParseID(id string) (string, string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("Unable to parse ID")
	}

	return parts[0], parts[1], parts[2], nil
}
