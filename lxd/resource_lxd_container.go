package lxd

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
	"github.com/lxc/lxd/shared/api"
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

			"file": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"content": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"target_file": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"uid": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"gid": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"mode": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"create_directories": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
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

	profiles := []string{}
	if v, ok := d.GetOk("profiles"); ok {
		for _, v := range v.([]interface{}) {
			profiles = append(profiles, v.(string))
		}
	}

	// client.Init = (name string, imgremote string, image string, profiles *[]string, config map[string]string, devices shared.Devices, ephem bool)
	var resp *api.Response
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

	// Upload any files, if specified
	if v, ok := d.GetOk("file"); ok {
		for _, v := range v.([]interface{}) {
			if err := resourceLxdContainerUploadFile(client, name, v); err != nil {
				return err
			}
		}
	}

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

	// changed determines if an update call needs made.
	var changed bool

	ct, err := client.ContainerInfo(name)
	if err != nil {
		return err
	}

	// Copy the current container configuration to the updatable container struct.
	newContainer := api.ContainerPut{
		Architecture: ct.Architecture,
		Config:       ct.Config,
		Devices:      ct.Devices,
		Ephemeral:    ct.Ephemeral,
		Profiles:     ct.Profiles,
		Restore:      ct.Restore,
	}

	if d.HasChange("profiles") {
		_, newProfiles := d.GetChange("profiles")
		if v, ok := newProfiles.([]interface{}); ok {
			changed = true
			var profiles []string
			for _, p := range v {
				profiles = append(profiles, p.(string))
			}

			newContainer.Profiles = profiles

			log.Printf("[DEBUG] Updated profiles: %#v", newContainer.Profiles)
		}
	}

	if d.HasChange("device") {
		changed = true
		old, new := d.GetChange("device")
		oldDevices := resourceLxdDevices(old)
		newDevices := resourceLxdDevices(new)

		for n, _ := range oldDevices {
			delete(newContainer.Devices, n)
		}

		for n, d := range newDevices {
			if n != "" {
				newContainer.Devices[n] = d
			}
		}

		log.Printf("[DEBUG] Updated device list: %#v", newContainer.Devices)
	}

	if changed {
		log.Printf("[DEBUG] Updating container %s: %#v", name, newContainer)
		err := client.UpdateContainerConfig(name, newContainer)
		if err != nil {
			return err
		}
	}

	if d.HasChange("file") {
		oldFiles, newFiles := d.GetChange("file")
		for _, v := range oldFiles.([]interface{}) {
			if err := resourceLxdContainerDeleteFile(client, name, v); err != nil {
				return err
			}
		}

		for _, v := range newFiles.([]interface{}) {
			if err := resourceLxdContainerUploadFile(client, name, v); err != nil {
				return err
			}
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
	if err != nil && err.Error() == "not found" {
		err = nil
	}
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

func resourceLxdContainerUploadFile(client *lxd.Client, container string, file interface{}) error {
	var uid, gid int
	var createDirectories bool
	fileInfo := file.(map[string]interface{})

	fileContent := fileInfo["content"].(string)
	fileTarget := fileInfo["target_file"].(string)

	if v, ok := fileInfo["uid"]; ok {
		uid = v.(int)
	}

	if v, ok := fileInfo["gid"]; ok {
		gid = v.(int)
	}

	if v, ok := fileInfo["create_directories"]; ok {
		createDirectories = v.(bool)
	}

	fileTarget, err := filepath.Abs(fileTarget)
	if err != nil {
		return fmt.Errorf("Could not santize destination target %s", fileTarget)
	}

	targetIsDir := strings.HasSuffix(fileTarget, "/")
	if targetIsDir {
		return fmt.Errorf("Target must be an absolute path with filename")
	}

	mode := os.FileMode(0755)
	if v, ok := fileInfo["mode"].(string); ok && v != "" {
		if len(v) != 3 {
			v = "0" + v
		}

		m, err := strconv.ParseInt(v, 0, 0)
		if err != nil {
			return fmt.Errorf("Could not determine file mode %s", v)
		}

		mode = os.FileMode(m)
	}

	log.Printf("[DEBUG] Attempting to upload file to %s with uid %d, gid %d, and mode %s",
		fileTarget, uid, gid, fmt.Sprintf("%04o", mode.Perm()))

	if createDirectories {
		if err := client.MkdirP(container, path.Dir(fileTarget), mode, uid, gid); err != nil {
			return fmt.Errorf("Could not create path %s", path.Dir(fileTarget))
		}
	}

	f := strings.NewReader(fileContent)
	if err := client.PushFile(
		container, fileTarget, gid, uid, fmt.Sprintf("%04o", mode.Perm()), f); err != nil {
		return fmt.Errorf("Could not upload file %s: %s", fileTarget, err)
	}

	log.Printf("[DEBUG] Successfully uploaded file %s", fileTarget)

	return nil
}

func resourceLxdContainerDeleteFile(client *lxd.Client, container string, file interface{}) error {
	fileInfo := file.(map[string]interface{})
	fileTarget := fileInfo["target_file"].(string)
	fileTarget, err := filepath.Abs(fileTarget)
	if err != nil {
		return fmt.Errorf("Could not santize destination target %s", fileTarget)
	}

	targetIsDir := strings.HasSuffix(fileTarget, "/")
	if targetIsDir {
		return fmt.Errorf("Target must be an absolute path with filename")
	}

	log.Printf("[DEBUG] Attempting to delete file %s", fileTarget)

	if err := client.DeleteFile(container, fileTarget); err != nil {
		return fmt.Errorf("Could not delete file %s: %s", fileTarget, err)
	}

	log.Printf("[DEBUG] Successfully deleted file %s", fileTarget)

	return nil
}
