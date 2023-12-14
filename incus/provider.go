package incus

import (
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	incus "github.com/lxc/incus/client"
	incus_api "github.com/lxc/incus/shared/api"
	incus_config "github.com/lxc/incus/shared/cliconfig"
	incus_localtls "github.com/lxc/incus/shared/tls"
	incus_util "github.com/lxc/incus/shared/util"
)

// A global mutex.
var mutex sync.RWMutex

// supportedIncusVersions defines Incus versions that are supported by the provider.
const supportedIncusVersions = ">= 0.3.0"

// incusProvider contains the Provider configuration and initialized remote clients.
type incusProvider struct {
	// terraformIncusConfigMap is a map of Incus remotes
	// which the user has defined in the Terraform schema/configuration.
	terraformIncusConfigMap map[string]terraformIncusConfig

	// IncusConfig is the converted form of terraformIncusConfig
	// in Incus's native data structure. This is lazy-loaded / created
	// only when a connection to an Incus remote/server happens.
	// https://github.com/lxc/incus/blob/main/shared/cliconfig/config.go
	IncusConfig *incus_config.Config

	// incusClientMap is a map of Incus client connections to Incus
	// remote servers. These are lazy-loaded / created only when
	// a connection to an Incus remote/server happens.
	//
	// While a client can also be retrieved from IncusConfig, this
	// map serves an additional purpose of ensuring Terraform has
	// successfully connected and authenticated to each defined
	// Incus server/remote.
	incusClientMap map[string]incus.Server

	// acceptRemoteCertificates toggles if an Incus remote SSL
	// certificate should be accepted.
	acceptRemoteCertificate bool

	// RefreshInterval is a custom interval for communicating
	// with remote Incus servers.
	RefreshInterval time.Duration

	// This is a mutex used to handle concurrent reads/writes.
	sync.RWMutex
}

// terraformIncusConfig represents Incus remote/server data
// as defined in a user's Terraform schema/configuration.
type terraformIncusConfig struct {
	name         string
	address      string
	port         string
	password     string
	scheme       string
	isDefault    bool
	bootstrapped bool
}

// Provider returns a terraform.ResourceProvider.
func Provider() *schema.Provider {
	// The provider definition
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			// I'd prefer to call this 'remote', however that was already used in the past
			// to set the name of the root level Incus remote in the provider
			// After an deprecation cycle we could rename this to 'remote'
			"incus_remote": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The FQDN or IP where the Incus daemon can be contacted. default = empty (read from lxc config).",
							Default:     "",
						},

						"default": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether the remote is the default one or not.",
						},

						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the Incus remote. Required when incus_scheme set to https, to enable locating server certificate.",
						},

						"password": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "The password for the remote.",
						},

						"port": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Port Incus Daemon API is listening on. default = 8443.",
							Default:     "8443",
						},

						"scheme": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "unix or https. default = unix.",
							ValidateFunc: validateIncusRemoteScheme,
							Default:      "unix",
						},
					},
				},
			},

			"config_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The directory to look for existing Incus configuration. default = $HOME/.config/incus.",
				DefaultFunc: func() (interface{}, error) {
					return os.ExpandEnv("$HOME/.config/incus"), nil
				},
			},

			"generate_client_certificates": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Automatically generate the Incus client certificates if they don't exist.",
				DefaultFunc: schema.EnvDefaultFunc("Incus_GENERATE_CLIENT_CERTS", "false"),
			},

			"accept_remote_certificate": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Accept the server certificate.",
				DefaultFunc: schema.EnvDefaultFunc("Incus_ACCEPT_SERVER_CERTIFICATE", "false"),
			},

			"refresh_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "How often to poll during state changes (default 10s).",
				Default:     "10s",
			},
			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The project where project-scoped resources will be created. Can be overridden in individual resources. default = default.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"incus_cached_image":        resourceIncusCachedImage(),
			"incus_publish_image":       resourceIncusPublishImage(),
			"incus_instance":            resourceIncusInstance(),
			"incus_instance_file":       resourceIncusInstanceFile(),
			"incus_network":             resourceIncusNetwork(),
			"incus_network_lb":          resourceIncusNetworkLB(),
			"incus_network_zone":        resourceIncusNetworkZone(),
			"incus_network_zone_record": resourceIncusNetworkZoneRecord(),
			"incus_profile":             resourceIncusProfile(),
			"incus_project":             resourceIncusProject(),
			"incus_snapshot":            resourceIncusSnapshot(),
			"incus_storage_pool":        resourceIncusStoragePool(),
			"incus_volume":              resourceIncusVolume(),
			"incus_volume_copy":         resourceIncusVolumeCopy(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	var config *incus_config.Config

	// If a configDir was specified, create a full configPath to the
	// config.yml file and try to load it.
	//
	// If there's an error loading config.yml, DefaultConfig will still
	// be used.
	configDir := d.Get("config_dir").(string)
	configPath := os.ExpandEnv(path.Join(configDir, "config.yml"))
	if v, err := incus_config.LoadConfig(configPath); err == nil {
		config = v
	}

	if config == nil {
		config = incus_config.DefaultConfig()
		config.ConfigDir = configDir
	}

	log.Printf("[DEBUG] Incus Config: %#v", config)

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

	// Determine if the Incus remote's SSL certificates should be
	// accepted. If this is set to false and if the remote's
	// certificates haven't already been accepted, the user will
	// need to accept the certificates out of band of Terraform.
	acceptRemoteCertificate := false
	if v, ok := d.Get("accept_remote_certificate").(bool); ok && v {
		acceptRemoteCertificate = true
	}

	// Determine if the client Incus (ie: the workstation running Terraform)
	// should generate client certificates if they don't already exist.
	if v, ok := d.Get("generate_client_certificates").(bool); ok && v {
		if err := config.GenerateClientCertificate(); err != nil {
			return nil, err
		}
	}

	if v, ok := d.Get("project").(string); ok && v != "" {
		config.ProjectOverride = v
	}

	// Create an incusProvider struct.
	// This struct is used to store information about this Terraform
	// provider's configuration for reference throughout the lifecycle.
	incusProv := incusProvider{
		IncusConfig:             config,
		RefreshInterval:         refreshIntervalParsed,
		acceptRemoteCertificate: acceptRemoteCertificate,
		incusClientMap:          make(map[string]incus.Server),
		terraformIncusConfigMap: make(map[string]terraformIncusConfig),
	}

	// Create remote from Environment variables (if defined).
	// This emulates the following Terraform config,
	// but with environment variables:
	//
	// incus_remote {
	//   name    = Incus_REMOTE
	//   address = Incus_ADDR
	//   ...
	// }
	envRemote := terraformIncusConfig{
		name:     os.Getenv("Incus_REMOTE"),
		address:  os.Getenv("Incus_ADDR"),
		port:     os.Getenv("Incus_PORT"),
		password: os.Getenv("Incus_PASSWORD"),
		scheme:   os.Getenv("Incus_SCHEME"),
	}

	// Build an Incus client from the environment-driven remote.
	// This will be the default remote unless overridden by an
	// explicitly defined remote in the Terraform configuration.
	if envRemote.name != "" {
		incusProv.setTerraformIncusConfig(envRemote.name, envRemote)
		incusProv.IncusConfig.DefaultRemote = envRemote.name
	}

	// Loop over Incus Remotes defined in the schema and create
	// an incusRemoteConfig for each one.
	//
	// This does not yet connect to any of the defined remotes,
	// it only stores the configuration information until it is
	// necessary to connect to the remote.
	//
	// This lazy loading allows this Incus provider to be used
	// in Terraform configurations where the Incus remote might not
	// exist yet.
	for _, v := range d.Get("incus_remote").([]interface{}) {
		remote := v.(map[string]interface{})
		incusRemote := terraformIncusConfig{
			name:      remote["name"].(string),
			address:   remote["address"].(string),
			port:      remote["port"].(string),
			password:  remote["password"].(string),
			scheme:    remote["scheme"].(string),
			isDefault: remote["default"].(bool),
		}

		incusProv.setTerraformIncusConfig(incusRemote.name, incusRemote)

		if incusRemote.isDefault {
			incusProv.IncusConfig.DefaultRemote = incusRemote.name
		}
	}

	// Ensure that Incus version meets the provider's version constraint.
	// Currently, only the default remote is verified.
	err = incusProv.verifyIncusVersion(incusProv.IncusConfig.DefaultRemote)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Incus Provider: %#v", &incusProv)

	// At this point, incusProv contains information about all Incus
	// remotes defined in the schema and through environment
	// variables.
	return &incusProv, nil
}

// createClient will create an Incus client for a given remote.
// The client is then stored in the incusProvider.Config collection of clients.
func (p *incusProvider) createClient(remoteName string) error {
	incusRemote, ok := p.getTerraformIncusConfig(remoteName)
	if !ok {
		return fmt.Errorf("Incus remote [%s] is not defined", remoteName)
	}

	name := incusRemote.name
	scheme := incusRemote.scheme
	password := incusRemote.password
	addr := incusRemote.address

	if addr != "" {
		daemonAddr, err := determineDaemonAddr(incusRemote)
		if err != nil {
			return fmt.Errorf("Unable to determine daemon address for remote [%s]: %s",
				incusRemote.name, err)
		}

		p.setIncusRemoteConfig(name, incus_config.Remote{Addr: daemonAddr})

		if scheme == "https" {
			p.RLock()
			// If the Incus remote's certificate does not exist on the client...
			serverCertf := p.IncusConfig.ServerCertPath(name)
			p.RUnlock()
			if !incus_util.PathExists(serverCertf) {
				// Try to obtain an early connection to the remote.
				// If it succeeds, then either the certificates between
				// the remote and the client have already been exchanged
				// or PKI is being used.
				rclient, _ := p.getIncusInstanceClient(name)
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
			incusRemote.bootstrapped = true
			p.setTerraformIncusConfig(remoteName, incusRemote)

			// Finally, make sure the client is authenticated.
			rclient, err := p.GetInstanceServer(name)
			if err != nil {
				return err
			}

			if err := authenticateToIncusServer(rclient, password); err != nil {
				return err
			}
		}
	}

	return nil
}

// getRemoteCertificate will attempt to retrieve a remote Incus server's
// certificate and save it to the servercerts path.
func (p *incusProvider) getRemoteCertificate(remoteName string) error {
	addr := p.getRemoteConfig(remoteName)
	certificate, err := incus_localtls.GetRemoteCertificate(addr.Addr, "terraform-provider-incus/2.0")
	if err != nil {
		return err
	}

	serverCertDir := p.IncusConfig.ConfigPath("servercerts")
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
// It returns an error if the remote is not a InstanceServer.
func (p *incusProvider) GetInstanceServer(remoteName string) (incus.InstanceServer, error) {
	s, err := p.GetServer(remoteName)
	if err != nil {
		return nil, err
	}

	ci, err := getIncusServerConnectionInfo(s)
	if err != nil {
		return nil, err
	}

	if ci.Protocol == "incus" {
		return s.(incus.InstanceServer), nil
	}

	err = fmt.Errorf("remote (%s / %s) is not a InstanceServer", remoteName, ci.Protocol)
	return nil, err
}

// GetImageServer returns a client for the named image server
// It returns an error if the named remote is not an ImageServer.
func (p *incusProvider) GetImageServer(remoteName string) (incus.ImageServer, error) {
	s, err := p.GetServer(remoteName)
	if err != nil {
		return nil, err
	}

	ci, err := getIncusServerConnectionInfo(s)
	if err != nil {
		return nil, err
	}

	if ci.Protocol == "simplestreams" || ci.Protocol == "incus" {
		return s.(incus.ImageServer), nil
	}

	err = fmt.Errorf(
		"remote (%s / %s / %s) is not an ImageServer",
		remoteName, ci.Addresses[0], ci.Protocol)

	return nil, err
}

// GetServer returns a client for the named remote.
// The returned client could be for an ImageServer or InstanceServer.
func (p *incusProvider) GetServer(remoteName string) (incus.Server, error) {
	if remoteName == "" {
		remoteName = p.IncusConfig.DefaultRemote
	}

	// Check and see if a client was already created and cached.
	client, ok := p.getIncusClient(remoteName)
	if ok {
		return client, nil
	}

	// If a client was not already created, create a new one.
	remote, ok := p.getTerraformIncusConfig(remoteName)
	if ok && !remote.bootstrapped {
		err := p.createClient(remoteName)
		if err != nil {
			return nil, fmt.Errorf("Unable to create client for remote [%s]: %s", remoteName, err)
		}
	}

	remoteConfig := p.getRemoteConfig(remoteName)

	// If remote address is not provided or is only set to the prefix for
	// Unix sockets (`unix://`) then determine which Incus directory
	// contains a writable unix socket.
	if remoteConfig.Addr == "" || remoteConfig.Addr == "unix://" {
		incusDir, err := determineIncusDir()
		if err != nil {
			return nil, err
		}

		_ = os.Setenv("Incus_DIR", incusDir)
	}

	var err error

	switch remoteConfig.Protocol {
	case "simplestreams":
		client, err = p.getIncusImageClient(remoteName)
	default:
		client, err = p.getIncusInstanceClient(remoteName)
	}

	if err != nil {
		return nil, err
	}

	// Add the client to the clientMap cache.
	p.setIncusClient(remoteName, client)

	return client, nil
}

// selectRemote is a convenience method that returns the 'remote' set
// in the Incus resource or the default remote configured on the Provider.
func (p *incusProvider) selectRemote(d *schema.ResourceData) string {
	var remoteName string
	if rem, ok := d.GetOk("remote"); ok && rem != "" {
		remoteName = rem.(string)
	} else {
		remoteName = p.IncusConfig.DefaultRemote
	}
	return remoteName
}

// setIncusRemoteConfig will add/set a remote configuration in a concurrent-safe way.
func (p *incusProvider) setIncusRemoteConfig(remoteName string, remote incus_config.Remote) {
	p.Lock()
	defer p.Unlock()

	p.IncusConfig.Remotes[remoteName] = remote
}

// getRemoteConfig will retrieve an Incus remote configuration in a concurrent-safe way.
func (p *incusProvider) getRemoteConfig(remoteName string) incus_config.Remote {
	p.RLock()
	defer p.RUnlock()

	return p.IncusConfig.Remotes[remoteName]
}

// getIncusInstanceClient will retrieve an Incus Instance client in a conncurrent-safe way.
func (p *incusProvider) getIncusInstanceClient(remoteName string) (incus.InstanceServer, error) {
	p.RLock()
	defer p.RUnlock()

	rclient, err := p.IncusConfig.GetInstanceServer(remoteName)
	return rclient, err
}

// getIncusImageClient will retrieve an Incus Image client in a conncurrent-safe way.
func (p *incusProvider) getIncusImageClient(remoteName string) (incus.ImageServer, error) {
	p.RLock()
	defer p.RUnlock()

	rclient, err := p.IncusConfig.GetImageServer(remoteName)
	return rclient, err
}

// setTerraformIncusConfig will add/set a Terraform Incus remote configuration to the
// collection of all Terraform Incus remotes in a concurrent-safe way.
func (p *incusProvider) setTerraformIncusConfig(remoteName string, incusRemote terraformIncusConfig) {
	p.Lock()
	defer p.Unlock()

	p.terraformIncusConfigMap[remoteName] = incusRemote
}

// getTerraformIncusConfig will retrieve a Terraform Incus remote configuration from the
// collection of all Terraform Incus remotes in a concurrent-safe way.
func (p *incusProvider) getTerraformIncusConfig(remoteName string) (terraformIncusConfig, bool) {
	p.RLock()
	defer p.RUnlock()

	terraformIncusConfig, ok := p.terraformIncusConfigMap[remoteName]
	return terraformIncusConfig, ok
}

// setIncusClient will add/set an Incus client to the collection of all Incus clients
// in a concurrent-safe way.
func (p *incusProvider) setIncusClient(remoteName string, incusClient incus.Server) {
	p.Lock()
	defer p.Unlock()

	p.incusClientMap[remoteName] = incusClient
}

// getIncusClient will retrieve an Incus client from the collection of all Incus clients
// in a concurrent-safe way.
func (p *incusProvider) getIncusClient(remoteName string) (incus.Server, bool) {
	p.RLock()
	defer p.RUnlock()

	incusClient, ok := p.incusClientMap[remoteName]
	return incusClient, ok
}

// verifyIncusVersion verifies whether the version of target Incus server matches the
// provider's required version contraint.
func (p *incusProvider) verifyIncusVersion(remoteName string) error {
	client, err := p.GetInstanceServer(remoteName)
	if err != nil {
		return err
	}

	server, _, err := client.GetServer()
	if err != nil {
		return err
	}

	serverVersion := server.Environment.ServerVersion
	ok, err := CheckVersion(serverVersion, supportedIncusVersions)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("Incus server %q with version %q does not meet the required version constraint: %q", remoteName, serverVersion, supportedIncusVersions)
	}

	return nil
}

// getIncusServerConnectionInfo returns an Incus server's connection info in a
// concurrent-safe way.
func getIncusServerConnectionInfo(server incus.Server) (*incus.ConnectionInfo, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	ci, err := server.GetConnectionInfo()
	return ci, err
}

// validateClient makes a simple GET request to the servers API.
func validateClient(client incus.InstanceServer) error {
	if client == nil {
		return fmt.Errorf("client is nil")
	}

	if _, _, err := client.GetServer(); err != nil {
		return err
	}

	return nil
}

// authenticateToIncusServer authenticates to a given remote Incus server.
// If successful, the Incus server becomes trusted to the Incus client,
// and vice-versa.
func authenticateToIncusServer(client incus.InstanceServer, token string) error {
	mutex.Lock()
	defer mutex.Unlock()

	srv, _, err := client.GetServer()

	if err != nil {
		return err
	}

	if srv.Auth == "trusted" {
		return nil
	}

	req := incus_api.CertificatesPost{
		TrustToken: token,
	}
	req.Type = "client"

	err = client.CreateCertificate(req)
	if err != nil {
		return fmt.Errorf("Unable to authenticate with remote server: %s", err)
	}

	_, _, err = client.GetServer()
	if err != nil {
		return err
	}

	return nil
}

// validateIncusRemoteScheme validates the `incus_remote.scheme` configuration
// value at parse time.
func validateIncusRemoteScheme(v interface{}, k string) ([]string, []error) {
	scheme := v.(string)
	if scheme != "https" && scheme != "unix" {
		return nil, []error{fmt.Errorf("Invalid Incus Remote scheme: %s", scheme)}
	}
	return nil, nil
}

// determineDaemonAddr helps determine the daemon addr of the remote.
func determineDaemonAddr(incusRemote terraformIncusConfig) (string, error) {
	var daemonAddr string
	if incusRemote.address != "" {
		switch incusRemote.scheme {
		case "unix", "":
			daemonAddr = fmt.Sprintf("unix:%s", incusRemote.address)
		case "https":
			daemonAddr = fmt.Sprintf("https://%s:%s", incusRemote.address, incusRemote.port)
		}
	}

	return daemonAddr, nil
}

// determineIncusDir determines which standard Incus directory contains a writable UNIX socket.
// If environment variable Incus_DIR or Incus_SOCKET is set, the function will return Incus directory
// based on the value from any of those variables.
func determineIncusDir() (string, error) {
	incusSocket, ok := os.LookupEnv("Incus_SOCKET")
	if ok {
		if IsSocketWritable(incusSocket) {
			return path.Dir(incusSocket), nil
		}

		return "", fmt.Errorf("Environment variable Incus_SOCKET points to either a non-existing or non-writable unix socket")
	}

	incusDir, ok := os.LookupEnv("Incus_DIR")
	if ok {
		socketPath := path.Join(incusDir, "unix.socket")
		if IsSocketWritable(socketPath) {
			return incusDir, nil
		}

		return "", fmt.Errorf("Environment variable Incus_DIR points to a Incus directory that does not contain a writable unix socket")
	}

	incusDirs := []string{
		"/var/lib/incus",
	}

	// Iterate over Incus directories and find a writable unix socket.
	for _, incusDir := range incusDirs {
		socketPath := path.Join(incusDir, "unix.socket")
		if IsSocketWritable(socketPath) {
			return incusDir, nil
		}
	}

	return "", fmt.Errorf("Incus socket with write permissions not found. Searched Incus directories: %v", incusDirs)
}
