package incus

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
)

func resourceIncusVolumeCopy() *schema.Resource {
	return &schema.Resource{
		Create: resourceIncusVolumeCopyCreate,
		Update: nil,
		Delete: resourceIncusVolumeDelete,
		Exists: resourceIncusVolumeExists,
		Read:   resourceIncusVolumeRead,
		CustomizeDiff: func(ctx context.Context, rd *schema.ResourceDiff, meta interface{}) error {
			return nil
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The name of the destination volume.",
			},
			"remote": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Default:     "",
				Description: "The destination remote.",
			},
			"target": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"pool": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The destination pool.",
			},
			"config": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"expanded_config": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"location": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_remote": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Default:     "",
				Description: "The remote from which the source volume is copied.",
			},
			"source_pool": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The source pool.",
			},
			"source_name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The name of the source volume.",
			},
			"content_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceIncusVolumeCopyCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	if v, ok := d.GetOk("target"); ok && v != "" {
		target := v.(string)
		server = server.UseTarget(target)
	}

	pool := d.Get("pool").(string)
	sourcePool := d.Get("source_pool").(string)

	sourceRemoteName := p.IncusConfig.DefaultRemote
	if rem, ok := d.GetOk("source_remote"); ok && rem != "" {
		sourceRemoteName = rem.(string)
	}
	sourceServer, err := p.GetInstanceServer(sourceRemoteName)
	if err != nil {
		return err
	}
	sourceName := d.Get("source_name").(string)
	sourceVolume := api.StorageVolume{
		Name: sourceName,
		Type: "custom",
	}
	name := d.Get("name").(string)
	args := &incus.StoragePoolVolumeCopyArgs{
		Name:       name,
		VolumeOnly: true,
	}

	log.Printf("Attempting to copy volume %s/%s to %s/%s", sourcePool, sourceName, pool, name)
	op, err := server.CopyStoragePoolVolume(pool, sourceServer, sourcePool, sourceVolume, args)
	if err != nil {
		return err
	}
	if err := op.Wait(); err != nil {
		return err
	}
	v := newVolumeID(pool, name, "custom")
	d.SetId(v.String())

	return resourceIncusVolumeRead(d, meta)
}
