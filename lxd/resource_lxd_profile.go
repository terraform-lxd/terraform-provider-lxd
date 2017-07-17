package lxd

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/lxc/lxd/shared/api"
)

func resourceLxdProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdProfileCreate,
		Update: resourceLxdProfileUpdate,
		Delete: resourceLxdProfileDelete,
		Exists: resourceLxdProfileExists,
		Read:   resourceLxdProfileRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"device": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"type": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: resourceLxdValidateDeviceType,
						},

						"properties": &schema.Schema{
							Type:     schema.TypeMap,
							Required: true,
						},
					},
				},
			},

			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
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

func resourceLxdProfileCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	server, err := p.GetContainerServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	config := resourceLxdConfigMap(d.Get("config"))
	devices := resourceLxdDevices(d.Get("device"))

	req := api.ProfilesPost{Name: name}
	req.Config = config
	req.Devices = devices
	req.Description = description

	if err := server.CreateProfile(req); err != nil {
		return err
	}

	d.SetId(name)

	return resourceLxdProfileRead(d, meta)
}

func resourceLxdProfileRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	server, err := p.GetContainerServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	profile, _, err := server.GetProfile(name)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved profile %s: %#v", name, profile)

	d.Set("description", profile.Description)
	d.Set("config", profile.Config)
	d.Set("device", profile.Devices)

	return nil
}

func resourceLxdProfileUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	server, err := p.GetContainerServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	var changed bool

	profile, etag, err := server.GetProfile(name)
	if err != nil {
		return err
	}

	// Copy the current profile config into the updatable profile struct.
	newProfile := api.ProfilePut{
		Config:      profile.Config,
		Description: profile.Description,
		Devices:     profile.Devices,
	}

	if d.HasChange("description") {
		changed = true
		_, newDescription := d.GetChange("description")
		newProfile.Description = newDescription.(string)
	}

	if d.HasChange("device") {
		changed = true
		old, new := d.GetChange("device")
		oldDevices := resourceLxdDevices(old)
		newDevices := resourceLxdDevices(new)

		for n, _ := range oldDevices {
			delete(newProfile.Devices, n)
		}

		for n, d := range newDevices {
			if n != "" {
				newProfile.Devices[n] = d
			}
		}

		log.Printf("[DEBUG] Updated device list: %#v", newProfile.Devices)
	}

	if changed {
		err := server.UpdateProfile(name, newProfile, etag)
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceLxdProfileDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*LxdProvider)
	server, err := p.GetContainerServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Id()

	if err = server.DeleteProfile(name); err != nil {
		return err
	}

	return nil
}

func resourceLxdProfileExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*LxdProvider)
	server, err := p.GetContainerServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	name := d.Id()

	exists = false

	profile, _, err := server.GetProfile(name)
	if err == nil && profile != nil {
		exists = true
	}

	return
}
