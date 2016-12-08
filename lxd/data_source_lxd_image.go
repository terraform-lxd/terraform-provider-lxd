package lxd

import (
	"errors"
	"log"
	"regexp"
	"strings"

	"sort"

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
				Default:  true,
				ForceNew: true,
			},
			"public": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
			"architecture": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "amd64",
				ForceNew: true,
			},
			"release": {
				Type:     schema.TypeString,
				Optional: true,
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

type Matches []shared.ImageInfo

func (m Matches) Len() int {
	return len(m)
}

func (m Matches) Less(i, j int) bool {
	return m[i].CreationDate.Unix() > m[j].CreationDate.Unix()
}

func (m Matches) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
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

	var matches Matches

	for _, v := range imInfo {
		log.Printf("\n[DEBUG] %s", gospew.Sdump(v))

		arch := d.Get("architecture").(string)
		if imgArch, ok := v.Properties["architecture"]; ok {
			if imgArch != arch {
				continue
			}
		}

		releaseFilter := d.Get("release").(string)
		if rel, ok := v.Properties["release"]; ok {
			if rel != releaseFilter {
				continue
			}
		}

		desc := d.Get("description_regex").(string)
		if imgDesc, ok := v.Properties["description"]; ok {
			if matched, _ := regexp.MatchString(desc, imgDesc); !matched {
				continue
			}
		}
		// for _, a := range v.Aliases {
		// 	if a.Name == d.Get("alias_regex").(string) {
		// 		setSchemaDataFromImageInfo(d, v)
		// 		break
		// 	}
		// }
		matches = append(matches, v)
	}

	if len(matches) > 1 {
		if d.Get("most_recent").(bool) {
			sort.Sort(matches)
			setSchemaDataFromImageInfo(d, matches[0])
			return nil
		}
		return errors.New("lookup returned too many results & most_recent == false")
	}

	return nil
}

func setSchemaDataFromImageInfo(d *schema.ResourceData, ii shared.ImageInfo) {
	d.SetId(ii.Fingerprint)
	d.Set("architecture", ii.Properties["architecture"])
	d.Set("public", ii.Public)
	d.Set("size", ii.Size)
	d.Set("fingerprint", ii.Fingerprint)
	d.Set("name", ii.Aliases[0].Name)
	d.Set("description", ii.Aliases[0].Description)
	d.Set("os", ii.Properties["os"])
	d.Set("release", ii.Properties["release"])

}

func validateRemote(v interface{}, n string) (ws []string, es []error) {
	val := v.(string)

	switch strings.ToLower(val) {
	case "provider":
		return nil, nil
	case "ubuntu":
		return nil, nil
	case "ubuntu-daily":
		return nil, nil
	case "images":
		// expected values
		return nil, nil
	}
	return nil, append(es, errors.New("Invalid remote value. Should be 'Provider', 'Images', 'Ubuntu' or 'Ubuntu-Daily'"))
}
