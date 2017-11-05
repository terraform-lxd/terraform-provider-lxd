package lxd

import (
	"crypto/sha1"
	"encoding/hex"
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
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"remote": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},

			"image": &schema.Schema{
				Type:             schema.TypeString,
				ForceNew:         true,
				Required:         true,
				DiffSuppressFunc: suppressImageDifferences,
			},

			"profiles": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: false,
				Computed: true,
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

			"limits": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
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
							StateFunc: func(v interface{}) string {
								hash := sha1.Sum([]byte(v.(string)))
								return hex.EncodeToString(hash[:])
							},
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

	// Using Partial to resume uploading files if there was a previous error.
	d.Partial(true)

	p := meta.(*LxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetContainerServer(remote)
	if err != nil {
		return err
	}
	refreshInterval := meta.(*LxdProvider).RefreshInterval

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
			file := v.(map[string]interface{})
			if err := resourceLxdContainerUploadFile(server, name, file); err != nil {
				return err
			}
			hash := sha1.Sum([]byte(file["content"].(string)))
			file["content"] = hex.EncodeToString(hash[:])
		}

		d.Set("file", files)
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

	return resourceLxdContainerRead(d, meta)
}

func resourceLxdContainerRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetContainerServer(remote)
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
				sshIP = ip.Address
				d.Set("mac_address", net.Hwaddr)
			}
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
						sshIP = ip.Address
						d.Set("mac_address", net.Hwaddr)
					}
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

	return nil
}

func resourceLxdContainerUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetContainerServer(remote)
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
			if err := resourceLxdContainerDeleteFile(server, name, v); err != nil {
				return err
			}
		}

		for _, v := range newFiles.([]interface{}) {
			if err := resourceLxdContainerUploadFile(server, name, v); err != nil {
				return err
			}
		}
	}

	return nil
}

func resourceLxdContainerDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*LxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetContainerServer(remote)
	if err != nil {
		return err
	}

	refreshInterval := meta.(*LxdProvider).RefreshInterval
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
	p := meta.(*LxdProvider)
	remote := p.selectRemote(d)
	server, err := p.GetContainerServer(remote)
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
	p := meta.(*LxdProvider)
	log.Printf("[DEBUG] Starting import for %s", d.Id())
	parts := strings.SplitN(d.Id(), "/", 2)

	remote, name, err := p.Config.ParseRemote(parts[0])
	if err != nil {
		return nil, err
	}

	d.SetId(name)
	if p.Config.DefaultRemote != remote {
		d.Set("remote", remote)
	}

	server, err := p.GetContainerServer(p.selectRemote(d))
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

func resourceLxdContainerUploadFile(server lxd.ContainerServer, container string, file interface{}) error {
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
		fileTarget, uid, gid, fmt.Sprintf("%04o", mode))

	if createDirectories {
		err := recursiveMkdir(server, container, path.Dir(fileTarget), mode, int64(uid), int64(gid))
		if err != nil {
			return fmt.Errorf("Could not upload file %s: %s", fileTarget, err)
		}
	}

	f := strings.NewReader(fileContent)
	args := lxd.ContainerFileArgs{
		Mode:      int(mode.Perm()),
		UID:       int64(uid),
		GID:       int64(gid),
		Type:      "file",
		Content:   f,
		WriteMode: "overwrite",
	}
	if err := server.CreateContainerFile(container, fileTarget, args); err != nil {
		return fmt.Errorf("Could not upload file %s: %s", fileTarget, err)
	}

	log.Printf("[DEBUG] Successfully uploaded file %s", fileTarget)

	return nil
}

func resourceLxdContainerDeleteFile(server lxd.ContainerServer, container string, file interface{}) error {
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

	if err := server.DeleteContainerFile(container, fileTarget); err != nil {
		return fmt.Errorf("Could not delete file %s: %s", fileTarget, err)
	}

	log.Printf("[DEBUG] Successfully deleted file %s", fileTarget)

	return nil
}

// recursiveMkdir was copied almost as-is from github.com/lxc/lxd/lxc/file.go
func recursiveMkdir(d lxd.ContainerServer, container string, p string, mode os.FileMode, uid int64, gid int64) error {
	/* special case, every container has a /, we don't need to do anything */
	if p == "/" {
		return nil
	}

	// Remove trailing "/" e.g. /A/B/C/. Otherwise we will end up with an
	// empty array entry "" which will confuse the Mkdir() loop below.
	pclean := filepath.Clean(p)
	parts := strings.Split(pclean, "/")
	i := len(parts)

	for ; i >= 1; i-- {
		cur := filepath.Join(parts[:i]...)
		_, resp, err := d.GetContainerFile(container, cur)
		if err != nil {
			continue
		}

		if resp.Type != "directory" {
			return fmt.Errorf("%s is not a directory", cur)
		}

		i++
		break
	}

	for ; i <= len(parts); i++ {
		cur := filepath.Join(parts[:i]...)
		if cur == "" {
			continue
		}

		args := lxd.ContainerFileArgs{
			UID:  uid,
			GID:  gid,
			Mode: int(mode.Perm()),
			Type: "directory",
		}

		err := d.CreateContainerFile(container, cur, args)
		if err != nil {
			return err
		}
	}

	return nil
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
