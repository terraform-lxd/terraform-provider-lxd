package incus

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/lxc/incus/shared/api"
)

func resourceIncusNetworkZoneRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceIncusNetworkZoneRecordCreate,
		Update: resourceIncusNetworkZoneRecordUpdate,
		Delete: resourceIncusNetworkZoneRecordDelete,
		Exists: resourceIncusNetworkZoneRecordExists,
		Read:   resourceIncusNetworkZoneRecordRead,
		Importer: &schema.ResourceImporter{
			State: resourceIncusNetworkZoneRecordImport,
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
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Optional: true,
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

func resourceIncusNetworkZoneRecordCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
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
	entries := resourceIncusNetworkZoneRecordEntries(d.Get("entry"))
	config := resourceIncusConfigMap(d.Get("config"))

	log.Printf("[DEBUG] Creating network zone record %s with config: %#v", name, config)

	req := api.NetworkZoneRecordsPost{}
	req.Name = name
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

	return resourceIncusNetworkZoneRecordRead(d, meta)
}

func resourceIncusNetworkZoneRecordRead(d *schema.ResourceData, meta interface{}) error {
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

	// Set the entries in the record
	entries := make([]map[string]interface{}, 0)
	for _, incusentry := range record.Entries {
		entry := make(map[string]interface{})
		entry["type"] = incusentry.Type
		entry["value"] = incusentry.Value
		entry["ttl"] = incusentry.TTL
		entries = append(entries, entry)
	}
	d.Set("entry", entries)

	return nil
}

func resourceIncusNetworkZoneRecordUpdate(d *schema.ResourceData, meta interface{}) error {
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
	zone := d.Get("zone").(string)
	_, etag, err := server.GetNetworkZoneRecord(zone, name)
	if err != nil {
		return err
	}

	config := resourceIncusConfigMap(d.Get("config"))
	entries := resourceIncusNetworkZoneRecordEntries(d.Get("entry"))
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

	return resourceIncusNetworkZoneRecordRead(d, meta)
}

func resourceIncusNetworkZoneRecordDelete(d *schema.ResourceData, meta interface{}) (err error) {
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
	zone := d.Get("zone").(string)

	err = server.DeleteNetworkZoneRecord(zone, name)
	if err != nil && isNotFoundError(err) {
		err = nil
	}

	return err
}

func resourceIncusNetworkZoneRecordExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
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

func resourceIncusNetworkZoneRecordImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
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

	zone := d.Get("zone").(string)

	record, _, err := server.GetNetworkZoneRecord(zone, name)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Import Retrieved network zone record %s: %#v", name, record)

	d.Set("name", name)
	return []*schema.ResourceData{d}, nil
}
