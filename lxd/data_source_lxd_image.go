package lxd

import (
	"errors"
	"log"
	"strings"

	gospew "github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
)

func dataSourceLxdImage() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceLxdImageRead,

		Schema: map[string]*schema.Schema{
			"remote": {
				Type:     schema.TypeString,
				Optional: true,
				// Expects "provider", "images", "ubuntu", "ubuntu-daily"
				Default:      "provider",
				ForceNew:     true,
				ValidateFunc: validateRemote,
			},

			"alias_regex": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				// ValidateFunc: validateNameRegex,
			},
			"description_regex": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				// ValidateFunc: validateNameRegex,
			},
			"most_recent": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"public": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
			"arch": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "amd64",
				ForceNew: true,
			},
			// Computed values.
			"size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"os": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// dataSourceLxdImageRead performs the image lookup.
func dataSourceLxdImageRead(d *schema.ResourceData, meta interface{}) error {
	var err error
	// get client for Provider remote
	c := meta.(*ProviderConfig).Client

	// Check if datasource is configured to use non-provider remote (i.e. one of the public remotes)
	dRemote := strings.ToLower(d.Get("remote").(string))
	if dRemote != "provider" {
		if remote, ok := lxd.DefaultRemotes[dRemote]; ok {
			c, err = lxd.NewClientFromInfo(lxd.ConnectInfo{
				Name:         dRemote,
				RemoteConfig: remote,
			})
			if err != nil {
				return err
			}
		}
	}

	// rem := meta.(*LxdProvider).Remote
	imInfo, err := c.ListImages()
	if err != nil {
		return err
	}

	for _, v := range imInfo {
		log.Printf("\n[DEBUG] %s", gospew.Sdump(v))

		for _, a := range v.Aliases {
			if a.Name == d.Get("alias_regex").(string) {
				setSchemaDataFromImageInfo(d, v)
				break
			}
		}
	}

	return nil
}

func setSchemaDataFromImageInfo(d *schema.ResourceData, ii shared.ImageInfo) {
	d.SetId(ii.Fingerprint)
	d.Set("arch", ii.Architecture)
	d.Set("public", ii.Public)
	d.Set("size", ii.Size)
	d.Set("fingerprint", ii.Fingerprint)
	d.Set("name", ii.Aliases[0].Name)
	d.Set("description", ii.Aliases[0].Description)
	d.Set("os", ii.Properties["os"])

}

func validateRemote(v interface{}, n string) (ws []string, es []error) {
	val := v.(string)

	switch strings.ToLower(val) {
	case "provider":
	case "ubuntu":
	case "ubuntu-daily":
	case "images":
		// expected values
		return nil, nil
	}
	return nil, append(es, errors.New("Invalid remote value. Should be 'Provider', 'Images', 'Ubuntu' or 'Ubuntu-daily'"))
}
