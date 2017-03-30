package lxd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared/api"
	"github.com/lxc/lxd/shared/version"
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

// The following re-implements private LXC client functions
func clientURL(baseURL string, elem ...string) string {
	// Normalize the URL
	path := strings.Join(elem, "/")
	entries := []string{}
	fields := strings.Split(path, "/")
	for i, entry := range fields {
		if entry == "" && i+1 < len(fields) {
			continue
		}

		entries = append(entries, entry)
	}
	path = strings.Join(entries, "/")

	// Assemble the final URL
	uri := baseURL + "/" + path

	// Aliases may contain a trailing slash
	if strings.HasPrefix(path, "1.0/images/aliases") {
		return uri
	}

	// File paths may contain a trailing slash
	if strings.Contains(path, "?") {
		return uri
	}

	// Nothing else should contain a trailing slash
	return strings.TrimSuffix(uri, "/")
}

func clientDoUpdateMethod(client *lxd.Client, method string, base string, args interface{}, rtype api.ResponseType) (*api.Response, error) {
	uri := clientURL(client.BaseURL, version.APIVersion, base)

	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(args)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] %s %s to %s", method, buf.String(), uri)

	req, err := http.NewRequest(method, uri, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", version.UserAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}

	return lxd.HoistResponse(resp, rtype)
}
