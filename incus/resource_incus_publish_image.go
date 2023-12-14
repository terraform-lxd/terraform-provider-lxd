package incus

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/lxc/incus/shared/api"
)

func resourceIncusPublishImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceIncusPublishImageCreate,
		Update: resourceIncusPublishImageUpdate,
		Delete: resourceIncusCachedImageDelete,
		Exists: resourceIncusCachedImageExists,
		Read:   resourceIncusPublishImageRead,

		Schema: map[string]*schema.Schema{
			"container": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"fingerprint": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},
			"architecture": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeInt,
				ForceNew: false,
				Computed: true,
			},
			"aliases": {
				Type:     schema.TypeList,
				ForceNew: false,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"properties": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
			},
			"public": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
				ForceNew: true,
			},
			"filename": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"compression_algorithm": {
				Type:     schema.TypeString,
				Default:  "gzip",
				Optional: true,
				ForceNew: true,
			},
			"triggers": {
				Description: "A map of arbitrary strings that, when changed, will force the resource to be replaced.",
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceIncusPublishImageCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)

	dstName := p.selectRemote(d)
	dstServer, err := p.GetInstanceServer(dstName)
	if err != nil {
		return err
	}
	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		dstServer = dstServer.UseProject(project)
	}

	container := d.Get("container").(string)
	ct, _, err := dstServer.GetInstanceState(container)
	if err != nil && isNotFoundError(err) {
		return err
	}

	if ct == nil {
		return fmt.Errorf("Unable to find the container: %s", container)
	}

	if ct.StatusCode != api.Stopped {
		return fmt.Errorf("Container is running")
	}

	aliases := make([]api.ImageAlias, 0)
	if v, ok := d.GetOk("aliases"); ok {
		for _, alias := range v.([]interface{}) {
			// Check image alias doesn't already exist on destination
			dstAliasTarget, _, _ := dstServer.GetImageAlias(alias.(string))
			if dstAliasTarget != nil {
				return fmt.Errorf("Image alias already exists on destination: %s", alias.(string))
			}

			ia := api.ImageAlias{
				Name: alias.(string),
			}

			aliases = append(aliases, ia)
		}
	}

	properties := resourceIncusConfigMap(d.Get("properties"))
	public := d.Get("public").(bool)
	filename := d.Get("filename").(string)
	compressionAlgorithm := d.Get("compression_algorithm").(string)

	// Execute the copy
	// Image copy arguments
	req := api.ImagesPost{
		Filename: filename,
		Aliases:  aliases,
		ImagePut: api.ImagePut{
			Public:     public,
			Properties: properties,
		},
		Source: &api.ImagesPostSource{
			Type: "container",
			Name: container,
		},
		CompressionAlgorithm: compressionAlgorithm,
	}

	op, err := dstServer.CreateImage(req, nil)
	if err != nil {
		return err
	}

	// Wait for operation to finish
	err = op.Wait()
	if err != nil {
		return err
	}

	opAPI := op.Get()

	// Grab the fingerprint
	fingerprint := opAPI.Metadata["fingerprint"].(string)

	// Image was successfully copied, set resource ID
	id := newPublishImageID(dstName, fingerprint)
	d.SetId(id.resourceID())

	return resourceIncusPublishImageRead(d, meta)
}

func resourceIncusPublishImageUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}
	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}
	id := newPublishImageIDFromResourceID(d.Id())

	if d.HasChange("aliases") {
		old, new := d.GetChange("aliases")
		oldSet := schema.NewSet(schema.HashString, old.([]interface{}))
		newSet := schema.NewSet(schema.HashString, new.([]interface{}))
		aliasesToRemove := oldSet.Difference(newSet)
		aliasesToAdd := newSet.Difference(oldSet)

		// Delete removed
		for _, a := range aliasesToRemove.List() {
			alias := a.(string)
			err := server.DeleteImageAlias(alias)
			if err != nil {
				return err
			}
		}
		// Add new
		for _, a := range aliasesToAdd.List() {
			alias := a.(string)

			req := api.ImageAliasesPost{}
			req.Name = alias
			req.Target = id.fingerprint

			err := server.CreateImageAlias(req)
			if err != nil {
				return err
			}
		}
	}

	if d.HasChange("properties") {
		properties := resourceIncusConfigMap(d.Get("properties"))

		req := api.ImagePut{
			Properties: properties,
		}

		err := server.UpdateImage(id.fingerprint, req, "")
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceIncusPublishImageRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*incusProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	id := newPublishImageIDFromResourceID(d.Id())

	img, _, err := server.GetImage(id.fingerprint)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("fingerprint", id.fingerprint)
	d.Set("architecture", img.Architecture)
	d.Set("created_at", img.CreatedAt.Unix())

	var aliases []string
	configAliases := d.Get("aliases").([]interface{})
	configSet := schema.NewSet(schema.HashString, configAliases)

	for _, a := range img.Aliases {
		if configSet.Contains(a.Name) {
			aliases = append(aliases, a.Name)
		} else {
			log.Println("[DEBUG] filtered alias ", a)
		}
	}
	d.Set("aliases", aliases)
	d.Set("properties", img.Properties)

	return nil
}

type publishImageID struct {
	remote      string
	fingerprint string
}

func newPublishImageID(remote, fingerprint string) publishImageID {
	return publishImageID{
		remote:      remote,
		fingerprint: fingerprint,
	}
}

func newPublishImageIDFromResourceID(id string) publishImageID {
	parts := strings.SplitN(id, "/", 2)
	return publishImageID{
		remote:      parts[0],
		fingerprint: parts[1],
	}
}

func (id publishImageID) resourceID() string {
	return fmt.Sprintf("%s/%s", id.remote, id.fingerprint)
}
