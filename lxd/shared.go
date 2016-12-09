package lxd

import (
	"fmt"
	"log"

	"github.com/lxc/lxd/shared"
)

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

func resourceLxdDevices(d interface{}) shared.Devices {
	devices := make(shared.Devices)
	for _, v := range d.([]interface{}) {
		device := make(shared.Device)
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
