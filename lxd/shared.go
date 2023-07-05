package lxd

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	lxd "github.com/lxc/lxd/client"
	"github.com/mitchellh/go-homedir"
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

func newVolumeID(pool, name, volType string) volumeID {
	return volumeID{pool: pool, name: name, volType: volType}
}

func newVolumeIDFromResourceID(id string) volumeID {
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

func newVolumeAttachmentID(pool, volumeName, attachedName string) volumeAttachmentID {
	return volumeAttachmentID{
		pool:         pool,
		volumeName:   volumeName,
		attachedName: attachedName,
	}
}

func newVolumeAttachmentIDFromResourceID(id string) volumeAttachmentID {
	pieces := strings.SplitN(id, "/", 3)
	log.Printf("[DEBUG] pieces: %#v", pieces)
	return volumeAttachmentID{pieces[0], pieces[1], pieces[2]}
}

type File struct {
	RemoteName        string
	ContainerName     string
	TargetFile        string
	Content           string
	Source            string
	UID               int
	GID               int
	Mode              string
	CreateDirectories bool
	Append            bool
}

func (f File) String() string {
	return fmt.Sprintf("%s:%s%s", f.RemoteName, f.ContainerName, f.TargetFile)
}

func newFileIDFromResourceID(id string) (string, string) {
	pieces := strings.SplitN(id, "/", 2)
	return pieces[0], fmt.Sprintf("/%s", pieces[1])
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

	rawDevices := d.(*schema.Set).List()
	for _, v := range rawDevices {
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
	validTypes := []string{
		"none", "disk", "nic", "unix-char", "unix-block", "usb", "gpu", "infiniband", "proxy",
	}

	if v == nil {
		return
	}

	value := v.(string)
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

func resourceLxdValidateInstanceType(v interface{}, k string) (ws []string, errors []error) {
	validTypes := []string{"container", "virtual-machine"}

	if v == nil {
		return
	}

	value := v.(string)
	valid := false

	for _, v := range validTypes {
		if value == v {
			valid = true
		}
	}

	if !valid {
		errors = append(errors, fmt.Errorf("Instance must have a type of: %v", validTypes))
	}

	return
}

func resourceLxdValidateNetworkType(v interface{}, k string) (ws []string, errors []error) {
	validTypes := []string{"bridge", "macvlan", "sriov", "ovn", "physical"}

	if v == nil {
		return
	}

	value := v.(string)
	valid := false

	for _, v := range validTypes {
		if value == v {
			valid = true
		}
	}

	if !valid {
		errors = append(errors, fmt.Errorf("Instance must have a type of: %v", validTypes))
	}

	return
}

// containerUploadFile will upload a file to a container.
func containerUploadFile(server lxd.ContainerServer, container string, file File) error {
	if file.Content != "" && file.Source != "" {
		return fmt.Errorf("only one of content or source can be specified")
	}

	targetFile, err := filepath.Abs(file.TargetFile)
	if err != nil {
		return fmt.Errorf("Could not determine destination target %s", targetFile)
	}

	targetIsDir := strings.HasSuffix(targetFile, "/")
	if targetIsDir {
		return fmt.Errorf("Target must be an absolute path with filename")
	}

	mode, err := strconv.ParseUint(file.Mode, 8, 32)
	if err != nil {
		return fmt.Errorf("Could not parse mode: %d", mode)
	}
	fileMode := os.FileMode(mode)

	// Build the file creation request, without the content.
	uid := int64(file.UID)
	gid := int64(file.GID)
	args := lxd.ContainerFileArgs{
		Mode: int(mode),
		UID:  int64(uid),
		GID:  int64(gid),
		Type: "file",
	}

	if file.Append {
		args.WriteMode = "append"
	} else {
		args.WriteMode = "overwrite"
	}

	// If content was specified, read the string.
	if file.Content != "" {
		args.Content = strings.NewReader(file.Content)
	}

	// If a source was specified, read the contents of the source file.
	if file.Source != "" {
		path, err := homedir.Expand(file.Source)
		if err != nil {
			return fmt.Errorf("unable to determine source file path: %s", err)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("unable to read source file: %s", err)
		}
		defer f.Close()

		args.Content = f
	}

	log.Printf("[DEBUG] Attempting to upload file to %s with uid %d, gid %d, and mode %s",
		targetFile, uid, gid, fmt.Sprintf("%04o", mode))

	if file.CreateDirectories {
		err := recursiveMkdir(server, container, path.Dir(targetFile), fileMode, uid, gid)
		if err != nil {
			return fmt.Errorf("Could not upload file %s: %s", targetFile, err)
		}
	}

	if err := server.CreateContainerFile(container, targetFile, args); err != nil {
		return fmt.Errorf("Could not upload file %s: %s", targetFile, err)
	}

	log.Printf("[DEBUG] Successfully uploaded file %s", targetFile)

	return nil
}

// containerDeleteFile will delete a file on a container.
func containerDeleteFile(server lxd.ContainerServer, container string, targetFile string) error {
	targetFile, err := filepath.Abs(targetFile)
	if err != nil {
		return fmt.Errorf("Could not sanitize destination target %s", targetFile)
	}

	targetIsDir := strings.HasSuffix(targetFile, "/")
	if targetIsDir {
		return fmt.Errorf("Target must be an absolute path with filename")
	}

	log.Printf("[DEBUG] Attempting to delete file %s", targetFile)

	if err := server.DeleteContainerFile(container, targetFile); err != nil {
		return fmt.Errorf("Could not delete file %s: %s", targetFile, err)
	}

	log.Printf("[DEBUG] Successfully deleted file %s", targetFile)

	return nil
}

// recursiveMkdir was copied almost as-is from github.com/canonical/lxd/blob/main/lxc/file.go
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
