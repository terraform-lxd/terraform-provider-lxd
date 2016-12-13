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
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				ForceNew: false,
			},

			"device": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"type": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: resourceLxdValidateDeviceType,
						},

						"properties": &schema.Schema{
							Type:     schema.TypeMap,
							Required: true,
						},
					},
				},
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
				ForceNew: true,
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
	image := d.Get("image").(string)
	config := resourceLxdConfigMap(d.Get("config"))
	devices := resourceLxdDevices(d.Get("device"))

	/*
	 * requested_empty_profiles means user requested empty
	 * !requested_empty_profiles but len(profArgs) == 0 means use profile default
	 */
	profiles := []string{}
	if v, ok := d.GetOk("profiles"); ok {
		for _, v := range v.([]interface{}) {
			profiles = append(profiles, v.(string))
		}
	}

	// client.Init = (name string, imgremote string, image string, profiles *[]string, config map[string]string, devices shared.Devices, ephem bool)
	var resp *lxd.Response
	if resp, err = client.Init(name, remote, image, &profiles, config, devices, ephem); err != nil {
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
	name := d.Id()

	container, err := client.ContainerInfo(name)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Retrieved container %s: %#v", name, container)

	ct, err := client.ContainerState(name)
	if err != nil {
		return err
	}

	d.Set("status", ct.Status)

	sshIP := ""
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
	client := meta.(*LxdProvider).Client
	name := d.Id()

	// st will hold the updated container information.
	var st shared.BriefContainerInfo
	var changed bool

	ct, err := client.ContainerInfo(name)
	if err != nil {
		return err
	}
	st.Devices = ct.Devices

	if d.HasChange("profiles") {
		_, newProfiles := d.GetChange("profiles")
		if v, ok := newProfiles.([]interface{}); ok {
			changed = true
			var profiles []string
			for _, p := range v {
				profiles = append(profiles, p.(string))
			}

			st.Profiles = profiles

			log.Printf("[DEBUG] Updated profiles: %#v", st.Profiles)
		}
	}

	if d.HasChange("device") {
		changed = true
		old, new := d.GetChange("device")
		oldDevices := resourceLxdDevices(old)
		newDevices := resourceLxdDevices(new)

		for n, _ := range oldDevices {
			delete(st.Devices, n)
		}

		for n, d := range newDevices {
			if n != "" {
				st.Devices[n] = d
			}
		}

		log.Printf("[DEBUG] Updated device list: %#v", st.Devices)
	}

	if changed {
		err := client.UpdateContainerConfig(name, st)
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceLxdContainerDelete(d *schema.ResourceData, meta interface{}) (err error) {
	client := meta.(*LxdProvider).Client
	name := d.Id()

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

	resp, err := client.Delete(name)
	if err != nil {
		return err
	}

	// Wait for the LXC container to be deleted
	err = client.WaitForSuccess(resp.Operation)
	if err != nil {
		return err
	}

	return nil
}

func resourceLxdContainerExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	client := meta.(*LxdProvider).Client
	name := d.Id()

	exists = false

	ct, err := client.ContainerState(name)
	if err == nil && ct != nil {
		exists = true
	}

	return
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
