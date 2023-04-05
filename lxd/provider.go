package lxd

import (
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	lxd "github.com/lxc/lxd/client"
	lxd_config "github.com/lxc/lxd/lxc/config"
	"github.com/lxc/lxd/shared"
	lxd_api "github.com/lxc/lxd/shared/api"
)

// A global mutex
var mutex sync.RWMutex

// lxdProvider contains the Provider configuration and initialized remote clients.
type lxdProvider struct {
	// terraformLXDConfigMap is a map of LXD remotes
	// which the user has defined in the Terraform schema/configuration.
	terraformLXDConfigMap map[string]terraformLXDConfig

	// LXDConfig is the converted form of terraformLXDConfig
	// in LXD's native data structure. This is lazy-loaded / created
	// only when a connection to an LXD remote/server happens.
	// https://github.com/lxc/lxd/blob/master/lxc/config/config.go
	LXDConfig *lxd_config.Config

	// lxdClientMap is a map of LXD client connections to LXD
	// remote servers. These are lazy-loaded / created only when
	// a connection to an LXD remote/server happens.
	//
	// While a client can also be retrieved from LXDConfig, this
	// map serves an additional purpose of ensuring Terraform has
	// successfully connected and authenticated to each defined
	// LXD server/remote.
	lxdClientMap map[string]lxd.Server

	// acceptRemoteCertificates toggles if an LXD remote SSL
	// certificate should be accepted.
	acceptRemoteCertificate bool

	// RefreshInterval is a custom interval for communicating
	// with remote LXD servers.
	RefreshInterval time.Duration

	// This is a mutex used to handle concurrent reads/writes.
	sync.RWMutex
}

// terraformLXDConfig represents LXD remote/server data
// as defined in a user's Terraform schema/configuration.
type terraformLXDConfig struct {
	name         string
	address      string
	port         string
	password     string
	scheme       string
	isDefault    bool
	bootstrapped bool
}

// Provider returns a terraform.ResourceProvider
func Provider() terraform.ResourceProvider {
	// The provider definition
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			// I'd prefer to call this 'remote', however that was already used in the past
			// to set the name of the root level LXD remote in the provider
			// After an deprecation cycle we could rename this to 'remote'
			"lxd_remote": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions["lxd_remote_address"],
							Default:     "",
						},

						"default": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: descriptions["lxd_remote_default"],
						},

						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions["lxd_remote_name"],
						},

						"password": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: descriptions["lxd_remote_password"],
						},

						"port": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions["lxd_remote_port"],
							Default:     "8443",
						},

						"scheme": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  descriptions["lxd_remote_scheme"],
							ValidateFunc: validateLxdRemoteScheme,
							Default:      "https",
						},
					},
				},
			},

			"address": {
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `lxd_remote.address` instead.",
			},

			"scheme": {
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `lxd_remote.scheme` instead.",
			},

			"port": {
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `lxd_remote.port` instead.",
			},

			"remote": {
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `lxd_remote.name` instead.",
			},

			"remote_password": {
				Type:      schema.TypeString,
				Sensitive: true,
				Optional:  true,
				Removed:   "Use `lxd_remote.password` instead.",
			},

			"config_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_config_dir"],
				DefaultFunc: func() (interface{}, error) {
					return os.ExpandEnv("$HOME/.config/lxc"), nil
				},
			},

			"generate_client_certificates": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions["lxd_generate_client_certs"],
				DefaultFunc: schema.EnvDefaultFunc("LXD_GENERATE_CLIENT_CERTS", ""),
			},

			"accept_remote_certificate": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions["lxd_accept_remote_certificate"],
				DefaultFunc: schema.EnvDefaultFunc("LXD_ACCEPT_SERVER_CERTIFICATE", ""),
			},

			"refresh_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_refresh_interval"],
				Default:     "10s",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_project"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"lxd_cached_image":            resourceLxdCachedImage(),
			"lxd_publish_image":           resourceLxdPublishImage(),
			"lxd_container":               resourceLxdContainer(),
			"lxd_container_file":          resourceLxdContainerFile(),
			"lxd_network":                 resourceLxdNetwork(),
			"lxd_profile":                 resourceLxdProfile(),
			"lxd_project":                 resourceLxdProject(),
			"lxd_snapshot":                resourceLxdSnapshot(),
			"lxd_storage_pool":            resourceLxdStoragePool(),
			"lxd_volume":                  resourceLxdVolume(),
			"lxd_volume_container_attach": resourceLxdVolumeContainerAttach(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"lxd_accept_remote_certificate":    "Accept the server certificate",
		"lxd_config_dir":                   "The directory to look for existing LXD configuration. default = $HOME/.config/lxc",
		"lxd_generate_client_certificates": "Automatically generate the LXD client certificates if they don't exist.",
		"lxd_refresh_interval":             "How often to poll during state changes (default 10s)",
		"lxd_remote_address":               "The FQDN or IP where the LXD daemon can be contacted. default = empty (read from lxc config)",
		"lxd_remote_scheme":                "unix or https. default = unix",
		"lxd_remote_port":                  "Port LXD Daemon API is listening on. default = 8443.",
		"lxd_remote_name":                  "Name of the LXD remote. Required when lxd_scheme set to https, to enable locating server certificate.",
		"lxd_remote_password":              "The password for the remote.",
		"lxd_project":                      "The project where project-scoped resources will be created. Can be overridden in individual resources. default = default",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	var config *lxd_config.Config

	// If a configDir was specified, create a full configPath to the
	// config.yml file and try to load it.
	//
	// If there's an error loading config.yml, DefaultConfig will still
	// be used.
	configDir := d.Get("config_dir").(string)
	configPath := os.ExpandEnv(path.Join(configDir, "config.yml"))
	if v, err := lxd_config.LoadConfig(configPath); err == nil {
		config = v
	}

	if config == nil {
		config = &lxd_config.DefaultConfig
		config.ConfigDir = configDir
	}

	log.Printf("[DEBUG] LXD Config: %#v", config)

	// Determine if a custom refresh_interval was used.
	// If it wasn't, default to 10 seconds.
	refreshInterval := d.Get("refresh_interval").(string)
	if refreshInterval == "" {
		refreshInterval = "10s"
	}
	refreshIntervalParsed, err := time.ParseDuration(refreshInterval)
	if err != nil {
		return nil, err
	}

	// Determine if the LXD remote's SSL certificates should be
	// accepted. If this is set to false and if the remote's
	// certificates haven't already been accepted, the user will
	// need to accept the certificates out of band of Terraform.
	acceptRemoteCertificate := false
	if v, ok := d.Get("accept_remote_certificate").(bool); ok && v {
		acceptRemoteCertificate = true
	}

	// Determine if the client LXD (ie: the workstation running Terraform)
	// should generate client certificates if they don't already exist.
	if v, ok := d.Get("generate_client_certificates").(bool); ok && v {
		if err := config.GenerateClientCertificate(); err != nil {
			return nil, err
		}
	}

	if v, ok := d.Get("project").(string); ok && v != "" {
		config.ProjectOverride = v
	}

	// Create an lxdProvider struct.
	// This struct is used to store information about this Terraform
	// provider's configuration for reference throughout the lifecycle.
	lxdProv := lxdProvider{
		LXDConfig:               config,
		RefreshInterval:         refreshIntervalParsed,
		acceptRemoteCertificate: acceptRemoteCertificate,
		lxdClientMap:            make(map[string]lxd.Server),
		terraformLXDConfigMap:   make(map[string]terraformLXDConfig),
	}

	// Create remote from Environment variables (if defined).
	// This emulates the following Terraform config,
	// but with environment variables:
	//
	// lxd_remote {
	//   name    = LXD_REMOTE
	//   address = LXD_ADDR
	//   ...
	// }
	envRemote := terraformLXDConfig{
		name:     os.Getenv("LXD_REMOTE"),
		address:  os.Getenv("LXD_ADDR"),
		port:     os.Getenv("LXD_PORT"),
		password: os.Getenv("LXD_PASSWORD"),
		scheme:   os.Getenv("LXD_SCHEME"),
	}

	// Build an LXD client from the environment-driven remote.
	// This will be the default remote unless overridden by an
	// explicitly defined remote in the Terraform configuration.
	if envRemote.name != "" {
		lxdProv.setTerraformLXDConfig(envRemote.name, envRemote)
		lxdProv.LXDConfig.DefaultRemote = envRemote.name
	}

	// Loop over LXD Remotes defined in the schema and create
	// an lxdRemoteConfig for each one.
	//
	// This does not yet connect to any of the defined remotes,
	// it only stores the configuration information until it is
	// necessary to connect to the remote.
	//
	// This lazy loading allows this LXD provider to be used
	// in Terraform configurations where the LXD remote might not
	// exist yet.
	for _, v := range d.Get("lxd_remote").([]interface{}) {
		remote := v.(map[string]interface{})
		lxdRemote := terraformLXDConfig{
			name:      remote["name"].(string),
			address:   remote["address"].(string),
			port:      remote["port"].(string),
			password:  remote["password"].(string),
			scheme:    remote["scheme"].(string),
			isDefault: remote["default"].(bool),
		}

		lxdProv.setTerraformLXDConfig(lxdRemote.name, lxdRemote)

		if lxdRemote.isDefault {
			lxdProv.LXDConfig.DefaultRemote = lxdRemote.name
		}

	}

	log.Printf("[DEBUG] LXD Provider: %#v", &lxdProv)

	// At this point, lxdProv contains information about all LXD
	// remotes defined in the schema and through environment
	// variables.
	return &lxdProv, nil
}

// createClient will create an LXD client for a given remote.
// The client is then stored in the lxdProvider.Config collection of clients.
func (p *lxdProvider) createClient(remoteName string) error {
	lxdRemote, ok := p.getTerraformLXDConfig(remoteName)
	if !ok {
		return fmt.Errorf("LXD remote [%s] is not defined", remoteName)
	}

	name := lxdRemote.name
	scheme := lxdRemote.scheme
	password := lxdRemote.password
	addr := lxdRemote.address

	if addr != "" {
		daemonAddr, err := determineDaemonAddr(lxdRemote)
		if err != nil {
			return fmt.Errorf("Unable to determine daemon address for remote [%s]: %s",
				lxdRemote.name, err)
		}

		p.setLXDRemoteConfig(name, lxd_config.Remote{Addr: daemonAddr})

		if scheme == "https" {
			// If the LXD remote's certificate does not exist on the client...
			serverCertf := p.LXDConfig.ServerCertPath(name)
			if !shared.PathExists(serverCertf) {
				// Try to obtain an early connection to the remote.
				// If it succeeds, then either the certificates between
				// the remote and the client have already been exchanged
				// or PKI is being used.
				rclient, _ := p.getLXDContainerClient(name)
				if err := validateClient(rclient); err != nil {
					// Either PKI isn't being used or certificates haven't been
					// exchanged. Try to add the remote certificate.
					if p.acceptRemoteCertificate {
						if err := p.getRemoteCertificate(name); err != nil {
							return fmt.Errorf("Could not get remote certificate: %s", err)
						}
					} else {
						return fmt.Errorf("Unable to communicate with remote. Either set " +
							"accept_remote_certificate to true or add the remote out of band " +
							"of Terraform and try again.")
					}
				}
			}

			// Set bootstrapped to true to prevent an infinite loop.
			// This is required for situations when a remote might be
			// defined in a config.yml file but the client has not yet
			// exchanged certificates with the remote.
			lxdRemote.bootstrapped = true
			p.setTerraformLXDConfig(remoteName, lxdRemote)

			// Finally, make sure the client is authenticated.
			rclient, err := p.GetInstanceServer(name)
			if err != nil {
				return err
			}

			if err := authenticateToLXDServer(rclient, password); err != nil {
				return err
			}

		}
	}

	return nil
}

// getRemoteCertificate will attempt to retrieve a remote LXD server's
// certificate and save it to the servercerts path.
func (p *lxdProvider) getRemoteCertificate(remoteName string) error {
	addr := p.getRemoteConfig(remoteName)
	certificate, err := shared.GetRemoteCertificate(addr.Addr)
	if err != nil {
		return err
	}

	serverCertDir := p.LXDConfig.ConfigPath("servercerts")
	if err := os.MkdirAll(serverCertDir, 0750); err != nil {
		return fmt.Errorf("Could not create server cert dir: %s", err)
	}

	certf := fmt.Sprintf("%s/%s.crt", serverCertDir, remoteName)
	certOut, err := os.Create(certf)
	if err != nil {
		return err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
	certOut.Close()

	return nil
}

// GetInstanceServer returns a client for the named remote.
// It returns an error if the remote is not a ContainerServer.
func (p *lxdProvider) GetInstanceServer(remoteName string) (lxd.ContainerServer, error) {
	s, err := p.GetServer(remoteName)
	if err != nil {
		return nil, err
	}

	ci, err := getLXDServerConnectionInfo(s)
	if err != nil {
		return nil, err
	}

	if ci.Protocol == "lxd" {
		return s.(lxd.ContainerServer), nil
	}

	err = fmt.Errorf("remote (%s / %s) is not a ContainerServer", remoteName, ci.Protocol)
	return nil, err
}

// GetImageServer returns a client for the named image server
// It returns an error if the named remote is not an ImageServer
func (p *lxdProvider) GetImageServer(remoteName string) (lxd.ImageServer, error) {
	s, err := p.GetServer(remoteName)
	if err != nil {
		return nil, err
	}

	ci, err := getLXDServerConnectionInfo(s)
	if err != nil {
		return nil, err
	}

	if ci.Protocol == "simplestreams" || ci.Protocol == "lxd" {
		return s.(lxd.ImageServer), nil
	}

	err = fmt.Errorf(
		"remote (%s / %s / %s) is not an ImageServer",
		remoteName, ci.Addresses[0], ci.Protocol)

	return nil, err
}

// GetServer returns a client for the named remote.
// The returned client could be for an ImageServer or ContainerServer
func (p *lxdProvider) GetServer(remoteName string) (lxd.Server, error) {
	if remoteName == "" {
		remoteName = p.LXDConfig.DefaultRemote
	}

	// Check and see if a client was already created and cached.
	if client, ok := p.getLXDClient(remoteName); ok {
		return client, nil
	}

	// If a client was not already created, create a new one.
	if v, ok := p.getTerraformLXDConfig(remoteName); ok && !v.bootstrapped {
		err := p.createClient(remoteName)
		if err != nil {
			return nil, fmt.Errorf("Unable to create client for remote [%s]: %s",
				remoteName, err)
		}
	}

	var client lxd.Server
	var err error

	remoteConfig := p.getRemoteConfig(remoteName)
	switch remoteConfig.Protocol {
	case "simplestreams":
		client, err = p.getLXDImageClient(remoteName)
	default:
		client, err = p.getLXDContainerClient(remoteName)
	}

	if err != nil {
		// If the reported error contained the path /var/lib/lxd/unix.socket,
		// it's possible that the user did not define any remotes in order to
		// use the implicit "local" remote which connects via a unix socket.
		//
		// When LXD is installed via a snap package, this path no longer works.
		// Therefore, we try to retry using the laziest method possible:
		// set the LXD_SOCKET environment variable to the snap path and see if
		// the connection works again.
		if strings.Contains(err.Error(), "/var/lib/lxd/unix.socket") {
			v := os.Getenv("LXD_SOCKET")
			os.Setenv("LXD_SOCKET", "/var/snap/lxd/common/lxd/unix.socket")
			defer os.Setenv("LXD_SOCKET", v)

			client, err = p.getLXDContainerClient(remoteName)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Add the client to the clientMap cache.
	p.setLXDClient(remoteName, client)

	return client, nil
}

// selectRemote is a convenience method that returns the 'remote' set
// in the LXD resource or the default remote configured on the Provider.
func (p *lxdProvider) selectRemote(d *schema.ResourceData) string {
	var remoteName string
	if rem, ok := d.GetOk("remote"); ok && rem != "" {
		remoteName = rem.(string)
	} else {
		remoteName = p.LXDConfig.DefaultRemote
	}
	return remoteName
}

// setLXDRemoteConfig will add/set a remote configuration in a concurrent-safe way.
func (p *lxdProvider) setLXDRemoteConfig(remoteName string, remote lxd_config.Remote) {
	p.Lock()
	defer p.Unlock()

	p.LXDConfig.Remotes[remoteName] = remote
}

// getRemoteConfig will retrieve an LXD remote configuration in a concurrent-safe way.
func (p *lxdProvider) getRemoteConfig(remoteName string) lxd_config.Remote {
	p.RLock()
	defer p.RUnlock()

	return p.LXDConfig.Remotes[remoteName]
}

// getLXDContainerClient will retrieve an LXD Container client in a conncurrent-safe way.
func (p *lxdProvider) getLXDContainerClient(remoteName string) (lxd.ContainerServer, error) {
	p.RLock()
	defer p.RUnlock()

	rclient, err := p.LXDConfig.GetInstanceServer(remoteName)
	return rclient, err
}

// getLXDImageClient will retrieve an LXD Image client in a conncurrent-safe way.
func (p *lxdProvider) getLXDImageClient(remoteName string) (lxd.ImageServer, error) {
	p.RLock()
	defer p.RUnlock()

	rclient, err := p.LXDConfig.GetImageServer(remoteName)
	return rclient, err
}

// setTerraformLXDConfig will add/set a Terraform LXD remote configuration to the
// collection of all Terraform LXD remotes in a concurrent-safe way.
func (p *lxdProvider) setTerraformLXDConfig(remoteName string, lxdRemote terraformLXDConfig) {
	p.Lock()
	defer p.Unlock()

	p.terraformLXDConfigMap[remoteName] = lxdRemote
}

// getTerraformLXDConfig will retrieve a Terraform LXD remote configuration from the
// collection of all Terraform LXD remotes in a concurrent-safe way.
func (p *lxdProvider) getTerraformLXDConfig(remoteName string) (terraformLXDConfig, bool) {
	p.RLock()
	defer p.RUnlock()

	terraformLXDConfig, ok := p.terraformLXDConfigMap[remoteName]
	return terraformLXDConfig, ok
}

// setLXDClient will add/set an LXD client to the collection of all LXD clients
// in a concurrent-safe way.
func (p *lxdProvider) setLXDClient(remoteName string, lxdClient lxd.Server) {
	p.Lock()
	defer p.Unlock()

	p.lxdClientMap[remoteName] = lxdClient
}

// getLXDClient will retrieve an LXD client from the collection of all LXD clients
// in a concurrent-safe way.
func (p *lxdProvider) getLXDClient(remoteName string) (lxd.Server, bool) {
	p.RLock()
	defer p.RUnlock()

	lxdClient, ok := p.lxdClientMap[remoteName]
	return lxdClient, ok
}

// getLXDServerConnectionInfo returns an LXD server's connection info in a
// concurrent-safe way.
func getLXDServerConnectionInfo(server lxd.Server) (*lxd.ConnectionInfo, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	ci, err := server.GetConnectionInfo()
	return ci, err
}

// validateClient makes a simple GET request to the servers API.
func validateClient(client lxd.ContainerServer) error {
	if client == nil {
		return fmt.Errorf("client is nil")
	}

	if _, _, err := client.GetServer(); err != nil {
		return err
	}

	return nil
}

// authenticateToLXDServer authenticates to a given remote LXD server.
// If successful, the LXD server becomes trusted to the LXD client,
// and vice-versa.
func authenticateToLXDServer(client lxd.ContainerServer, password string) error {
	mutex.Lock()
	defer mutex.Unlock()

	srv, _, err := client.GetServer()

	if err != nil {
		return err
	}

	if srv.Auth == "trusted" {
		return nil
	}

	req := lxd_api.CertificatesPost{
		Password: password,
	}
	req.Type = "client"

	err = client.CreateCertificate(req)
	if err != nil {
		return fmt.Errorf("Unable to authenticate with remote server: %s", err)
	}

	srv, _, err = client.GetServer()
	if err != nil {
		return err
	}

	return nil
}

// validateLxdRemoteScheme validates the `lxd_remote.scheme` configuration
// value at parse time.
func validateLxdRemoteScheme(v interface{}, k string) ([]string, []error) {
	scheme := v.(string)
	if scheme != "https" && scheme != "unix" {
		return nil, []error{fmt.Errorf("Invalid LXD Remote scheme: %s", scheme)}
	}
	return nil, nil
}

// determineDaemonAddr helps determine the daemon addr of the remote.
func determineDaemonAddr(lxdRemote terraformLXDConfig) (string, error) {
	var daemonAddr string
	if lxdRemote.address != "" {
		switch lxdRemote.scheme {
		case "unix", "":
			daemonAddr = fmt.Sprintf("unix:%s", lxdRemote.address)
		case "https":
			daemonAddr = fmt.Sprintf("https://%s:%s", lxdRemote.address, lxdRemote.port)
		}
	}

	return daemonAddr, nil
}
