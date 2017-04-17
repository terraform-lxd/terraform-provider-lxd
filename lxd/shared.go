package lxd

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
	"github.com/lxc/lxd/shared/api"
	"github.com/lxc/lxd/shared/version"
)

// Complex resource ID types
type volumeId struct {
	pool    string
	name    string
	volType string
}

func (v volumeId) String() string {
	return fmt.Sprintf("%s/%s/%s", v.pool, v.name, v.volType)
}

func NewVolumeId(pool, name, volType string) volumeId {
	return volumeId{pool: pool, name: name, volType: volType}
}

func NewVolumeIdFromResourceId(id string) volumeId {
	pieces := strings.SplitN(id, "/", 3)
	return volumeId{pieces[0], pieces[1], pieces[2]}
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

// addServer adds a remote server to the local LXD configuration.
func addServer(client *lxd.Client, remote string) (*lxd.Client, error) {
	addr := client.Config.Remotes[remote]

	log.Printf("[DEBUG] Attempting to retrieve remote server certificate")
	var certificate *x509.Certificate
	var err error
	certificate, err = getRemoteCertificate(addr.Addr)
	if err != nil {
		return nil, err
	}

	dnam := client.Config.ConfigPath("servercerts")
	if err := os.MkdirAll(dnam, 0750); err != nil {
		return nil, fmt.Errorf("Could not create server cert dir")
	}

	certf := fmt.Sprintf("%s/%s.crt", dnam, client.Name)
	certOut, err := os.Create(certf)
	if err != nil {
		return nil, err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
	certOut.Close()

	// Setup a new connection, this time with the remote certificate
	client, err = lxd.NewClient(&client.Config, remote)
	if err != nil {
		return nil, err
	}

	// Validate the client before returning
	if _, err := client.GetServerConfig(); err != nil {
		return nil, err
	}

	return client, nil
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

func getRemoteCertificate(address string) (*x509.Certificate, error) {
	// Setup a permissive TLS config
	tlsConfig, err := shared.GetTLSConfig("", "", "", nil)
	if err != nil {
		return nil, err
	}

	tlsConfig.InsecureSkipVerify = true
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		Dial:            shared.RFC3493Dialer,
		Proxy:           shared.ProxyFromEnvironment,
	}

	// Connect
	client := &http.Client{Transport: tr}
	resp, err := client.Get(address)
	if err != nil {
		return nil, err
	}

	// Retrieve the certificate
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return nil, fmt.Errorf("Unable to read remote TLS certificate")
	}

	return resp.TLS.PeerCertificates[0], nil
}
