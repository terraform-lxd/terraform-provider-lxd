package lxd

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	lxd "github.com/lxc/lxd/client"
)

func resourceLxdVolumeContainerAttach() *schema.Resource {
	return &schema.Resource{
		Create: resourceLxdVolumeContainerAttachCreate,
		Delete: resourceLxdVolumeContainerAttachDelete,
		Exists: resourceLxdVolumeContainerAttachExists,
		Read:   resourceLxdVolumeContainerAttachRead,

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

			"container_name": &schema.Schema{
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
				Default:  "",
			},
		},
	}
}

func resourceLxdVolumeContainerAttachCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	pool := d.Get("pool").(string)
	volumeName := d.Get("volume_name").(string)
	containerName := d.Get("container_name").(string)

	devPath := d.Get("path").(string)
	devName := ""

	if v, ok := d.Get("device_name").(string); ok && v != "" {
		devName = v
	} else {
		devName = volumeName
	}

	// Make sure the volume exists
	if _, _, err := client.GetStoragePoolVolume(pool, "custom", volumeName); err != nil {
		return fmt.Errorf("Volume does not exist or is not of type custom")
	}

	log.Printf("Attempting to attach volume %s to container %s", volumeName, containerName)

	// props := []string{
	// 	fmt.Sprintf("pool=%s", pool),
	// 	fmt.Sprintf("path=%s", devPath),
	// 	fmt.Sprintf("source=%s", volumeName),
	// }
	props := map[string]string{
		"pool":   pool,
		"path":   devPath,
		"source": volumeName,
	}

	container, etag, err := client.GetContainer(containerName)
	if err != nil {
		return err
	}
	container.Devices[devName] = props

	op, err := client.UpdateContainer(containerName, container.Writable(), etag)
	if err != nil {
		return fmt.Errorf("Error attaching volume: %s", err)
	}

	// Wait for the volume to attach
	if err := op.Wait(); err != nil {
		return err
	}

	v := NewVolumeAttachmentId(pool, volumeName, containerName)
	log.Printf("[DEBUG] volume attachment id: %s", v.String())
	d.SetId(v.String())

	return resourceLxdVolumeContainerAttachRead(d, meta)
}

func resourceLxdVolumeContainerAttachRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	v := NewVolumeAttachmentIdFromResourceId(d.Id())

	deviceName, deviceInfo, err := resourceLxdVolumeContainerAttachedVolume(client, v)
	if err != nil {
		return err
	}

	d.Set("pool", v.pool)
	d.Set("volume_name", v.volumeName)
	d.Set("container_name", v.attachedName)
	d.Set("device_name", deviceName)
	d.Set("path", deviceInfo["path"])

	return nil
}

func resourceLxdVolumeContainerAttachDelete(d *schema.ResourceData, meta interface{}) (err error) {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return err
	}

	// v := NewVolumeAttachmentIdFromResourceId(d.Id())
	deviceName := d.Get("device_name").(string)

	exists, err := resourceLxdVolumeContainerAttachExists(d, meta)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("The specified volume does not exist")
	}

	container, etag, err := client.GetContainer(d.Get("container_name").(string))
	if err != nil {
		return err
	}
	if _, ok := container.Devices[deviceName]; !ok {
		// Device not attached to container
		return nil
	}
	delete(container.Devices, deviceName)

	op, err := client.UpdateContainer(container.Name, container.Writable(), etag)
	if err != nil {
		return fmt.Errorf("Unable to detach volume: %s", err)
	}

	if err := op.Wait(); err != nil {
		return err
	}

	return nil
}

func resourceLxdVolumeContainerAttachExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*LxdProvider)
	client, err := p.GetClient(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	v := NewVolumeAttachmentIdFromResourceId(d.Id())
	exists = false

	_, _, err = resourceLxdVolumeContainerAttachedVolume(client, v)
	if err != nil {
		return
	}

	exists = true
	return
}

func resourceLxdVolumeContainerAttachedVolume(
	client lxd.ContainerServer, v volumeAttachmentId) (string, map[string]string, error) {
	var deviceName string
	var deviceInfo map[string]string

	container, _, err := client.GetContainer(v.attachedName)
	if err != nil {
		return deviceName, deviceInfo, err
	}
	log.Printf("[DEBUG] Container devices: %#v", container.Devices)

	for n, d := range container.Devices {
		if d["type"] == "disk" && d["pool"] == v.pool && d["source"] == v.volumeName {
			if deviceName != "" {
				return deviceName, deviceInfo, fmt.Errorf("Multiple matching volumes were found: %s", deviceName)
			}

			deviceName = n
			deviceInfo = d
		}
	}

	if deviceName == "" {
		msg := fmt.Errorf("Unable to determine device name for volume %s on container %s", v.volumeName, v.attachedName)
		return deviceName, deviceInfo, msg
	}

	return deviceName, deviceInfo, nil
}
