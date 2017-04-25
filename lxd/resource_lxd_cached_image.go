package lxd

import (
	"fmt"
	"log"

	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lxc/lxd"
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
		},
	}
}

func resourceLxdCachedImageCreate(d *schema.ResourceData, meta interface{}) error {
	dst := meta.(*LxdProvider).Client
	remote := meta.(*LxdProvider).Remote
	config := meta.(*LxdProvider).Config

	srcName := d.Get("source_remote").(string)
	src, err := lxd.NewClient(config, srcName)
	if err != nil {
		return err
	}

	image := d.Get("source_image").(string)
	// has the user provided an fingerprint or alias?
	aliasTarget := src.GetAlias(image)
	if aliasTarget != "" {
		image = aliasTarget
	}

	aliases := make([]string, 0)
	if v, ok := d.GetOk("aliases"); ok {
		for _, alias := range v.([]interface{}) {
			// Check image alias doesn't already exist on destination
			dstAliasTarget := dst.GetAlias(alias.(string))
			if dstAliasTarget != "" {
				return fmt.Errorf("Image alias already exists on destination: %s", alias.(string))
			}

			aliases = append(aliases, alias.(string))
		}
	}

	// Get data about remote image, also checks it exists
	imgInfo, err := src.GetImageInfo(image)
	if err != nil {
		return err
	}

	copyAliases := d.Get("copy_aliases").(bool)

	// Execute the copy
	err = src.CopyImage(image, dst, copyAliases, aliases, false, false, resourceLxdCachedImageCopyProgressHandler)
	if err != nil {
		return err
	}

	// Image was successfully copied, set resource ID
	id := newCachedImageId(remote, imgInfo.Fingerprint)
	d.SetId(id.resourceId())

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

func resourceLxdCachedImageCopyProgressHandler(prog string) {
	log.Println("[DEBUG] - image copy progress: ", prog)
}

func resourceLxdCachedImageUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client
	id := newCachedImageIdFromResourceId(d.Id())

	if d.HasChange("aliases") {
		old, new := d.GetChange("aliases")
		oldSet := schema.NewSet(schema.HashString, old.([]interface{}))
		newSet := schema.NewSet(schema.HashString, new.([]interface{}))
		aliasesToRemove := oldSet.Difference(newSet)
		aliasesToAdd := newSet.Difference(oldSet)

		// Delete removed
		for _, a := range aliasesToRemove.List() {
			alias := a.(string)
			client.DeleteAlias(alias)
		}
		// Add new
		for _, a := range aliasesToAdd.List() {
			alias := a.(string)
			client.PostAlias(alias, "", id.fingerprint)
		}
	}

	return nil
}

func resourceLxdCachedImageDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client
	id := newCachedImageIdFromResourceId(d.Id())

	err := client.DeleteImage(id.fingerprint)

	return err
}

func resourceLxdCachedImageExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*LxdProvider).Client
	id := newCachedImageIdFromResourceId(d.Id())

	_, err := client.GetImageInfo(id.fingerprint)
	if err != nil {
		if err.Error() == "not found" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceLxdCachedImageRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client
	id := newCachedImageIdFromResourceId(d.Id())

	img, err := client.GetImageInfo(id.fingerprint)
	if err != nil {
		if err.Error() == "not found" {
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

type cachedImageId struct {
	remote      string
	fingerprint string
}

func newCachedImageId(remote, fingerprint string) cachedImageId {
	return cachedImageId{
		remote:      remote,
		fingerprint: fingerprint,
	}
}

func newCachedImageIdFromResourceId(id string) cachedImageId {
	parts := strings.SplitN(id, "/", 2)
	return cachedImageId{
		remote:      parts[0],
		fingerprint: parts[1],
	}
}

func (id cachedImageId) resourceId() string {
	return fmt.Sprintf("%s/%s", id.remote, id.fingerprint)
}
