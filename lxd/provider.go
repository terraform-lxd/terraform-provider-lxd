package lxd

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
)

type LxdProvider struct {
	Remote          string
	Client          *lxd.Client
	Config          *lxd.Config
	RefreshInterval time.Duration
}

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {

	// The actual provider
	return &schema.Provider{
		Schema: map[string]*schema.Schema{

			"address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_address"],
				DefaultFunc: schema.EnvDefaultFunc("LXD_ADDR", "/var/lib/lxd/unix.socket"),
			},

			"scheme": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_scheme"],
				DefaultFunc: schema.EnvDefaultFunc("LXD_SCHEME", "unix"),
			},

			"port": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_port"],
				DefaultFunc: schema.EnvDefaultFunc("LXD_PORT", "8443"),
			},

			"remote": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["lxd_remote"],
				DefaultFunc: schema.EnvDefaultFunc("LXD_REMOTE", "local"),
			},

			"remote_password": &schema.Schema{
				Type:        schema.TypeString,
				Sensitive:   true,
				Optional:    true,
				Description: descriptions["lxd_remote_password"],
				DefaultFunc: schema.EnvDefaultFunc("LXD_REMOTE_PASSWORD", ""),
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
		"lxd_address":                      "The FQDN or IP where the LXD daemon can be contacted. (default = /var/lib/lxd/unix.socket)",
		"lxd_scheme":                       "unix or https (default = unix)",
		"lxd_port":                         "Port LXD Daemon is listening on (default 8443).",
		"lxd_remote":                       "Name of the LXD remote. Required when lxd_scheme set to https, to enable locating server certificate.",
		"lxd_remote_password":              "The password for the remote.",
		"lxd_config_dir":                   "The directory to look for existing LXD configuration (default = $HOME/.config/lxc).",
		"lxd_generate_client_certificates": "Automatically generate the LXD client certificates if they don't exist.",
		"lxd_accept_remote_certificate":    "Accept the server certificate",
		"lxd_refresh_interval":             "How often to poll during state changes (default 10s)",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	remote := d.Get("remote").(string)
	scheme := d.Get("scheme").(string)

	daemon_addr := ""
	switch scheme {
	case "unix":
		daemon_addr = fmt.Sprintf("unix:%s", d.Get("address"))
	case "https":
		daemon_addr = fmt.Sprintf("https://%s:%s", d.Get("address"), d.Get("port"))
	default:
		err := fmt.Errorf("Invalid scheme: %s", scheme)
		return nil, err
	}

	// build LXD config
	config := &lxd.Config{
		ConfigDir: d.Get("config_dir").(string),
		Remotes:   make(map[string]lxd.RemoteConfig),
	}
	config.Remotes[remote] = lxd.RemoteConfig{Addr: daemon_addr}

	// Load static Public remotes
	for k, v := range lxd.DefaultRemotes {
		config.Remotes[k] = v
	}
	log.Printf("[DEBUG] LXD Config: %#v", config)

	// Build a basic client
	client, err := lxd.NewClient(config, remote)
	if err != nil {
		err := fmt.Errorf("Could not create LXD client: %s", err)
		return nil, err
	}

	if scheme == "https" {
		// Validate the client certificates or try to generate them.
		if err := validateClientCerts(d, *config); err != nil {
			return nil, err
		}

		// Validate the server certificate or try to add the remote server.
		serverCertf := config.ServerCertPath(remote)
		if !shared.PathExists(serverCertf) {
			// Check if PKI is in use by validating a client
			if err := validateClient(client); err != nil {
				// PKI probably isn't in use. Try to add the remote certificate.
				if v, ok := d.Get("accept_remote_certificate").(bool); ok && v {
					if err := getRemoteCertificate(client, remote); err != nil {
						return nil, fmt.Errorf("Could get remote certificate: %s", err)
					}
				} else {
					return nil, fmt.Errorf("Unable to communicate with remote. Either set " +
						"accept_remote_certificate to true or add the remote out of band " +
						"of Terraform and try again.")
				}
			}
		}

		// Finally, make sure the client is authenticated.
		// A new client must be created, or there will be a certificate error.
		client, err = lxd.NewClient(config, remote)
		if err != nil {
			return nil, err
		}
		if err := checkClientAuth(d, client); err != nil {
			return nil, err
		}
	}

	// Final client configuration
	log.Printf("[DEBUG] LXD Client: %#v", client)

	// Make sure it's valid before proceeding
	if err := validateClient(client); err != nil {
		return nil, err
	}

	refresh_interval := d.Get("refresh_interval").(string)
	if refresh_interval == "" {
		refresh_interval = "10s"
	}
	refresh_interval_parsed, err := time.ParseDuration(refresh_interval)
	if err != nil {
		return nil, err
	}

	lxdProv := LxdProvider{
		Remote:          remote,
		Client:          client,
		Config:          config,
		RefreshInterval: refresh_interval_parsed,
	}

	return &lxdProv, nil
}

func validateClient(client *lxd.Client) error {
	if _, err := client.GetServerConfig(); err != nil {
		return err
	}
	return nil
}

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
				"Either set generate_client_certs to true or generate the "+
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

func checkClientAuth(d *schema.ResourceData, client *lxd.Client) error {
	remote := d.Get("remote").(string)
	if client.AmTrusted() {
		log.Printf("[DEBUG] LXC client is trusted with %s", remote)
		return nil
	}

	remotePassword := d.Get("remote_password").(string)
	if err := client.AddMyCertToServer(remotePassword); err != nil {
		return fmt.Errorf("Unable to authenticate with remote server: %s", err)
	}

	log.Printf("[DEBUG] Successfully authenticated with remote %s", remote)

	return nil
}
