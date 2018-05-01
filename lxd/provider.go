package lxd

import (
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	lxd "github.com/lxc/lxd/client"
	lxd_config "github.com/lxc/lxd/lxc/config"
	"github.com/lxc/lxd/shared"
	lxd_api "github.com/lxc/lxd/shared/api"
)

// lxdProvider contains the Provider configuration and initialized remote clients
type lxdProvider struct {
	sync.RWMutex

	Config          *lxd_config.Config
	RefreshInterval time.Duration

	acceptRemoteCertificate bool
	clientMap               map[string]lxd.Server
	remoteConfigMap         map[string]lxdRemoteConfig
}

type lxdRemoteConfig struct {
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
			"lxd_remote": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions["lxd_remote_address"],
							Default:     "",
						},

						"default": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Description: descriptions["lxd_remote_default"],
						},

						"name": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: descriptions["lxd_remote_name"],
						},

						"password": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: descriptions["lxd_remote_password"],
						},

						"port": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: descriptions["lxd_remote_port"],
							Default:     "8443",
						},

						"scheme": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Description:  descriptions["lxd_remote_scheme"],
							ValidateFunc: validateLxdRemoteScheme,
							Default:      "https",
						},
					},
				},
			},

			"address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `lxd_remote.address` instead.",
			},

			"scheme": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `lxd_remote.scheme` instead.",
			},

			"port": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `lxd_remote.port` instead.",
			},

			"remote": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `lxd_remote.name` instead.",
			},

			"remote_password": &schema.Schema{
				Type:      schema.TypeString,
				Sensitive: true,
				Optional:  true,
				Removed:   "Use `lxd_remote.password` instead.",
			},

			"config_dir": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_config_dir"],
				DefaultFunc: func() (interface{}, error) {
					return os.ExpandEnv("$HOME/.config/lxc"), nil
				},
			},

			"generate_client_certificates": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions["lxd_generate_client_certs"],
				DefaultFunc: schema.EnvDefaultFunc("LXD_GENERATE_CLIENT_CERTS", ""),
			},

			"accept_remote_certificate": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions["lxd_accept_remote_certificate"],
				DefaultFunc: schema.EnvDefaultFunc("LXD_ACCEPT_SERVER_CERTIFICATE", ""),
			},

			"refresh_interval": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_refresh_interval"],
				Default:     "10s",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"lxd_cached_image":            resourceLxdCachedImage(),
			"lxd_container":               resourceLxdContainer(),
			"lxd_container_file":          resourceLxdContainerFile(),
			"lxd_network":                 resourceLxdNetwork(),
			"lxd_profile":                 resourceLxdProfile(),
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
	if conf, err := lxd_config.LoadConfig(configPath); err == nil {
		config = conf
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

	// Create an lxdProvider struct.
	// This struct is used to store information about this Terraform
	// provider's configuration for reference throughout the lifecycle.
	lxdProv := lxdProvider{
		Config:                  config,
		RefreshInterval:         refreshIntervalParsed,
		acceptRemoteCertificate: acceptRemoteCertificate,
		clientMap:               make(map[string]lxd.Server),
		remoteConfigMap:         make(map[string]lxdRemoteConfig),
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
	envRemote := lxdRemoteConfig{
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
		lxdProv.Lock()
		lxdProv.remoteConfigMap[envRemote.name] = envRemote
		lxdProv.Unlock()

		lxdProv.Config.DefaultRemote = envRemote.name
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
		lxdRemote := lxdRemoteConfig{
			name:      remote["name"].(string),
			address:   remote["address"].(string),
			port:      remote["port"].(string),
			password:  remote["password"].(string),
			scheme:    remote["scheme"].(string),
			isDefault: remote["default"].(bool),
		}

		lxdProv.Lock()
		lxdProv.remoteConfigMap[lxdRemote.name] = lxdRemote
		lxdProv.Unlock()

		if lxdRemote.isDefault {
			lxdProv.Config.DefaultRemote = lxdRemote.name
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
func (p *lxdProvider) createClient(remote string) error {
	lxdRemote, ok := p.remoteConfigMap[remote]
	if !ok {
		return fmt.Errorf("LXD remote [%s] is not defined", remote)
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

		p.Lock()
		p.Config.Remotes[name] = lxd_config.Remote{Addr: daemonAddr}
		p.Unlock()

		if scheme == "https" {
			rclient, _ := p.Config.GetContainerServer(name)

			// Validate the server certificate or try to add the remote server.
			serverCertf := p.Config.ServerCertPath(name)
			if !shared.PathExists(serverCertf) {
				// Try to obtain an early connection to the remote.
				// If it succeeds, then either the certificates between
				// the remote and the client have already been exchanged
				// or PKI is being used.
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
			p.Lock()
			p.remoteConfigMap[remote] = lxdRemote
			p.Unlock()

			// Finally, make sure the client is authenticated.
			// A new client must be created, or there will be a certificate error.
			rclient, err = p.GetContainerServer(name)
			if err != nil {
				return err
			}

			if err := authenticateToLXDServer(rclient, name, password); err != nil {
				return err
			}

		}
	}

	return nil
}

// getRemoteCertificate will attempt to retrieve a remote LXD server's
// certificate and save it to the servercerts path.
func (p *lxdProvider) getRemoteCertificate(remote string) error {
	addr := p.Config.Remotes[remote]
	certificate, err := shared.GetRemoteCertificate(addr.Addr)
	if err != nil {
		return err
	}

	serverCertDir := p.Config.ConfigPath("servercerts")
	if err := os.MkdirAll(serverCertDir, 0750); err != nil {
		return fmt.Errorf("Could not create server cert dir: %s", err)
	}

	certf := fmt.Sprintf("%s/%s.crt", serverCertDir, remote)
	certOut, err := os.Create(certf)
	if err != nil {
		return err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
	certOut.Close()

	return nil
}

// newClient creates and returns an LXD client for the named remote.
// The created client is stored for later use.
func (p *lxdProvider) newClient(remote string) (lxd.Server, error) {
	var client lxd.Server
	var err error

	// If the remote doesn't exist in the config, then create a client
	// for the remote.
	if v, ok := p.remoteConfigMap[remote]; ok && !v.bootstrapped {
		err = p.createClient(remote)
		if err != nil {
			return nil, fmt.Errorf("Unable to create client for remote [%s]: %s",
				remote, err)
		}
	}

	switch p.Config.Remotes[remote].Protocol {
	case "simplestreams":
		client, err = p.Config.GetImageServer(remote)
	default:
		client, err = p.Config.GetContainerServer(remote)
	}

	if err != nil {
		return nil, err
	}

	// Add the client to the clientMap cache.
	p.Lock()
	p.clientMap[remote] = client
	p.Unlock()

	return client, nil
}

// GetContainerServer returns a client for the named remote.
// It returns an error if the remote is not a ContainerServer.
func (p *lxdProvider) GetContainerServer(remote string) (lxd.ContainerServer, error) {
	s, err := p.GetServer(remote)
	if err != nil {
		return nil, err
	}

	ci, err := s.GetConnectionInfo()
	if ci.Protocol == "lxd" {
		return s.(lxd.ContainerServer), nil
	}

	err = fmt.Errorf("remote (%s / %s) is not a ContainerServer", remote, ci.Protocol)
	return nil, err
}

// GetImageServer returns a client for the named image server
// It returns an error if the named remote is not an ImageServer
func (p *lxdProvider) GetImageServer(remote string) (lxd.ImageServer, error) {
	s, err := p.GetServer(remote)
	if err != nil {
		return nil, err
	}

	ci, err := s.GetConnectionInfo()
	if ci.Protocol == "simplestreams" || ci.Protocol == "lxd" {
		return s.(lxd.ImageServer), nil
	}

	err = fmt.Errorf(
		"remote (%s / %s / %s) is not an ImageServer",
		remote, ci.Addresses[0], ci.Protocol)

	return nil, err
}

// GetServer returns a client for the named remote.
// The returned client could be for an ImageServer or ContainerServer
func (p *lxdProvider) GetServer(remote string) (lxd.Server, error) {
	if remote == "" {
		remote = p.Config.DefaultRemote
	}

	if client, ok := p.clientMap[remote]; ok {
		return client, nil
	}

	return p.newClient(remote)
}

// selectRemote is a convenience method that returns the 'remote' set
// in the LXD resource or the default remote configured on the Provider.
func (p *lxdProvider) selectRemote(d *schema.ResourceData) string {
	var remote string
	if rem, ok := d.GetOk("remote"); ok && rem != "" {
		remote = rem.(string)
	} else {
		remote = p.Config.DefaultRemote
	}
	return remote
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
func authenticateToLXDServer(client lxd.ContainerServer, remote, password string) error {
	srv, _, err := client.GetServer()
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
func determineDaemonAddr(lxdRemote lxdRemoteConfig) (string, error) {
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
