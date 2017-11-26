package lxd

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	lxd "github.com/lxc/lxd/client"
)

func resourceLxdVolumeProfileAttach() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdVolumeProfileAttachCreate,
		Delete: resourceLxdVolumeProfileAttachDelete,
		Exists: resourceLxdVolumeProfileAttachExists,
		Read:   resourceLxdVolumeProfileAttachRead,

		Schema: map[string]*schema.Schema{
			"pool": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"volume_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"profile_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"path": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"device_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
			},

			"remote": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}

func resourceLxdVolumeProfileAttachCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetContainerServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	pool := d.Get("pool").(string)
	volumeName := d.Get("volume_name").(string)
	profileName := d.Get("profile_name").(string)
	devPath := d.Get("path").(string)

	var devName string
	if v, ok := d.Get("device_name").(string); ok && v != "" {
		devName = v
	} else {
		devName = volumeName
	}

	// Make sure the volume exists
	if _, _, err := server.GetStoragePoolVolume(pool, "custom", volumeName); err != nil {
		return fmt.Errorf("Volume does not exist or is not of type custom")
	}

	log.Printf("Attempting to attach volume %s to profile %s", volumeName, profileName)

	props := map[string]string{
		"pool":   pool,
		"path":   devPath,
		"source": volumeName,
		"type":   "disk",
	}

	profile, etag, err := server.GetProfile(profileName)
	if err != nil {
		return err
	}
	profile.Devices[devName] = props

	err = server.UpdateProfile(profileName, profile.Writable(), etag)
	if err != nil {
		return fmt.Errorf("Error attaching volume: %s", err)
	}

	v := newVolumeAttachmentID(pool, volumeName, profileName)
	log.Printf("[DEBUG] volume attachment id: %s", v.String())
	d.SetId(v.String())

	return resourceLxdVolumeProfileAttachRead(d, meta)
}

func resourceLxdVolumeProfileAttachRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetContainerServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	v := newVolumeAttachmentIDFromResourceID(d.Id())

	deviceName, deviceInfo, err := resourceLxdVolumeProfileAttachedVolume(server, v)
	if err != nil {
		return err
	}

	d.Set("pool", v.pool)
	d.Set("volume_name", v.volumeName)
	d.Set("profile_name", v.attachedName)
	d.Set("device_name", deviceName)
	d.Set("path", deviceInfo["path"])

	return nil
}

func resourceLxdVolumeProfileAttachDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetContainerServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	deviceName := d.Get("device_name").(string)

	exists, err := resourceLxdVolumeProfileAttachExists(d, meta)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("The specified volume does not exist")
	}

	profile, etag, err := server.GetProfile(d.Get("profile_name").(string))
	if err != nil {
		return err
	}
	if _, ok := profile.Devices[deviceName]; !ok {
		// Device not attached to profile
		return nil
	}
	delete(profile.Devices, deviceName)

	err = server.UpdateProfile(profile.Name, profile.Writable(), etag)
	if err != nil {
		return fmt.Errorf("Unable to detach volume: %s", err)
	}

	return nil
}

func resourceLxdVolumeProfileAttachExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetContainerServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	v := newVolumeAttachmentIDFromResourceID(d.Id())
	exists = false

	_, _, err = resourceLxdVolumeProfileAttachedVolume(server, v)
	if err != nil {
		return
	}

	exists = true
	return
}

func resourceLxdVolumeProfileAttachedVolume(
	server lxd.ContainerServer, v volumeAttachmentID) (string, map[string]string, error) {
	var deviceName string
	var deviceInfo map[string]string

	profile, _, err := server.GetProfile(v.attachedName)
	if err != nil {
		return deviceName, deviceInfo, err
	}
	log.Printf("[DEBUG] Profile devices: %#v", profile.Devices)

	for n, d := range profile.Devices {
		if d["type"] == "disk" && d["pool"] == v.pool && d["source"] == v.volumeName {
			if deviceName != "" {
				return deviceName, deviceInfo, fmt.Errorf("Multiple matching volumes were found: %s", deviceName)
			}

			deviceName = n
			deviceInfo = d
		}
	}

	if deviceName == "" {
		msg := fmt.Errorf("Unable to determine device name for volume %s on profile %s", v.volumeName, v.attachedName)
		return deviceName, deviceInfo, msg
	}

	return deviceName, deviceInfo, nil
}
