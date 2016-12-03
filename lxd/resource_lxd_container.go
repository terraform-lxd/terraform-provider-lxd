package lxd

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
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
	config := resourceLxdContainerConfigMap(d.Get("config"))

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
	if resp, err = client.Init(name, remote, d.Get("image").(string), &profiles, config, nil, ephem); err != nil {
		return err
	}

	// Wait for the LXC container to be created
	err = client.WaitForSuccess(resp.Operation)
	if err != nil {
		return err
	}

	// Start container
	_, err = client.Action(name, shared.Start, -1, false, false)
	if err != nil {
		// Container has been created, but daemon rejected start request
		return err
	}

	// Wait until the container is in a Running state
	stateConf := &resource.StateChangeConf{
		Target:     []string{"Running"},
		Refresh:    resourceLxdContainerRefresh(client, name),
		Timeout:    3 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	if _, err = stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for container (%s) to become active: %s", name, err)
	}

	d.SetId(name)
	return resourceLxdContainerRead(d, meta)
}

func resourceLxdContainerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*LxdProvider).Client

	container, err := client.ContainerInfo(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Retrieved container %s: %#v", d.Id(), container)

	sshIP := ""

	ct, err := client.ContainerState(d.Id())
	if err != nil {
		return err
	}

	d.Set("status", ct.Status)

	for iface, net := range ct.Network {
		if iface != "lo" {
			for _, ip := range net.Addresses {
				if ip.Family == "inet" {
					d.Set("ip_address", ip.Address)
					sshIP = ip.Address
					d.Set("mac_address", net.Hwaddr)
				}
			}
		}
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

	ct, _ := client.ContainerState(name)
	if ct.Status == "Running" {
		if _, err := client.Action(name, shared.Stop, 30, true, false); err != nil {
			return err
		}

		// Wait until the container is in a Stopped state
		stateConf := &resource.StateChangeConf{
			Target:     []string{"Stopped"},
			Refresh:    resourceLxdContainerRefresh(client, name),
			Timeout:    3 * time.Minute,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		if _, err = stateConf.WaitForState(); err != nil {
			return fmt.Errorf("Error waiting for container (%s) to stop: %s", name, err)
		}
	}

	if _, err = client.Delete(name); err != nil {
		return err
	}

	return nil
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

func resourceLxdContainerConfigMap(c interface{}) map[string]string {
	config := make(map[string]string)
	if v, ok := c.(map[string]interface{}); ok {
		for key, val := range v {
			config[key] = val.(string)
		}
	}

	log.Printf("[DEBUG]: LXD Container Configuration Map: %#v", config)

	return config
}

func resourceLxdContainerRefresh(client *lxd.Client, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		ct, err := client.ContainerState(name)
		if err != nil {
			return ct, "Error", err
		}

		return ct, ct.Status, nil
	}
}

func getContainerState(client *lxd.Client, name string) *shared.ContainerState {
	ct, err := client.ContainerState(name)
	if err != nil {
		return nil
	}
	return ct
}
