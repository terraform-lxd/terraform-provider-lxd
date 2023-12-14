package incus

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/lxc/incus/shared/api"
)

func resourceIncusProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceIncusProfileCreate,
		Update: resourceIncusProfileUpdate,
		Delete: resourceIncusProfileDelete,
		Exists: resourceIncusProfileExists,
		Read:   resourceIncusProfileRead,
		Importer: &schema.ResourceImporter{
			State: resourceIncusProfileImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"device": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: resourceIncusValidateDeviceType,
						},

						"properties": {
							Type:     schema.TypeMap,
							Required: true,
						},
					},
				},
			},

			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceIncusProfileCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	config := resourceIncusConfigMap(d.Get("config"))
	devices := resourceIncusDevices(d.Get("device"))

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	req := api.ProfilesPost{Name: name}
	req.Config = config
	req.Devices = devices
	req.Description = description

	if err := server.CreateProfile(req); err != nil {
		return err
	}

	d.SetId(name)

	return resourceIncusProfileRead(d, meta)
}

func resourceIncusProfileRead(d *schema.ResourceData, meta interface{}) error {
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
	profile, _, err := server.GetProfile(name)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieved profile %s: %#v", name, profile)

	d.Set("description", profile.Description)
	d.Set("config", profile.Config)

	devices := make([]map[string]interface{}, 0)
	for name, incusdevice := range profile.Devices {
		device := make(map[string]interface{})
		device["name"] = name
		device["type"] = incusdevice["type"]
		delete(incusdevice, "type")
		device["properties"] = incusdevice
		devices = append(devices, device)
	}
	d.Set("device", devices)
	return nil
}

func resourceIncusProfileUpdate(d *schema.ResourceData, meta interface{}) error {
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

	if d.HasChange("config") {
		changed = true
		_, newConfig := d.GetChange("config")
		newProfile.Config = resourceIncusConfigMap(newConfig)
	}

	if d.HasChange("device") {
		changed = true
		old, new := d.GetChange("device")
		oldDevices := resourceIncusDevices(old)
		newDevices := resourceIncusDevices(new)

		for n := range oldDevices {
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

	return resourceIncusProfileRead(d, meta)
}

func resourceIncusProfileDelete(d *schema.ResourceData, meta interface{}) (err error) {
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

	return server.DeleteProfile(name)
}

func resourceIncusProfileExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
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

	profile, _, err := server.GetProfile(name)
	if err == nil && profile != nil {
		exists = true
	}

	return
}

func resourceIncusProfileImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
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

	profile, _, err := server.GetProfile(name)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Import Retrieved profile %s: %#v", name, profile)

	d.Set("name", name)
	return []*schema.ResourceData{d}, nil
}
