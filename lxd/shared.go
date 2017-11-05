package lxd

import (
	"fmt"
	"log"
	"strings"
)

// Complex resource ID types
type volumeID struct {
	pool    string
	name    string
	volType string
}

func (v volumeID) String() string {
	return fmt.Sprintf("%s/%s/%s", v.pool, v.name, v.volType)
}

func NewVolumeID(pool, name, volType string) volumeID {
	return volumeID{pool: pool, name: name, volType: volType}
}

func NewVolumeIDFromResourceID(id string) volumeID {
	pieces := strings.SplitN(id, "/", 3)
	return volumeID{pieces[0], pieces[1], pieces[2]}
}

type volumeAttachmentID struct {
	pool         string
	volumeName   string
	attachedName string
}

func (v volumeAttachmentID) String() string {
	return fmt.Sprintf("%s/%s/%s", v.pool, v.volumeName, v.attachedName)
}

func NewVolumeAttachmentID(pool, volumeName, attachedName string) volumeAttachmentID {
	return volumeAttachmentID{
		pool:         pool,
		volumeName:   volumeName,
		attachedName: attachedName,
	}
}

func NewVolumeAttachmentIdFromResourceId(id string) volumeAttachmentID {
	pieces := strings.SplitN(id, "/", 3)
	log.Printf("[DEBUG] pieces: %#v", pieces)
	return volumeAttachmentID{pieces[0], pieces[1], pieces[2]}
}

// Helper functions
func resourceLxdConfigMap(c interface{}) map[string]string {
	config := make(map[string]string)
	if v, ok := c.(map[string]interface{}); ok {
		for key, val := range v {
			config[key] = val.(string)
		}
	}

	log.Printf("[DEBUG] LXD Configuration Map: %#v", config)

	return config
}

// resourceLxdConfigMapAppend appends a map of configuration values
// to an existing map. All appended config values are prefixed
// with the config namespace.
func resourceLxdConfigMapAppend(config map[string]string, append interface{}, namespace string) map[string]string {
	if config == nil {
		panic("config is nil")
	}

	if string(namespace[len(namespace)-1]) != "." {
		namespace += "."
	}

	if v, ok := append.(map[string]interface{}); ok {
		for key, val := range v {
			config[namespace+key] = val.(string)
		}
	} else {
		panic("append map is not of type map[string]string")
	}

	log.Printf("[DEBUG] LXD Configuration Map: %#v", config)

	return config
}

func resourceLxdDevices(d interface{}) map[string]map[string]string {
	devices := make(map[string]map[string]string)
	for _, v := range d.([]interface{}) {
		device := make(map[string]string)
		d := v.(map[string]interface{})
		deviceName := d["name"].(string)
		deviceType := d["type"].(string)
		deviceProperties := d["properties"].(map[string]interface{})
		device["type"] = deviceType
		for key, val := range deviceProperties {
			device[key] = val.(string)
		}

		devices[deviceName] = device
	}

	log.Printf("[DEBUG] LXD Devices: %#v", devices)

	return devices
}

func resourceLxdValidateDeviceType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	validTypes := []string{"none", "disk", "nic", "unix-char", "unix-block", "usb", "gpu"}
	valid := false

	for _, v := range validTypes {
		if value == v {
			valid = true
		}
	}

	if !valid {
		errors = append(errors, fmt.Errorf("Device must have a type of: %v", validTypes))
	}

	return
}
