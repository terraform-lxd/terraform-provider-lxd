package lxd

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lxc/lxd/shared/api"
)

func resourceLxdStoragePool() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdStoragePoolCreate,
		Update: resourceLxdStoragePoolUpdate,
		Delete: resourceLxdStoragePoolDelete,
		Exists: resourceLxdStoragePoolExists,
		Read:   resourceLxdStoragePoolRead,
		Importer: &schema.ResourceImporter{
			State: resourceLxdStoragePoolImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"driver": {
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

			"config": {
				Type:     schema.TypeMap,
				Required: true,
				ForceNew: false,
			},

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
		},
	}
}

func resourceLxdStoragePoolCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	driver := d.Get("driver").(string)
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("Attempting to create storage pool %s", name)
	post := api.StoragePoolsPost{}
	post.Name = name
	post.Driver = driver
	post.Config = config

	if err := server.CreateStoragePool(post); err != nil {
		return err
	}

	d.SetId(name)

	return resourceLxdStoragePoolRead(d, meta)
}

func resourceLxdStoragePoolRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	pool, _, err := server.GetStoragePool(name)
	if err != nil {
		return err
	}
	d.Set("driver", pool.Driver)
	config := pool.Config
	delete(config, "name")
	for k := range config {
		if strings.HasPrefix(k, "volatile") {
			// The original source is stored under volatile.initial_source
			// so we override "source" with its value.
			if k == "volatile.initial_source" {
				config["source"] = config[k]
			}

			// Delete all "volatile" keys.
			delete(config, k)
		}
	}
	d.Set("config", config)

	log.Printf("[DEBUG] Retrieved storage pool %s: %#v", name, pool)

	return nil
}

func resourceLxdStoragePoolUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	if d.HasChange("config") {
		pool, etag, err := server.GetStoragePool(name)
		if err != nil {
			return err
		}

		config := resourceLxdConfigMap(d.Get("config"))
		pool.Config = config

		log.Printf("[DEBUG] Updated storage pool %s config: %#v", name, pool)

		post := api.StoragePoolPut{}
		post.Config = config
		if err := server.UpdateStoragePool(name, post, etag); err != nil {
			return err
		}
	}

	return nil
}

func resourceLxdStoragePoolDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	return server.DeleteStoragePool(name)
}

func resourceLxdStoragePoolExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	name := d.Id()
	exists = false

	_, _, err = server.GetStoragePool(name)
	if err == nil {
		exists = true
	}

	return
}

func resourceLxdStoragePoolImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	p := meta.(*lxdProvider)
	remote, name, err := p.LXDConfig.ParseRemote(d.Id())

	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Import storage pool from remote: %s name: %s", remote, name)

	d.SetId(name)
	if p.LXDConfig.DefaultRemote != remote {
		d.Set("remote", remote)
	}

	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return nil, err
	}

	pool, _, err := server.GetStoragePool(name)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Import Retrieved storage pool %s: %#v", name, pool)

	d.Set("name", name)
	return []*schema.ResourceData{d}, nil
}
