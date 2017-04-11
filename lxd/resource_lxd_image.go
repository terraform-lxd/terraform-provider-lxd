package lxd

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceLxdImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdImageCreate,
		Update: resourceLxdImageUpdate,
		Delete: resourceLxdImageDelete,
		Exists: resourceLxdImageExists,
		Read:   resourceLxdImageRead,

		Schema: map[string]*schema.Schema{

			"source_remote": {
				Type:     schema.TypeSet,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"scheme": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions["lxd_scheme"],
							Default:     "https",
						},

						"port": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions["lxd_port"],
							DefaultFunc: schema.EnvDefaultFunc("LXD_PORT", "8443"),
						},

						"remote": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions["lxd_remote"],
							DefaultFunc: schema.EnvDefaultFunc("LXD_IMAGES_REMOTE", "images"),
						},

						"remote_password": &schema.Schema{
							Type:        schema.TypeString,
							Sensitive:   true,
							Optional:    true,
							Description: descriptions["lxc_remote_password"],
							DefaultFunc: schema.EnvDefaultFunc("LXD_REMOTE_PASSWORD", ""),
						},

						"config_dir": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions["lxd_config_dir"],
							DefaultFunc: func() (interface{}, error) {
								return os.ExpandEnv("$HOME/.config/lxc"), nil
							},
						},
					},
				},
			},

			"copy_aliases": {
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			"aliases": &schema.Schema{
				Type:     schema.TypeList,
				ForceNew: false,
				Required: false,
			},

			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceLxdImageCreate(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceLxdImageUpdate(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceLxdImageDelete(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceLxdImageExists(d *schema.ResourceData, meta interface{}) (bool, error) {

	return false, nil
}

func resourceLxdImageRead(d *schema.ResourceData, meta interface{}) error {

	return nil
}
