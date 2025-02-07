package config

import (
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	lxd "github.com/canonical/lxd/client"
	lxdConfig "github.com/canonical/lxd/lxc/config"
	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/api"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

// supportedLXDVersions defines LXD versions that are supported by the provider.
const supportedLXDVersions = ">= 4.0.0"

// Options for provider config initialization.
type Options struct {
	// ConfigDir represents the directory where certificates are stored and
	// LXD configuration is searched for.
	ConfigDir string

	// AcceptServerCertificate determines whether server certificate should
	// be accepted if missing.
	AcceptServerCertificate bool

	// GenerateClientCertificates determines whether the client certificates
	// should be generated if missing.
	GenerateClientCertificates bool
}

// LxdRemote contains remote server protocol and address. In addition, it may
// contain either a trust token or password that is used for initial server
// authentication if necessary.
type LxdRemote struct {
	Protocol  string
	Address   string
	Token     string
	Password  string
	IsDefault bool

	// server represents cached client connection to the remote server.
	// This is lazy-loaded / stored when a connection is established for
	// the first time.
	server lxd.Server
}

// LxdProviderConfig contains the provider configuration and initialized
// remote servers.
type LxdProviderConfig struct {
	// config is the LXD configuration file that contains remotes
	// used by the LXD client.
	config *lxdConfig.Config

	// version of the provider.
	version string

	// acceptServerCertificates indicates that SSL certificate from an LXD
	// remote should be accepted.
	acceptServerCertificate bool

	// remotes is a map of all remotes accessible to the provider.
	remotes map[string]LxdRemote

	// mux is a lock that handle concurrent reads/writes to the LXD config.
	mux sync.RWMutex
}

// NewLxdProviderConfig initializes a new immutable provider configuration and populates
// it with remotes that are accessible to the provider throughout its lifecycle. Remotes
// are also loaded from LXD configuration file and from environment variables.
//
// Remotes have the following priority:
// Terraform configuration > environment variables > LXD configuration file.
func NewLxdProviderConfig(version string, remotes map[string]LxdRemote, options Options) (*LxdProviderConfig, error) {
	configDir := options.ConfigDir

	// Determine LXD config directory.
	if configDir == "" {
		// Determine LXD configuration directory. First check for the presence
		// of the /var/snap/lxd directory. If the directory exists, return
		// snap's config path. Otherwise return the fallback path.
		_, err := os.Stat("/var/snap/lxd")
		if err == nil || os.IsExist(err) {
			configDir = "$HOME/snap/lxd/common/config"
		} else {
			configDir = "$HOME/.config/lxc"
		}
	}

	configDir = os.ExpandEnv(configDir)
	configPath := filepath.Join(configDir, "config.yml")

	// Try to load config.yml from determined configDir. Otherwise load default config.
	config, err := lxdConfig.LoadConfig(configPath)
	if err != nil {
		config = lxdConfig.DefaultConfig()
		config.ConfigDir = configDir
	}

	p := &LxdProviderConfig{
		acceptServerCertificate: options.AcceptServerCertificate,
		version:                 version,
		config:                  config,
		remotes:                 make(map[string]LxdRemote),
	}

	// Load remotes from config to ensure we have a single source of trusth for all remotes.
	for name, remote := range config.Remotes {
		r := LxdRemote{
			Protocol: remote.Protocol,
			Address:  remote.Addr,
		}

		if r.Protocol == "" {
			r.Protocol = "lxd"
		}

		err := p.setRemote(name, r)
		if err != nil {
			return nil, fmt.Errorf("LXD configuration contains invalid remote %q: %v", name, err)
		}
	}

	// Load LXD remote from environment variables (if defined).
	// This emulates the Terraform provider "remote" config:
	//
	// remote {
	//   name     = LXD_REMOTE
	//   address  = LXD_ADDR
	//   token    = LXD_TOKEN
	//   password = LXD_PASSWORD
	// }
	envRemoteName := os.Getenv("LXD_REMOTE")
	if envRemoteName != "" {
		protocol := "lxd"

		// Resolve the LXD address from environment variable.
		address, err := DetermineLXDAddress(protocol, os.Getenv("LXD_ADDR"))
		if err != nil {
			return nil, fmt.Errorf("Failed to construct LXD address for remote %q defined through environment variables: %v", envRemoteName, err)
		}

		// Deprecated!
		envScheme := os.Getenv("LXD_SCHEME")
		if envScheme != "" {
			return nil, fmt.Errorf("Environment variable LXD_SCHEME is deprecated. Use LXD_ADDR=%q instead", address)
		}

		// Deprecated!
		envPort := os.Getenv("LXD_PORT")
		if envPort != "" {
			return nil, fmt.Errorf("Environment variable LXD_PORT is deprecated. Use LXD_ADDR=%q instead", address)
		}

		// This will be the default remote unless overridden by an
		// explicitly defined remote in the Terraform configuration.
		envRemote := LxdRemote{
			Address:   address,
			Password:  os.Getenv("LXD_PASSWORD"),
			Token:     os.Getenv("LXD_TOKEN"),
			Protocol:  protocol,
			IsDefault: true,
		}

		err = p.setRemote(envRemoteName, envRemote)
		if err != nil {
			return nil, fmt.Errorf("LXD remote %q defined through environment variables is invalid: %v", envRemoteName, err)
		}
	}

	var defaultRemotes []string

	// Load LXD remote from Terraform configuration.
	for name, remote := range remotes {
		err := p.setRemote(name, remote)
		if err != nil {
			return nil, fmt.Errorf("Invalid remote %q: %v", name, err)
		}

		if remote.IsDefault {
			defaultRemotes = append(defaultRemotes, name)
		}
	}

	// Ensure only one remote is configured as default.
	if len(defaultRemotes) > 1 {
		return nil, fmt.Errorf("Multiple remotes are configured as default: [%v]", strings.Join(defaultRemotes, ", "))
	}

	// Generate client certificates (if necessary).
	if options.GenerateClientCertificates {
		err = p.GenerateClientCertificate()
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

// InstanceServer returns a LXD InstanceServer client for the given remote.
// An error is returned if the remote is not a InstanceServer.
func (p *LxdProviderConfig) InstanceServer(remoteName string, project string, target string) (lxd.InstanceServer, error) {
	server, err := p.server(remoteName)
	if err != nil {
		return nil, err
	}

	connInfo, err := server.GetConnectionInfo()
	if err != nil {
		return nil, err
	}

	if connInfo.Protocol != "lxd" {
		return nil, fmt.Errorf("Remote %q (%s) is not an InstanceServer", remoteName, connInfo.Protocol)
	}

	instServer, ok := server.(lxd.InstanceServer)
	if !ok {
		return nil, fmt.Errorf("Remote %q is not an InstanceServer", remoteName)
	}

	instServer = instServer.UseProject(project)
	instServer = instServer.UseTarget(target)

	return instServer, nil
}

// ImageServer returns a LXD ImageServer client for the given remote.
// An error is returned if the remote is not an ImageServer.
func (p *LxdProviderConfig) ImageServer(remoteName string) (lxd.ImageServer, error) {
	server, err := p.server(remoteName)
	if err != nil {
		return nil, err
	}

	connInfo, err := server.GetConnectionInfo()
	if err != nil {
		return nil, err
	}

	if connInfo.Protocol != "simplestreams" && connInfo.Protocol != "lxd" {
		return nil, fmt.Errorf("Remote %q (%s / %s) is not an ImageServer", remoteName, connInfo.Protocol, connInfo.Addresses[0])
	}

	imageServer, ok := server.(lxd.ImageServer)
	if !ok {
		return nil, fmt.Errorf("Remote %q is not an ImageServer", remoteName)
	}

	return imageServer, nil
}

// server returns a server for the named remote. The returned server
// can be either of type ImageServer or InstanceServer.
func (p *LxdProviderConfig) server(remoteName string) (lxd.Server, error) {
	p.mux.Lock()
	defer p.mux.Unlock()

	// If remote is not set, use default remote.
	if remoteName == "" {
		remoteName = p.config.DefaultRemote
	}

	remote, ok := p.remotes[remoteName]
	if !ok {
		return nil, fmt.Errorf("Remote %q does not exist", remoteName)
	}

	// Check if there is an already initialized LXD server.
	server := remote.server
	if server != nil {
		return server, nil
	}

	// Initialize new server for the given remote.
	if remote.Protocol == "simplestreams" {
		imgServer, err := p.config.GetImageServer(remoteName)
		if err != nil {
			return nil, err
		}

		// Cache initialized server.
		remote.server = imgServer
	} else {
		// getLxdServer retrieves the instance server and the corresponding api server
		// for the given remote.
		getLxdServer := func(remoteName string) (lxd.InstanceServer, *api.Server, error) {
			instServer, err := p.config.GetInstanceServer(remoteName)
			if err != nil {
				return nil, nil, err
			}

			apiServer, _, err := instServer.GetServer()
			if err != nil {
				return nil, nil, err
			}

			return instServer, apiServer, nil
		}

		isHTTPS := strings.HasPrefix(remote.Address, "https://")

		// Try to obtain an early connection to the remote server.
		instServer, apiServer, err := getLxdServer(remoteName)
		if err != nil {
			// For non-https remotes we should be able to communicate with
			// them. It is most likely an issue in the configuration.
			if !isHTTPS {
				return nil, err
			}

			// Failure for HTTPS remote indicates that either PKI is not being
			// used or certificates have not been exchanged yet.
			certPath := p.config.ServerCertPath(remoteName)
			if shared.PathExists(certPath) {
				// Server's certificate exists locally, but we are still
				// unable to communicate with the server.
				return nil, err
			}

			// Try to accept the remote certificate.
			err := p.acceptRemoteCertificate(remoteName, remote.Token, remote.Address)
			if err != nil {
				return nil, fmt.Errorf("Failed to accept server certificate for remote %q: %v", remoteName, err)
			}

			// Retrieve the LXD server again. Now the connection must
			// succeed because we have accepted the remote certificate.
			instServer, apiServer, err = getLxdServer(remoteName)
			if err != nil {
				return nil, err
			}
		}

		// Authenticate against HTTPS remote if it is not already trusted.
		if isHTTPS && apiServer.Auth != "trusted" {
			req := api.CertificatesPost{
				Type: "client",
			}

			if remote.Token != "" {
				if instServer.HasExtension("explicit_trust_token") {
					req.TrustToken = remote.Token
				} else {
					req.Password = remote.Token // nolint: staticcheck
				}
			} else if remote.Password != "" {
				req.Password = remote.Password // nolint: staticcheck
			}

			// Create new certificate.
			errCert := instServer.CreateCertificate(req)

			// Refresh the server and check again whether the server is trusted.
			apiServer, _, err = instServer.GetServer()
			if err != nil {
				return nil, err
			}

			if apiServer.Auth != "trusted" {
				return nil, fmt.Errorf("Unable to authenticate with remote server: %v", errCert)
			}
		}

		// Ensure LXD version is supported by the provider.
		serverVersion := apiServer.Environment.ServerVersion

		ok, err := utils.CheckVersion(serverVersion, supportedLXDVersions)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, fmt.Errorf("LXD server with version %q does not meet the required version constraint: %q", serverVersion, supportedLXDVersions)
		}

		// Cache initialized server.
		remote.server = instServer
	}

	p.remotes[remoteName] = remote
	return remote.server, nil
}

// setRemote validates the remote and stores it into the provider config with a given name
// overwriting any existing remote.
func (p *LxdProviderConfig) setRemote(remoteName string, remote LxdRemote) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	// Validate remote.
	if remoteName == "" {
		return errors.New("Remote name cannot be empty")
	}

	if remote.Password != "" && remote.Token != "" {
		return errors.New("Remote token and password are mutually exclusive")
	}

	if !strings.HasPrefix(remote.Address, "https:") && !strings.HasPrefix(remote.Address, "unix:") {
		return fmt.Errorf(`Invalid address %q. Address must start with "https:" or "unix:"`, remote.Address)
	}

	validProtocols := []string{"lxd", "simplestreams"}
	if !slices.Contains(validProtocols, remote.Protocol) {
		return fmt.Errorf("Invalid protocol %q. Value must be one of: [%s]", remote.Protocol, strings.Join(validProtocols, ", "))
	}

	// Set default server. Only LXD server can be default server.
	if remote.IsDefault {
		if remote.Protocol != "lxd" {
			return fmt.Errorf(`Remote %q cannot be set as default remote. Default remote must use "lxd" protocol`, remoteName)
		}

		p.config.DefaultRemote = remoteName
	}

	// Store remote in LXD config to make it accessible within the LXD client.
	p.config.Remotes[remoteName] = lxdConfig.Remote{
		Addr:     remote.Address,
		Protocol: remote.Protocol,
	}

	p.remotes[remoteName] = remote
	return nil
}

// acceptRemoteCertificate retrieves the unverified peer certificate found at
// the remote address and stores it locally.
func (p *LxdProviderConfig) acceptRemoteCertificate(remoteName string, token string, url string) error {
	// Check if we are allowed to blindly accept the remote certificate.
	// When the trust token is used, the fingerprint contained in the token
	// is used to ensure we get the right certificate.
	if token == "" && !p.acceptServerCertificate {
		return errors.New("Unable to communicate with remote server. " +
			`You can set "accept_remote_certificate" to true, add ` +
			"the remote out of band of Terraform, or use the trust token.")
	}

	// Try to retrieve server's certificate.
	cert, err := shared.GetRemoteCertificate(url, "terraform-provider-lxd/"+p.version)
	if err != nil {
		return err
	}

	if token != "" {
		// Decode token.
		decodedToken, err := shared.CertificateTokenDecode(token)
		if err != nil {
			return fmt.Errorf("Failed decoding trust token for remote %q: %v", remoteName, err)
		}

		// Compare token and certificate fingerprints.
		certFingerprint := shared.CertFingerprint(cert)
		if certFingerprint != decodedToken.Fingerprint {
			return fmt.Errorf("Fingerprint mismatch between trust token and certificate from remote %q", remoteName)
		}
	}

	certPath := p.config.ServerCertPath(remoteName)
	certDir := filepath.Dir(certPath)

	// Ensure the certificate directory exists.
	err = os.MkdirAll(certDir, 0750)
	if err != nil {
		return err
	}

	// Open certificate file.
	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}

	defer certFile.Close()

	// Store certificate locally.
	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if err != nil {
		return err
	}

	return nil
}

// SelectRemote resolves the provided remote name. If the remote with a
// given name is not found, the default remote is returned.
func (p *LxdProviderConfig) SelectRemote(remoteName string) string {
	p.mux.RLock()
	defer p.mux.RUnlock()

	_, ok := p.remotes[remoteName]
	if ok {
		return remoteName
	}

	return p.config.DefaultRemote
}

// GenerateClientCertificate generates the client certificate if it does
// not already exist.
func (p *LxdProviderConfig) GenerateClientCertificate() error {
	p.mux.RLock()
	defer p.mux.RUnlock()

	err := p.config.GenerateClientCertificate()
	if err != nil {
		return fmt.Errorf("Failed to generate client certificate: %w", err)
	}

	return nil
}

// DefaultTimeout returns the default time period after which a resource
// action (read/create/update/delete) is expected to time out.
func (p *LxdProviderConfig) DefaultTimeout() time.Duration {
	return 5 * time.Minute
}

// DetermineLXDAddress is a helper function that constructs the server
// address from the provided protocol, scheme, address, and port.
func DetermineLXDAddress(protocol string, address string) (string, error) {
	var scheme string

	// Try to extract scheme from the address.
	if strings.Contains(address, "://") {
		scheme = strings.SplitN(address, "://", 2)[0]
	}

	// If scheme is still empty, determine it based on the value.
	// If address is empty or starts with "/", assume unix socket.
	if scheme == "" {
		scheme = "https"
		if address == "" || strings.HasPrefix(address, "/") {
			scheme = "unix"
		}
	}

	// Error out if simplestreams protocol is used with non-HTTPS scheme.
	if scheme != "https" && protocol == "simplestreams" {
		return "", fmt.Errorf("Simplestreams remote address %q requires HTTPS scheme", address)
	}

	// Prepend the scheme to the address.
	if !strings.HasPrefix(address, scheme+"://") {
		address = scheme + "://" + address
	}

	switch scheme {
	case "unix":
		return address, nil
	case "https":
		// Parse as URL.
		url, err := url.Parse(address)
		if err != nil {
			return "", fmt.Errorf("Failed to parse address %q: %w", address, err)
		}

		// Ensure hostname is not empty.
		if url.Hostname() == "" {
			return "", fmt.Errorf("Invalid HTTPS address %q", address)
		}

		// If port is empty, determine it based on the used protocol.
		if url.Port() == "" {
			port := "8443"
			if protocol == "simplestreams" {
				port = "443"
			}

			url.Host = url.Hostname() + ":" + port
		}

		return url.String(), nil
	default:
		return "", fmt.Errorf("Invalid scheme %q: Value must be one of: [unix, https]", scheme)
	}
}
