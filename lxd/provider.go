package lxd

import (
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
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
	Config          *lxd_config.Config
	RefreshInterval time.Duration

	acceptRemoteCertificate bool
	clientMap               map[string]lxd.Server
	remoteSchemas           map[string]interface{}
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
	remoteSchemas := make(map[string]interface{})

	// Load remotes from LXC config
	//
	// This will not error if the `config_dir`` is not set, or LXC `config.yml`
	// does not exist. The config reader will initialise the default config
	// in this case, which includes the well-known public remotes and local
	// unix socket remote.
	configDir := d.Get("config_dir").(string)
	configPath := os.ExpandEnv(path.Join(configDir, "config.yml"))
	if conf, err := lxd_config.LoadConfig(configPath); err != nil {
		config = &lxd_config.DefaultConfig
		// set configDir, otherwise auto generate certs
		// will end up in the current working directory
		config.ConfigDir = configDir
	} else {
		config = conf
	}

	refreshInterval := d.Get("refresh_interval").(string)
	if refreshInterval == "" {
		refreshInterval = "10s"
	}
	refreshIntervalParsed, err := time.ParseDuration(refreshInterval)
	if err != nil {
		return nil, err
	}

	acceptRemoteCertificate := false
	if v, ok := d.Get("accept_remote_certificate").(bool); ok && v {
		acceptRemoteCertificate = true
	}

	lxdProv := lxdProvider{
		Config:                  config,
		RefreshInterval:         refreshIntervalParsed,
		acceptRemoteCertificate: acceptRemoteCertificate,
	}

	// Validate the client certificates or try to generate them.
	generateCertificates := d.Get("generate_client_certificates").(bool)
	if generateCertificates {
		if err := config.GenerateClientCertificate(); err != nil {
			return nil, err
		}
	}

	log.Printf("[DEBUG] LXD Config: %#v", config)

	// Create remote from Environment variables (if defined).
	// This emulates the following Terraform config,
	// but with environment variables:
	//
	// lxd_remote {
	//   name    = LXD_REMOTE
	//   address = LXD_ADDR
	//   ...
	// }
	envRemote := map[string]interface{}{
		"name":     os.Getenv("LXD_REMOTE"),
		"address":  os.Getenv("LXD_ADDR"),
		"port":     os.Getenv("LXD_PORT"),
		"password": os.Getenv("LXD_PASSWORD"),
		"scheme":   os.Getenv("LXD_SCHEME"),
		"default":  true,
	}

	// Build an LXD client from the environment-driven remote.
	// LXD_REMOTE must be set, or we ignore all the rest of the env vars.
	if envRemote["name"] != "" {
		name := envRemote["name"].(string)
		remoteSchemas[name] = envRemote
		err = lxdProv.providerConfigureClient(envRemote)
		if err != nil {
			return nil, fmt.Errorf("Unable to create client for remote [%s]: %s",
				name, err)
		}
	}

	// Loop over LXD Remotes defined in provider and initialise.
	for _, rem := range d.Get("lxd_remote").([]interface{}) {
		lxdRemote := rem.(map[string]interface{})
		name := lxdRemote["name"].(string)
		remoteSchemas[name] = rem

		err := lxdProv.providerConfigureClient(lxdRemote)
		if err != nil {
			return nil, fmt.Errorf("Unable to create client for remote [%s]: %s",
				lxdRemote["name"].(string), err)
		}
	}

	lxdProv.remoteSchemas = remoteSchemas

	log.Printf("[DEBUG] LXD Provider: %#v", lxdProv)

	return &lxdProv, nil
}

// providerConfigureClient will create an LXD client for a given remote.
// The client is then stored in the p.Config collection of clients.
func (p *lxdProvider) providerConfigureClient(lxdRemote map[string]interface{}) error {
	name := lxdRemote["name"].(string)
	port := lxdRemote["port"].(string)
	scheme := lxdRemote["scheme"].(string)
	password := lxdRemote["password"].(string)

	if addr, ok := lxdRemote["address"]; ok {
		daemonAddr := ""
		switch scheme {
		case "unix", "":
			daemonAddr = fmt.Sprintf("unix:%s", addr)
		case "https":
			daemonAddr = fmt.Sprintf("https://%s:%s", addr, port)
		}

		p.Config.Remotes[name] = lxd_config.Remote{Addr: daemonAddr}

		if lxdRemote["default"].(bool) {
			p.Config.DefaultRemote = lxdRemote["name"].(string)
		}

		if scheme == "https" {
			rclient, err := p.Config.GetContainerServer(name)

			// Validate the server certificate or try to add the remote server.
			serverCertf := p.Config.ServerCertPath(name)
			if !shared.PathExists(serverCertf) {
				// Check if PKI is in use by validating the client.
				if err := validateClient(rclient); err != nil {
					// PKI probably isn't in use. Try to add the remote certificate.
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

			// Finally, make sure the client is authenticated.
			// A new client must be created, or there will be a certificate error.
			_, err = p.initClient(name)
			if err != nil {
				return err
			}
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

// InitClient creates and returns an LXD client for the named remote
// The created client is stored for later use
func (p *lxdProvider) initClient(remote string) (lxd.Server, error) {
	var client lxd.Server
	var err error

	if p.Config.Remotes[remote].Protocol == "simplestreams" {
		client, err = p.Config.GetImageServer(remote)
	} else {
		client, err = p.Config.GetContainerServer(remote)
	}
	if err != nil {
		return nil, err
	}

	if p.clientMap == nil {
		p.clientMap = make(map[string]lxd.Server)
	}

	p.clientMap[remote] = client
	return client, nil
}

// GetContainerServer returns a client for the named remote
// It returns an error if the remote is not a ContainerServer
func (p *lxdProvider) GetContainerServer(remote string) (lxd.ContainerServer, error) {
	s, err := p.GetServer(remote)
	if err != nil {
		return nil, err
	}
	ci, err := s.GetConnectionInfo()
	if ci.Protocol == "lxd" {
		return s.(lxd.ContainerServer), nil
	}

	return nil, fmt.Errorf("remote (%s / %s) is not a ContainerServer", remote, ci.Protocol)
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
	return nil, fmt.Errorf("remote (%s / %s / %s) is not an ImageServer", remote, ci.Addresses[0], ci.Protocol)
}

// GetServer returns an client for the named remote
// The returned client could be for an ImageServer or ContainerServer
func (p *lxdProvider) GetServer(remote string) (lxd.Server, error) {
	if remote == "" {
		remote = p.Config.DefaultRemote
	}

	if client, ok := p.clientMap[remote]; ok {
		return client, nil
	}

	return p.initClient(remote)
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

// selectRemoteSchema is a convenience method tha returns the schema of the
// remote set in the LXD resource or the default remote configured on
// the Provider.
func (p *lxdProvider) selectRemoteSchema(remote string) (map[string]interface{}, error) {
	if remote != "" {
		return p.remoteSchemas[remote].(map[string]interface{}), nil
	} else {
		for _, v := range p.remoteSchemas {
			remoteSchemaData := v.(map[string]interface{})
			if remoteSchemaData["default"] == true {
				return remoteSchemaData, nil
			}
		}
	}

	return nil, fmt.Errorf("unable to find appropriate remote schema")
}

func validateClient(client lxd.ContainerServer) error {
	if client == nil {
		return errors.New("client is nil")
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
