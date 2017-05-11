package lxd

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
)

// LxdProvider contains the Provider configuration and initialized remote clients
type LxdProvider struct {
	Config          *lxd.Config
	RefreshInterval time.Duration

	acceptRemoteCertificate bool
	clientMap               map[string]*lxd.Client
}

// InitClient creates and returns an LXD client for the named remote
func (p *LxdProvider) initClient(remote string) (*lxd.Client, error) {
	client, err := lxd.NewClient(p.Config, remote)
	if err != nil {
		return nil, err
	}

	if p.clientMap == nil {
		p.clientMap = make(map[string]*lxd.Client)
	}

	p.clientMap[remote] = client
	return client, nil
}

// GetClient returns an LXD client for the named remote
func (p *LxdProvider) GetClient(remote string) (*lxd.Client, error) {
	if remote == "" {
		remote = p.Config.DefaultRemote
	}

	if client, ok := p.clientMap[remote]; ok {
		return client, nil
	}

	return p.initClient(remote)
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
	var config *lxd.Config

	// Load remotes from LXC config
	//
	// This will not error if the `config_dir`` is not set, or LXC `config.yml`
	// does not exist. The config reader will initialise the default config
	// in this case, which includes the well-known public remotes and local
	// unix socket remote.
	configDir := d.Get("config_dir").(string)
	configPath := os.ExpandEnv(path.Join(configDir, "config.yml"))
	if conf, err := lxd.LoadConfig(configPath); err != nil {
		return nil, fmt.Errorf("Could not read the lxc config: [%s]. Error: %s", configPath, err)
	} else {
		delete(conf.Remotes, "local")
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

	lxdProv := LxdProvider{
		Config:                  config,
		RefreshInterval:         refreshIntervalParsed,
		acceptRemoteCertificate: acceptRemoteCertificate,
	}

	// Validate the client certificates or try to generate them.
	if err := validateClientCerts(d, *config); err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] LXD Config: %#v", config)

	// Create remote from Environment variables (if defined)
	envRemote := map[string]interface{}{
		"name":     os.Getenv("LXD_REMOTE"),
		"address":  os.Getenv("LXD_ADDR"),
		"port":     os.Getenv("LXD_PORT"),
		"password": os.Getenv("LXD_PASSWORD"),
		"scheme":   os.Getenv("LXD_SCHEME"),
		"default":  true,
	}
	err = lxdProv.providerConfigureClient(envRemote)
	if err != nil {
		return nil, fmt.Errorf("Unable to create client for remote [%s]: %s", envRemote["name"].(string), err)
	}

	// Loop over LXD Remotes defined in provider and initialise
	for _, rem := range d.Get("lxd_remote").([]interface{}) {
		lxdRemote := rem.(map[string]interface{})
		err := lxdProv.providerConfigureClient(lxdRemote)
		if err != nil {
			return nil, fmt.Errorf("Unable to create client for remote [%s]: %s", lxdRemote["name"].(string), err)
		}
	}

	log.Printf("[DEBUG] LXD Provider: %#v", lxdProv)

	return &lxdProv, nil
}

func (p *LxdProvider) providerConfigureClient(lxdRemote map[string]interface{}) error {
	name := lxdRemote["name"].(string)
	scheme := lxdRemote["scheme"].(string)
	port := lxdRemote["port"].(string)
	password := lxdRemote["password"].(string)

	if addr, ok := lxdRemote["address"]; ok {
		daemonAddr := ""
		switch scheme {
		case "unix", "":
			daemonAddr = fmt.Sprintf("unix:%s", addr)
		case "https":
			daemonAddr = fmt.Sprintf("https://%s:%s", addr, port)
		}

		p.Config.Remotes[name] = lxd.RemoteConfig{Addr: daemonAddr}

		if lxdRemote["default"].(bool) {
			p.Config.DefaultRemote = lxdRemote["name"].(string)
		}

		if scheme == "https" {
			rclient, err := lxd.NewClient(p.Config, name)

			// Validate the server certificate or try to add the remote server.
			serverCertf := p.Config.ServerCertPath(name)
			if !shared.PathExists(serverCertf) {
				// Check if PKI is in use by validating a client
				if err := validateClient(rclient); err != nil {
					// PKI probably isn't in use. Try to add the remote certificate.
					if p.acceptRemoteCertificate {
						if err := getRemoteCertificate(rclient, name); err != nil {
							return fmt.Errorf("Could get remote certificate: %s", err)
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
			// rclient, err = lxd.NewClient(p.Config, name)
			rclient, err = p.initClient(name)
			if err != nil {
				return err
			}
			if err := checkClientAuth(name, password, rclient); err != nil {
				return err
			}
			// Make sure it's valid before proceeding
			if err := validateClient(rclient); err != nil {
				return err
			}

		}
	}
	return nil
}

// validateClient makes a simple GET request to the servers API
func validateClient(client *lxd.Client) error {
	if _, err := client.GetServerConfig(); err != nil {
		return err
	}
	return nil
}

// validateClientCerts checks if LXD client certificate & key already exist on disk
// if not and if 'generate_client_certificates' = true it will generate them
func validateClientCerts(d *schema.ResourceData, config lxd.Config) error {
	certf := config.ConfigPath("client.crt")
	keyf := config.ConfigPath("client.key")
	if !shared.PathExists(certf) || !shared.PathExists(keyf) {
		if v, ok := d.Get("generate_client_certificates").(bool); ok && v {
			log.Printf("[DEBUG] Attempting to generate client certificates")
			if err := shared.FindOrGenCert(certf, keyf, true); err != nil {
				return err
			}
		} else {
			err := fmt.Errorf("Certificate or key not found:\n\t%s\n\t%s\n"+
				"Either set generate_client_certificates to true or generate the "+
				"certificates out of band of Terraform and try again", certf, keyf)
			return err
		}
	}
	return nil
}

// getRemoteCertificate gets an LXD server's certificate
func getRemoteCertificate(client *lxd.Client, remote string) error {
	var certificate *x509.Certificate
	addr := client.Config.Remotes[remote]

	log.Printf("[DEBUG] Attempting to retrieve remote server certificate")
	// Setup a permissive TLS config
	tlsConfig, err := shared.GetTLSConfig("", "", "", nil)
	if err != nil {
		return err
	}

	tlsConfig.InsecureSkipVerify = true
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		Dial:            shared.RFC3493Dialer,
		Proxy:           shared.ProxyFromEnvironment,
	}

	// Connect
	httpClient := &http.Client{Transport: tr}
	resp, err := httpClient.Get(addr.Addr)
	if err != nil {
		return err
	}

	// Retrieve the certificate
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return fmt.Errorf("Unable to read remote TLS certificate")
	}

	certificate = resp.TLS.PeerCertificates[0]

	dnam := client.Config.ConfigPath("servercerts")
	if err := os.MkdirAll(dnam, 0750); err != nil {
		return fmt.Errorf("Could not create server cert dir: %s", err)
	}

	certf := fmt.Sprintf("%s/%s.crt", dnam, client.Name)
	certOut, err := os.Create(certf)
	if err != nil {
		return err
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
	certOut.Close()

	return nil
}

func checkClientAuth(remoteName, remotePassword string, client *lxd.Client) error {
	log.Printf("[DEBUG] Checking client is authenticated for remote: %s", remoteName)

	if client.AmTrusted() {
		log.Printf("[DEBUG] LXC client is trusted with %s", remoteName)
		return nil
	}

	if err := client.AddMyCertToServer(remotePassword); err != nil {
		return fmt.Errorf("Unable to authenticate with remote server: %s", err)
	}

	log.Printf("[DEBUG] Successfully authenticated with remote %s", remoteName)

	return nil
}

// selectRemote is a convenience method that returns the 'remote' set
// in the LXD resource or the default remote configured on the Provider
func (p *LxdProvider) selectRemote(d *schema.ResourceData) string {
	var remote string
	if rem, ok := d.GetOk("remote"); ok && rem != "" {
		remote = rem.(string)
	} else {
		remote = p.Config.DefaultRemote
	}
	return remote
}

// validateLxdRemoteScheme validates the `lxd_remote.scheme` configuration
// value as parse time
func validateLxdRemoteScheme(v interface{}, k string) ([]string, []error) {
	scheme := v.(string)
	if scheme != "https" && scheme != "unix" {
		return nil, []error{fmt.Errorf("Invalid LXD Remote scheme: %s", scheme)}
	}
	return nil, nil
}
