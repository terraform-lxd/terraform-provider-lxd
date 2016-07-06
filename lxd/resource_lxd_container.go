package lxd

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
)

func resourceLxdContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdContainerCreate,
		Update: resourceLxdContainerUpdate,
		Delete: resourceLxdContainerDelete,
		Exists: resourceLxdContainerExists,
		Read:   resourceLxdContainerRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"image": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"profiles": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				ForceNew: false,
			},

			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"ephemeral": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"privileged": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: false,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: false,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"mac_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceLxdContainerCreate(d *schema.ResourceData, meta interface{}) error {
	var err error
	client := meta.(*LxdProvider).Client
	remote := meta.(*LxdProvider).Remote

	name := d.Get("name").(string)
	ephem := d.Get("ephemeral").(bool)

	/*
	 * requested_empty_profiles means user requested empty
	 * !requested_empty_profiles but len(profArgs) == 0 means use profile default
	 */
	profiles := []string{}
	if v := d.Get("profiles"); v != nil {
		vs := v.(*schema.Set)
		for _, v := range vs.List() {
			profiles = append(profiles, v.(string))
		}
	}

	//client.Init = (name string, imgremote string, image string, profiles *[]string, config map[string]string, ephem bool)
	var resp *lxd.Response
	if resp, err = client.Init(name, remote, d.Get("image").(string), &profiles, nil, nil, ephem); err != nil {
		return err
	}

	d.SetId(name)

	err = client.WaitForSuccess(resp.Operation)
	if err == nil {
		// Start container
		resp, err = client.Action(name, shared.Start, -1, false, false)
		if err != nil {
			// Container has been created, but daemon rejected start request
			return err
		}

		if err := client.WaitForSuccess(resp.Operation); err != nil {
			// Container could not be started
			return err
		}

		return resourceLxdContainerRead(d, meta)
	}

	// Resource didn't create
	d.SetId("")

	return err
}

func resourceLxdContainerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client

	sshIP := ""
	gotIp := false
	cycles := 0

	// wait for NIC to come up and get IP from DHCP
	for !gotIp && cycles < 15 {
		cycles += 1
		ct, _ := client.ContainerState(d.Get("name").(string))
		d.Set("status", ct.Status)
		for iface, net := range ct.Network {
			if iface != "lo" {
				for _, ip := range net.Addresses {
					if ip.Family == "inet" {
						d.Set("ip_address", ip.Address)
						d.Set("mac_address", net.Hwaddr)
						gotIp = true
					}
				}
			}
		}
		time.Sleep(1 * time.Second)
	}

	// Initialize the connection info
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": sshIP,
	})

	return nil
}

func resourceLxdContainerUpdate(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceLxdContainerDelete(d *schema.ResourceData, meta interface{}) (err error) {
	client := meta.(*LxdProvider).Client
	name := d.Get("name").(string)

	ct, _ := client.ContainerState(d.Get("name").(string))
	if ct.Status == "Running" {
		stopResp, err := client.Action(name, shared.Stop, 30, true, false)
		if err == nil {
			err = client.WaitForSuccess(stopResp.Operation)
		}
	}

	if err == nil {
		var resp *lxd.Response
		if resp, err = client.Delete(name); err == nil {
			err = client.WaitForSuccess(resp.Operation)
		}
	}

	return err
}

func resourceLxdContainerExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	client := meta.(*LxdProvider).Client

	exists = false

	ct, err := client.ContainerState(d.Get("name").(string))
	if err == nil && ct != nil {
		exists = true
	}

	return
}

func getContainerState(client *lxd.Client, name string) *shared.ContainerState {
	ct, err := client.ContainerState(name)
	if err != nil {
		return nil
	}
	return ct
}
