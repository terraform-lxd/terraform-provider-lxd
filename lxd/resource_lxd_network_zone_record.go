package lxd

import (
	"log"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLxdNetworkZoneRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdNetworkZoneRecordCreate,
		Update: resourceLxdNetworkZoneRecordUpdate,
		Delete: resourceLxdNetworkZoneRecordDelete,
		Exists: resourceLxdNetworkZoneRecordExists,
		Read:   resourceLxdNetworkZoneRecordRead,
		Importer: &schema.ResourceImporter{
			State: resourceLxdNetworkZoneRecordImport,
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

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"zone": {
				Type:     schema.TypeString,
				Required: true,
			},

			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"entry": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
							// ValidateFunc: resourceLxdValidateDeviceType,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceLxdNetworkZoneRecordCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
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
	zone := d.Get("zone").(string)
	entries := resourceLxdNetworkZoneRecordEntries(d.Get("entry"))
	config := resourceLxdConfigMap(d.Get("config"))

	log.Printf("[DEBUG] Creating network zone record %s with config: %#v", name, config)
	req := api.NetworkZoneRecordsPost{Name: name}
	req.Config = config
	req.Entries = entries
	req.Description = desc

	mutex.Lock()
	err = server.CreateNetworkZoneRecord(zone, req)
	mutex.Unlock()

	if err != nil {
		return err
	}

	d.SetId(name)

	return resourceLxdNetworkZoneRecordRead(d, meta)
}

func resourceLxdNetworkZoneRecordRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Id()
	zone := d.Get("zone").(string)

	record, _, err := server.GetNetworkZoneRecord(zone, name)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}

		return err
	}

	log.Printf("[DEBUG] Retrieved network zone record %s: %#v", name, record)

	d.Set("config", record.Config)
	d.Set("description", record.Description)
	d.Set("entry", record.Entries)

	return nil
}

func resourceLxdNetworkZoneRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Id()
	zone := d.Get("zone").(string)
	_, etag, err := server.GetNetworkZoneRecord(zone, name)
	if err != nil {
		return err
	}

	config := resourceLxdConfigMap(d.Get("config"))
	entries := resourceLxdNetworkZoneRecordEntries(d.Get("entry"))
	desc := d.Get("description").(string)

	req := api.NetworkZoneRecordPut{
		Config:      config,
		Entries:     entries,
		Description: desc,
	}

	err = server.UpdateNetworkZoneRecord(zone, name, req, etag)
	if err != nil {
		return err
	}

	return resourceLxdNetworkZoneRecordRead(d, meta)
}

func resourceLxdNetworkZoneRecordDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Id()
	zone := d.Get("zone").(string)

	err = server.DeleteNetworkZoneRecord(zone, name)
	if err != nil && isNotFoundError(err) {
		err = nil
	}

	return err
}

func resourceLxdNetworkZoneRecordExists(
	d *schema.ResourceData,
	meta interface{},
) (exists bool, err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Id()
	zone := d.Get("zone").(string)

	exists = false

	v, _, err := server.GetNetworkZoneRecord(zone, name)
	if err != nil && isNotFoundError(err) {
		err = nil
	}
	if err == nil && v != nil {
		exists = true
	}

	return
}

func resourceLxdNetworkZoneRecordImport(
	d *schema.ResourceData,
	meta interface{},
) ([]*schema.ResourceData, error) {
	p := meta.(*lxdProvider)
	remote, name, err := p.LXDConfig.ParseRemote(d.Id())

	if err != nil {
		return nil, err
	}

	d.SetId(name)
	zone := d.Get("zone").(string)

	if p.LXDConfig.DefaultRemote != remote {
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

	record, _, err := server.GetNetworkZoneRecord(zone, name)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Import Retrieved network zone record %s: %#v", name, record)

	d.Set("name", name)
	return []*schema.ResourceData{d}, nil
}
