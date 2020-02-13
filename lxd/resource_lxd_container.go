package lxd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	lxd "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

var updateTimeout = int(time.Duration(time.Second * 300).Seconds())

func resourceLxdContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdContainerCreate,
		Update: resourceLxdContainerUpdate,
		Delete: resourceLxdContainerDelete,
		Exists: resourceLxdContainerExists,
		Read:   resourceLxdContainerRead,
		Importer: &schema.ResourceImporter{
			State: resourceLxdContainerImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},

			"image": {
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				DiffSuppressFunc: suppressImageDifferences,
			},

			"profiles": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"device": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: resourceLxdValidateDeviceType,
						},

						"properties": {
							Type:     schema.TypeMap,
							Required: true,
						},
					},
				},
			},

			"config": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"limits": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
			},

			"ephemeral": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"wait_for_network": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: false,
			},

			"privileged": {
				Type:       schema.TypeBool,
				Optional:   true,
				Default:    false,
				ForceNew:   false,
				Deprecated: "Use a config setting of security.privileged=1 instead",
			},

			"file": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"content": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"source": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"target_file": {
							Type:     schema.TypeString,
							Required: true,
						},

						"uid": {
							Type:     schema.TypeInt,
							Optional: true,
						},

						"gid": {
							Type:     schema.TypeInt,
							Optional: true,
						},

						"mode": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"create_directories": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: false,
			},

			"ipv4_address": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: false,
			},

			"ipv6_address": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: false,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"mac_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceLxdContainerCreate(d *schema.ResourceData, meta interface{}) error {
	var err error

	// Using Partial to resume uploading files if there was a previous error.
	d.Partial(true)

	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}
	refreshInterval := meta.(*lxdProvider).RefreshInterval

	name := d.Get("name").(string)
	ephem := d.Get("ephemeral").(bool)
	image := d.Get("image").(string)
	imgRemote := remote
	if imgParts := strings.SplitN(image, ":", 2); len(imgParts) == 2 {
		imgRemote = imgParts[0]
		image = imgParts[1]
	}
	imgServer, err := p.GetImageServer(imgRemote)
	if err != nil {
		return fmt.Errorf("could not create image server client: %v", err)
	}

	// Prepare container config
	config := resourceLxdConfigMap(d.Get("config"))
	config = resourceLxdConfigMapAppend(config, d.Get("limits"), "limits.")

	devices := resourceLxdDevices(d.Get("device"))

	profiles := []string{}
	if v, ok := d.GetOk("profiles"); ok {
		for _, v := range v.([]interface{}) {
			profiles = append(profiles, v.(string))
		}
	}

	// If no profiles were set, use the default profile
	if len(profiles) == 0 {
		profiles = append(profiles, "default")
	}

	// build API request
	createReq := api.ContainersPost{}
	createReq.Name = name
	createReq.Profiles = profiles
	createReq.Config = config
	createReq.Devices = devices
	createReq.Ephemeral = ephem

	// Gather info about source image
	//
	// Optimisation for simplestreams
	var imgInfo *api.Image
	if conn, _ := imgServer.GetConnectionInfo(); conn.Protocol == "simplestreams" {
		imgInfo = &api.Image{}
		imgInfo.Fingerprint = image
		imgInfo.Public = true
		createReq.Source.Alias = image
	} else {
		// Attempt to resolve an image alias
		alias, _, err := imgServer.GetImageAlias(image)
		if err == nil {
			createReq.Source.Alias = image
			image = alias.Target
		}

		// Get the image info
		imgInfo, _, err = imgServer.GetImage(image)
		if err != nil {
			return fmt.Errorf("could not get image info: %v", err)
		}
	}

	// Create container. It will not be running after this operation
	op1, err := server.CreateContainerFromImage(imgServer, *imgInfo, createReq)
	if err != nil {
		return err
	}

	// Wait for the container to be created
	err = op1.Wait()
	if err != nil {
		return fmt.Errorf("failed to create container (%s): %s", name, err)
	}

	// Container has been created, store ID
	d.SetId(name)

	d.SetPartial("name")
	d.SetPartial("image")
	d.SetPartial("profiles")
	d.SetPartial("ephemeral")
	d.SetPartial("privileged")
	d.SetPartial("config")
	d.SetPartial("limits")
	d.SetPartial("device")
	d.SetPartial("remote")

	// Upload any files, if specified,
	// and set the contents to a hash in the State
	if files, ok := d.GetOk("file"); ok {
		for _, v := range files.([]interface{}) {
			f := v.(map[string]interface{})
			file := File{
				ContainerName:     name,
				TargetFile:        f["target_file"].(string),
				Content:           f["content"].(string),
				Source:            f["source"].(string),
				UID:               f["uid"].(int),
				GID:               f["gid"].(int),
				Mode:              f["mode"].(string),
				CreateDirectories: f["create_directories"].(bool),
			}

			if err := containerUploadFile(server, name, file); err != nil {
				return err
			}
		}

		err := d.Set("file", files)
		if err != nil {
			return fmt.Errorf("unable to set file in state: %s", err)
		}
	}

	d.SetPartial("file")
	d.Partial(false)

	// Start container
	startReq := api.ContainerStatePut{
		Action:  "start",
		Timeout: updateTimeout,
		Force:   false,
	}
	op2, err := server.UpdateContainerState(name, startReq, "")
	if err != nil {
		// Container has been created, but daemon rejected start request
		return fmt.Errorf("LXD server rejected request to start container (%s): %s", name, err)
	}

	if err = op2.Wait(); err != nil {
		return fmt.Errorf("failed to start container (%s): %s", name, err)
	}

	// Even though op.Wait has completed,
	// wait until we can see the container is running via a new API call.
	// At a minimum, this adds some padding between API calls.
	stateConf := &resource.StateChangeConf{
		Target:     []string{"Running"},
		Refresh:    resourceLxdContainerRefresh(server, name),
		Timeout:    3 * time.Minute,
		Delay:      refreshInterval,
		MinTimeout: 3 * time.Second,
	}

	if _, err = stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for container (%s) to become active: %s", name, err)
	}

	if d.Get("wait_for_network").(bool) {
		// Lxd will return "Running" even if "inet" has not yet been set.
		// wait until we see an "inet" ip_address before reading the state.
		networkConf := &resource.StateChangeConf{
			Target:     []string{"OK"},
			Refresh:    resourceLxdContainerWaitForNetwork(server, name),
			Timeout:    3 * time.Minute,
			Delay:      refreshInterval,
			MinTimeout: 3 * time.Second,
		}

		if _, err = networkConf.WaitForState(); err != nil {
			return fmt.Errorf("Error waiting for container (%s) network information: %s", name, err)
		}
	}

	return resourceLxdContainerRead(d, meta)
}

func resourceLxdContainerRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	name := d.Id()

	container, _, err := server.GetContainer(name)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Retrieved container %s: %#v", name, container)

	state, _, err := server.GetContainerState(name)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Retrieved container state %s:\n%#v", name, state)

	d.Set("ephemeral", container.Ephemeral)
	d.Set("privileged", false) // Create has no handling for it yet

	config := make(map[string]string)
	limits := make(map[string]string)
	for k, v := range container.Config {
		if strings.Contains(k, "limits.") {
			limits[strings.TrimPrefix(k, "limits.")] = v
		} else if strings.HasPrefix(k, "boot.") {
			config[k] = v
		} else if strings.HasPrefix(k, "environment.") {
			config[k] = v
		} else if strings.HasPrefix(k, "raw.") {
			config[k] = v
		} else if strings.HasPrefix(k, "security.") {
			config[k] = v
		} else if strings.HasPrefix(k, "user.") {
			config[k] = v
		}
	}
	d.Set("config", config)
	d.Set("limits", limits)

	d.Set("status", container.Status)

	sshIP := ""
	// First see if there was an access_interface set.
	// If there was, base ip_address and mac_address off of it.
	var aiFound bool
	if ai, ok := container.Config["user.access_interface"]; ok {
		net := state.Network[ai]
		for _, ip := range net.Addresses {
			if ip.Family == "inet" {
				aiFound = true
				d.Set("ip_address", ip.Address)
				d.Set("ipv4_address", ip.Address)
				sshIP = ip.Address
				d.Set("mac_address", net.Hwaddr)
			}
		}

		if found, addr := findIPv6Address(&net); found {
			d.Set("ipv6_address", addr)
		}
	}

	// If the above wasn't successful, try to automatically
	// determine the ip_address and mac_address.
	if !aiFound {
		for iface, net := range state.Network {
			if iface != "lo" {
				for _, ip := range net.Addresses {
					if ip.Family == "inet" {
						d.Set("ip_address", ip.Address)
						d.Set("ipv4_address", ip.Address)
						sshIP = ip.Address
						d.Set("mac_address", net.Hwaddr)
					}
				}

				if found, addr := findIPv6Address(&net); found {
					d.Set("ipv6_address", addr)
				}
			}
		}
	}

	// Initialize the connection info
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": sshIP,
	})

	// Set the profiles used by the container
	d.Set("profiles", container.Profiles)

	// Set the devices used by the container
	devices := make([]map[string]interface{}, 0)
	for name, lxddevice := range container.Devices {
		device := make(map[string]interface{})
		device["name"] = name
		device["type"] = lxddevice["type"]
		delete(lxddevice, "type")
		device["properties"] = lxddevice
		devices = append(devices, device)
	}
	d.Set("device", devices)

	return nil
}

func resourceLxdContainerUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	name := d.Id()

	// changed determines if an update call needs made.
	var changed bool

	ct, etag, err := server.GetContainer(name)
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

		for n := range oldDevices {
			delete(newContainer.Devices, n)
		}

		for n, d := range newDevices {
			if n != "" {
				newContainer.Devices[n] = d
			}
		}

		log.Printf("[DEBUG] Updated device list: %#v", newContainer.Devices)
	}

	if d.HasChange("limits") {
		changed = true
		oldLimits, newLimits := d.GetChange("limits")

		for k := range oldLimits.(map[string]interface{}) {
			delete(newContainer.Config, k)
		}

		for k, v := range newLimits.(map[string]interface{}) {
			newContainer.Config[fmt.Sprintf("limits.%s", k)] = v.(string)
		}
	}

	if changed {
		log.Printf("[DEBUG] Updating container %s: %#v", name, newContainer)
		op, err := server.UpdateContainer(name, newContainer, etag)
		if err != nil {
			return err
		}
		if err = op.Wait(); err != nil {
			return err
		}
	}

	if d.HasChange("file") {
		oldFiles, newFiles := d.GetChange("file")
		for _, v := range oldFiles.([]interface{}) {
			f := v.(map[string]interface{})
			targetFile := f["target_file"].(string)

			if err := containerDeleteFile(server, name, targetFile); err != nil {
				return err
			}
		}

		for _, v := range newFiles.([]interface{}) {
			f := v.(map[string]interface{})
			newFile := File{
				ContainerName:     name,
				TargetFile:        f["target_file"].(string),
				Content:           f["content"].(string),
				Source:            f["source"].(string),
				UID:               f["uid"].(int),
				GID:               f["gid"].(int),
				Mode:              f["mode"].(string),
				CreateDirectories: f["create_directories"].(bool),
			}

			if err := containerUploadFile(server, name, newFile); err != nil {
				return err
			}
		}
	}

	return resourceLxdContainerRead(d, meta)
}

func resourceLxdContainerDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	refreshInterval := meta.(*lxdProvider).RefreshInterval
	name := d.Id()

	ct, etag, _ := server.GetContainerState(name)
	if ct.Status == "Running" {
		stopReq := api.ContainerStatePut{
			Action:  "stop",
			Timeout: updateTimeout,
		}

		op, err := server.UpdateContainerState(name, stopReq, etag)
		if err != nil {
			return err
		}
		if err = op.Wait(); err != nil {
			return fmt.Errorf("Error waiting for container (%s) to stop: %s", name, err)
		}

		// Even though op.Wait has completed,
		// wait until we can see the container has stopped via a new API call.
		// At a minimum, this adds some padding between API calls.
		stateConf := &resource.StateChangeConf{
			Target:     []string{"Stopped"},
			Refresh:    resourceLxdContainerRefresh(server, name),
			Timeout:    3 * time.Minute,
			Delay:      refreshInterval,
			MinTimeout: 3 * time.Second,
		}

		if _, err = stateConf.WaitForState(); err != nil {
			return fmt.Errorf("Error waiting for container (%s) to stop: %s", name, err)
		}

	}

	op, err := server.DeleteContainer(name)
	if err != nil {
		return err
	}

	// Wait for the LXC container to be deleted
	err = op.Wait()
	if err != nil {
		return err
	}

	return nil
}

func resourceLxdContainerExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return false, err
	}

	name := d.Id()

	exists = false

	ct, _, err := server.GetContainerState(name)
	if err != nil && err.Error() == "not found" {
		err = nil
	}
	if err == nil && ct != nil {
		exists = true
	}

	return
}

func resourceLxdContainerImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	p := meta.(*lxdProvider)
	log.Printf("[DEBUG] Starting import for %s", d.Id())
	parts := strings.SplitN(d.Id(), "/", 2)

	remote, name, err := p.LXDConfig.ParseRemote(parts[0])
	if err != nil {
		return nil, err
	}

	d.SetId(name)
	if p.LXDConfig.DefaultRemote != remote {
		d.Set("remote", remote)
	}

	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return nil, err
	}

	ct, _, err := server.GetContainerState(name)
	if err != nil {
		return nil, err
	}

	if ct == nil {
		return nil, fmt.Errorf("Unable to get container state")
	}

	log.Printf("[DEBUG] Import container state %#v", ct)
	d.SetId(name)
	d.Set("name", name)

	if len(parts) == 2 {
		d.Set("image", parts[1])
	}

	return []*schema.ResourceData{d}, err
}

func resourceLxdContainerRefresh(server lxd.ContainerServer, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		st, _, err := server.GetContainerState(name)
		if err != nil {
			return st, "Error", err
		}

		return st, st.Status, nil
	}
}

func resourceLxdContainerWaitForNetwork(server lxd.ContainerServer, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		st, _, err := server.GetContainerState(name)
		if err != nil {
			return st, "Error", err
		}

		for iface, net := range st.Network {
			if iface != "lo" {
				for _, ip := range net.Addresses {
					if ip.Family == "inet" {
						return st, "OK", nil
					}
				}
			}
		}
		return st, "NOT FOUND", nil
	}
}

// Suppress Diff on empty name
func suppressImageDifferences(k, old, new string, d *schema.ResourceData) bool {
	log.Printf("[DEBUG] comparing old %#v and new %#v :: id %s status %#v", old, new, d.Id(), d.Get("Status"))
	if old == "" && d.Id() != "" {
		// special case for imports, empty image string is the result of nt knowing which base image name/alias was used to create this host
		return true
	}
	return false
}

// Find last global IPv6 address or return any last IPv6 address
// if there is no global address. This works analog to the IPv4 selection
// mechanism but favors global addresses.
func findIPv6Address(network *api.ContainerStateNetwork) (bool, string) {
	var address string

	for _, ip := range network.Addresses {
		if ip.Family == "inet6" && ip.Scope == "global" {
			address = ip.Address
		}
	}

	if len(address) > 0 {
		return true, address
	}

	for _, ip := range network.Addresses {
		if ip.Family == "inet6" {
			address = ip.Address
		}
	}

	return len(address) > 0, address
}
