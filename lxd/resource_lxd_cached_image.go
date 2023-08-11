package lxd

import (
	"fmt"
	"log"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLxdCachedImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdCachedImageCreate,
		Update: resourceLxdCachedImageUpdate,
		Delete: resourceLxdCachedImageDelete,
		Exists: resourceLxdCachedImageExists,
		Read:   resourceLxdCachedImageRead,

		Schema: map[string]*schema.Schema{

			"aliases": {
				Type:     schema.TypeList,
				ForceNew: false,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"copy_aliases": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
				ForceNew: true,
			},

			"source_image": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"source_remote": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},

			"type": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "container",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "container" && value != "virtual-machine" {
						errors = append(errors, fmt.Errorf(
							"Only container and virtual-machine are supported values for 'type'"))
					}
					return
				},
			},

			// Computed attributes

			"architecture": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"created_at": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"copied_aliases": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceLxdCachedImageCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)

	dstName := p.selectRemote(d)
	dstServer, err := p.GetInstanceServer(dstName)
	if err != nil {
		return err
	}
	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		dstServer = dstServer.UseProject(project)
	}

	srcName := d.Get("source_remote").(string)
	imgServer, err := p.GetImageServer(srcName)
	if err != nil {
		return err
	}

	var imageType string
	if v, ok := d.GetOk("type"); ok && v != "" {
		imageType = v.(string)
	} else {
		return fmt.Errorf("Missing image type")
	}

	image := d.Get("source_image").(string)
	// has the user provided an fingerprint or alias?
	aliasTarget, _, _ := imgServer.GetImageAliasType(imageType, image)
	if aliasTarget != nil {
		image = aliasTarget.Target
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

	// Get data about remote image, also checks it exists
	imgInfo, _, err := imgServer.GetImage(image)
	if err != nil {
		return err
	}

	copyAliases := d.Get("copy_aliases").(bool)

	// Execute the copy
	// Image copy arguments
	args := lxd.ImageCopyArgs{
		Aliases: aliases,
		Public:  false,
	}

	op, err := dstServer.CopyImage(imgServer, *imgInfo, &args)
	if err != nil {
		return err
	}

	// Wait for operation to finish
	err = op.Wait()
	if err != nil {
		return err
	}

	// Image was successfully copied, set resource ID
	id := newCachedImageID(dstName, imgInfo.Fingerprint)
	d.SetId(id.resourceID())

	// store remote aliases that we've copied, so we can filter them out later
	copied := make([]string, 0)
	if copyAliases {
		for _, a := range imgInfo.Aliases {
			copied = append(copied, a.Name)
		}
	}
	d.Set("copied_aliases", copied)

	return resourceLxdCachedImageRead(d, meta)
}

func resourceLxdCachedImageUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}
	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}
	id := newCachedImageIDFromResourceID(d.Id())

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

	return nil
}

func resourceLxdCachedImageDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}
	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	id := newCachedImageIDFromResourceID(d.Id())

	op, err := server.DeleteImage(id.fingerprint)
	if err != nil {
		return err
	}

	return op.Wait()
}

func resourceLxdCachedImageExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return false, err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	id := newCachedImageIDFromResourceID(d.Id())

	_, _, err = server.GetImage(id.fingerprint)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceLxdCachedImageRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}
	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	id := newCachedImageIDFromResourceID(d.Id())

	img, _, err := server.GetImage(id.fingerprint)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("fingerprint", id.fingerprint)
	d.Set("source_remote", d.Get("source_remote"))
	d.Set("copy_aliases", d.Get("copy_aliases"))
	d.Set("architecture", img.Architecture)
	d.Set("created_at", img.CreatedAt.Unix())

	// Read aliases from img and set in resource data
	// If the user has set 'copy_aliases' to true, then the
	// locally cached image will have aliases set that aren't
	// in the Terraform config.
	// These need to be filtered out here so not to cause a diff.
	var aliases []string
	copiedAliases := d.Get("copied_aliases").([]interface{})
	configAliases := d.Get("aliases").([]interface{})
	copiedSet := schema.NewSet(schema.HashString, copiedAliases)
	configSet := schema.NewSet(schema.HashString, configAliases)

	for _, a := range img.Aliases {
		if configSet.Contains(a.Name) || !copiedSet.Contains(a.Name) {
			aliases = append(aliases, a.Name)
		} else {
			log.Println("[DEBUG] filtered alias ", a)
		}
	}
	d.Set("aliases", aliases)

	return nil
}

type cachedImageID struct {
	remote      string
	fingerprint string
}

func newCachedImageID(remote, fingerprint string) cachedImageID {
	return cachedImageID{
		remote:      remote,
		fingerprint: fingerprint,
	}
}

func newCachedImageIDFromResourceID(id string) cachedImageID {
	parts := strings.SplitN(id, "/", 2)
	return cachedImageID{
		remote:      parts[0],
		fingerprint: parts[1],
	}
}

func (id cachedImageID) resourceID() string {
	return fmt.Sprintf("%s/%s", id.remote, id.fingerprint)
}
