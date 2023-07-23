package lxd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

var instanceUpdateTimeout = int(time.Duration(time.Second * 300).Seconds())

func resourceLxdInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdInstanceCreate,
		Update: resourceLxdInstanceUpdate,
		Delete: resourceLxdInstanceDelete,
		Exists: resourceLxdInstanceExists,
		Read:   resourceLxdInstanceRead,
		Importer: &schema.ResourceImporter{
			State: resourceLxdInstanceImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"type": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				ValidateFunc: resourceLxdValidateInstanceType,
			},

			"remote": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},

			"target": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
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
							Default:  "0755",
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

			"start_on_create": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: false,
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceLxdInstanceCreate(d *schema.ResourceData, meta interface{}) error {
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

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	// Prepare instance config
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
	createReq := api.InstancesPost{}
	createReq.Name = name
	createReq.Profiles = profiles
	createReq.Config = config
	createReq.Devices = devices
	createReq.Ephemeral = ephem

	instanceType := d.Get("type").(string)
	if instanceType == "container" {
		createReq.Type = api.InstanceTypeContainer
	} else if instanceType == "virtual-machine" {
		createReq.Type = api.InstanceTypeVM
	} else if instanceType == "" {
		createReq.Type = api.InstanceTypeAny
	} else {
		return fmt.Errorf("invalid type: %s", instanceType)
	}

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

	// Use specific target node in a cluster
	if target, ok := d.GetOk("target"); ok && target != "" {
		server = server.UseTarget(target.(string))
	}

	// Create instance. It will not be running after this operation
	op1, err := server.CreateInstanceFromImage(imgServer, *imgInfo, createReq)
	if err != nil {
		return err
	}

	// Wait for the instance to be created
	err = op1.Wait()
	if err != nil {
		return fmt.Errorf("failed to create instance (%s): %s", name, err)
	}

	// Instance has been created, store ID
	d.SetId(name)

	forceWaitForUpload := false
	files, hasFiles := d.GetOk("file")
	if hasFiles {
		if createReq.Type == api.InstanceTypeVM {
			forceWaitForUpload = true
		} else {
			// Only upload files before starting if the instance is not a vm
			err := uploadFiles(d, server, name, files)
			if err != nil {
				return err
			}
		}
	}

	d.Partial(false)

	if d.Get("start_on_create").(bool) || forceWaitForUpload {
		// Start instance
		startReq := api.InstanceStatePut{
			Action:  "start",
			Timeout: instanceUpdateTimeout,
			Force:   false,
		}
		op2, err := server.UpdateInstanceState(name, startReq, "")
		if err != nil {
			// Instance has been created, but daemon rejected start request
			return fmt.Errorf("LXD server rejected request to start instance (%s): %s", name, err)
		}

		if err = op2.Wait(); err != nil {
			return fmt.Errorf("failed to start instance (%s): %s", name, err)
		}

		// Even though op.Wait has completed,
		// wait until we can see the instance is running via a new API call.
		// At a minimum, this adds some padding between API calls.
		stateConf := &retry.StateChangeConf{
			Target:     []string{"Running"},
			Refresh:    resourceLxdInstanceRefresh(server, name),
			Timeout:    3 * time.Minute,
			Delay:      refreshInterval,
			MinTimeout: 3 * time.Second,
		}

		if _, err = stateConf.WaitForState(); err != nil {
			return fmt.Errorf("Error waiting for instance (%s) to become active: %s", name, err)
		}

		if d.Get("wait_for_network").(bool) || forceWaitForUpload {
			// Lxd will return "Running" even if "inet" has not yet been set.
			// wait until we see an "inet" ip_address before reading the state.
			networkConf := &retry.StateChangeConf{
				Target:     []string{"OK"},
				Refresh:    resourceLxdInstanceWaitForNetwork(server, name),
				Timeout:    3 * time.Minute,
				Delay:      refreshInterval,
				MinTimeout: 3 * time.Second,
			}

			if _, err = networkConf.WaitForState(); err != nil {
				return fmt.Errorf("Error waiting for instance (%s) network information: %s", name, err)
			}
		}
	}

	if forceWaitForUpload {
		// If LXD knows about the VM ip, then the agent must be running.
		// Should be safe to upload now
		err := uploadFiles(d, server, name, files)
		if err != nil {
			return err
		}
	}

	return resourceLxdInstanceRead(d, meta)
}

func resourceLxdInstanceRead(d *schema.ResourceData, meta interface{}) error {
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

	name := d.Id()
	// https://github.com/canonical/lxd/blob/main/client/lxd_instances.go
	instance, _, err := server.GetInstance(name)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Retrieved instance %s: %#v", name, instance)

	state, _, err := server.GetInstanceState(name)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Retrieved instance state %s:\n%#v", name, state)

	if instance.Type == "" {
		// If the LXD server does not support virtualization (e.g. because not
		// supported) or the instances API is not available `instance.Type`
		// might be a blank string. In that case we fall back to `"container"`
		// to avoid constant changes to the resource definition.
		d.Set("type", "container")
	} else {
		d.Set("type", instance.Type)
	}

	d.Set("ephemeral", instance.Ephemeral)
	d.Set("privileged", false) // Create has no handling for it yet
	d.Set("target", instance.Location)

	config := make(map[string]string)
	limits := make(map[string]string)
	for k, v := range instance.Config {
		switch {
		case strings.Contains(k, "limits."):
			limits[strings.TrimPrefix(k, "limits.")] = v
		case strings.HasPrefix(k, "boot."):
			config[k] = v
		case strings.HasPrefix(k, "environment."):
			config[k] = v
		case strings.HasPrefix(k, "raw."):
			config[k] = v
		case strings.HasPrefix(k, "linux."):
			config[k] = v
		case strings.HasPrefix(k, "security."):
			config[k] = v
		case strings.HasPrefix(k, "user."):
			config[k] = v
		}
	}
	d.Set("config", config)
	d.Set("limits", limits)

	d.Set("status", instance.Status)

	sshIP := ""
	// First see if there was an access_interface set.
	// If there was, base ip_address and mac_address off of it.
	var aiFound bool
	if ai, ok := instance.Config["user.access_interface"]; ok {
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

	// Set the profiles used by the instance
	d.Set("profiles", instance.Profiles)

	// Set the devices used by the instance
	devices := make([]map[string]interface{}, 0)
	for name, lxddevice := range instance.Devices {
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

func resourceLxdInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
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
	name := d.Id()

	// changed determines if an update call needs made.
	var changed bool

	ct, etag, err := server.GetInstance(name)
	if err != nil {
		return err
	}

	// Copy the current instance configuration to the updatable instance struct.
	// https://github.com/canonical/lxd/blob/3df4aa84e8a86f5186b312243dc212ff8da06941/shared/api/instance.go#L136
	newInstance := api.InstancePut{
		Architecture: ct.Architecture,
		Config:       ct.Config,
		Devices:      ct.Devices,
		Ephemeral:    ct.Ephemeral,
		Profiles:     ct.Profiles,
		Restore:      ct.Restore,
	}

	if d.HasChanges("config") {
		oldConfig, newConfig := d.GetChange("config")
		oldConfigMap := resourceLxdConfigMap(oldConfig)
		newConfigMap := resourceLxdConfigMap(newConfig)

		for key := range oldConfigMap {
			if _, inNew := newConfigMap[key]; !inNew {
				if _, exists := newInstance.Config[key]; exists {
					changed = true
					delete(newInstance.Config, key)
				}
			}
		}

		for key, value := range newConfigMap {
			if val, exists := newInstance.Config[key]; !exists || val != value {
				changed = true
				newInstance.Config[key] = value
			}
		}
	}

	if d.HasChange("profiles") {
		_, newProfiles := d.GetChange("profiles")
		if v, ok := newProfiles.([]interface{}); ok {
			changed = true
			var profiles []string
			for _, p := range v {
				profiles = append(profiles, p.(string))
			}

			newInstance.Profiles = profiles

			log.Printf("[DEBUG] Updated profiles: %#v", newInstance.Profiles)
		}
	}

	if d.HasChange("device") {
		changed = true
		old, new := d.GetChange("device")
		oldDevices := resourceLxdDevices(old)
		newDevices := resourceLxdDevices(new)

		for n := range oldDevices {
			delete(newInstance.Devices, n)
		}

		for n, d := range newDevices {
			if n != "" {
				newInstance.Devices[n] = d
			}
		}

		log.Printf("[DEBUG] Updated device list: %#v", newInstance.Devices)
	}

	if d.HasChange("limits") {
		changed = true
		oldLimits, newLimits := d.GetChange("limits")

		for k := range oldLimits.(map[string]interface{}) {
			delete(newInstance.Config, k)
		}

		for k, v := range newLimits.(map[string]interface{}) {
			newInstance.Config[fmt.Sprintf("limits.%s", k)] = v.(string)
		}
	}

	if changed {
		log.Printf("[DEBUG] Updating instance %s: %#v", name, newInstance)
		op, err := server.UpdateInstance(name, newInstance, etag)
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

			if err := instanceDeleteFile(server, name, targetFile); err != nil {
				return err
			}
		}

		for _, v := range newFiles.([]interface{}) {
			f := v.(map[string]interface{})
			newFile := File{
				InstanceName:      name,
				TargetFile:        f["target_file"].(string),
				Content:           f["content"].(string),
				Source:            f["source"].(string),
				UID:               f["uid"].(int),
				GID:               f["gid"].(int),
				Mode:              f["mode"].(string),
				CreateDirectories: f["create_directories"].(bool),
			}

			if err := instanceUploadFile(server, name, newFile); err != nil {
				return err
			}
		}
	}

	return resourceLxdInstanceRead(d, meta)
}

func resourceLxdInstanceDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)

	remote := p.selectRemote(d)
	server, err := p.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	refreshInterval := meta.(*lxdProvider).RefreshInterval
	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	name := d.Id()
	ct, etag, _ := server.GetInstanceState(name)
	if ct.Status == "Running" {
		stopReq := api.InstanceStatePut{
			Action:  "stop",
			Timeout: instanceUpdateTimeout,
		}

		op, err := server.UpdateInstanceState(name, stopReq, etag)
		if err != nil {
			return err
		}
		if err = op.Wait(); err != nil {
			return fmt.Errorf("Error waiting for instance (%s) to stop: %s", name, err)
		}

		// Even though op.Wait has completed,
		// wait until we can see the instance has stopped via a new API call.
		// At a minimum, this adds some padding between API calls.
		stateConf := &retry.StateChangeConf{
			Target:     []string{"Stopped"},
			Refresh:    resourceLxdInstanceRefresh(server, name),
			Timeout:    3 * time.Minute,
			Delay:      refreshInterval,
			MinTimeout: 3 * time.Second,
		}

		if _, err = stateConf.WaitForState(); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				// Ephemeral instances will be deleted when they are stopped
				// so we can just return nil here and end the Delete call early.
				return nil
			}

			return fmt.Errorf("Error waiting for instance (%s) to stop: %s", name, err)
		}
	}

	op, err := server.DeleteInstance(name)
	if err != nil {
		return err
	}

	// Wait for the LXD instance to be deleted
	err = op.Wait()
	if err != nil {
		return err
	}

	return nil
}

func resourceLxdInstanceExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
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

	exists = false
	name := d.Id()
	ct, _, err := server.GetInstanceState(name)
	if err != nil && err.Error() == "not found" {
		err = nil
	}
	if err == nil && ct != nil {
		exists = true
	}

	return exists, nil
}

func resourceLxdInstanceImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
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
	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	ct, _, err := server.GetInstanceState(name)
	if err != nil {
		return nil, err
	}

	if ct == nil {
		return nil, fmt.Errorf("Unable to get instance state")
	}

	log.Printf("[DEBUG] Import instance state %#v", ct)
	d.SetId(name)
	d.Set("name", name)

	if len(parts) == 2 {
		d.Set("image", parts[1])
	}

	return []*schema.ResourceData{d}, err
}

func resourceLxdInstanceRefresh(server lxd.InstanceServer, name string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		st, _, err := server.GetInstanceState(name)
		if err != nil {
			return st, "Error", err
		}

		return st, st.Status, nil
	}
}

func resourceLxdInstanceWaitForNetwork(server lxd.InstanceServer, name string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		st, _, err := server.GetInstanceState(name)
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
func findIPv6Address(network *api.InstanceStateNetwork) (bool, string) {
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

func uploadFiles(d *schema.ResourceData, server lxd.InstanceServer, name string, files interface{}) error {
	for _, v := range files.([]interface{}) {
		f := v.(map[string]interface{})
		file := File{
			InstanceName:      name,
			TargetFile:        f["target_file"].(string),
			Content:           f["content"].(string),
			Source:            f["source"].(string),
			UID:               f["uid"].(int),
			GID:               f["gid"].(int),
			Mode:              f["mode"].(string),
			CreateDirectories: f["create_directories"].(bool),
		}

		if err := instanceUploadFile(server, name, file); err != nil {
			return err
		}
	}

	err := d.Set("file", files)
	if err != nil {
		return fmt.Errorf("unable to set file in state: %s", err)
	}

	return nil
}
