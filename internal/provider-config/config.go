package config

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
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

// DefaultProject is the default LXD project used by the provider when no project is specified.
const DefaultProject = "default"

// LxdRemote contains the configuration for a single LXD remote.
type LxdRemote struct {
	Protocol string
	Address  string

	// Server certificate verification (fingerprint of the server's TLS certificate).
	ServerCertificateFingerprint string

	// mTLS authentication.
	ClientCertificate string
	ClientKey         string

	// Trust token for initial trust of client certificates.
	TrustToken string

	// Bearer token authentication.
	BearerToken string

	// server represents a cached client connection to the remote server.
	server lxd.Server
}

// LxdProviderConfig contains the provider configuration and initialized
// remote servers.
type LxdProviderConfig struct {
	// version of the provider.
	version string

	// remotes is a map of all remotes accessible to the provider.
	remotes map[string]LxdRemote

	// defaultRemote is the name of the default remote, which is used when a
	// resource or data source does not explicitly specify a remote.
	defaultRemote string

	// mux is a lock that handle concurrent reads/writes to the LXD config.
	mux sync.RWMutex
}

// NewLxdProviderConfig initializes a new provider configuration from the given
// remotes and options. At least one remote must be provided.
func NewLxdProviderConfig(version string, remotes map[string]LxdRemote, defaultRemote string) (*LxdProviderConfig, error) {
	if len(remotes) == 0 {
		return nil, fmt.Errorf("At least one remote must be defined in the provider configuration")
	}

	config := &LxdProviderConfig{
		version: version,
		remotes: builtinRemotes(),
	}

	// Validate remotes.
	for name, remote := range remotes {
		if name == "" {
			return nil, fmt.Errorf("Remote name cannot be empty")
		}

		if remote.Protocol == "" {
			remote.Protocol = "lxd"
		}

		if remote.Protocol != "lxd" && remote.Protocol != "simplestreams" {
			return nil, fmt.Errorf("Invalid protocol %q for remote %q. Value must be one of: [lxd, simplestreams]", remote.Protocol, name)
		}

		if !strings.HasPrefix(remote.Address, "https:") && !strings.HasPrefix(remote.Address, "unix:") {
			return nil, fmt.Errorf(`Invalid remote address %q. Address must start with "https:" or "unix:"`, remote.Address)
		}

		config.remotes[name] = remote

		// Set default remote if the name matches or if only 1 remote is defined.
		if defaultRemote == name || (defaultRemote == "" && len(remotes) == 1) {
			config.defaultRemote = name
		}
	}

	// Ensure default remote points to a valid remote when multiple remotes are defined.
	if config.defaultRemote == "" {
		if defaultRemote != "" {
			return nil, fmt.Errorf("Default remote %q is not defined in the provider configuration", defaultRemote)
		}

		return nil, errors.New("When multiple remotes are defined, a default remote must be specified")
	}

	return config, nil
}

// InstanceServer returns a LXD InstanceServer client for the given remote.
// An error is returned if the remote is not a InstanceServer.
func (p *LxdProviderConfig) InstanceServer(remoteName string, project string, target string) (lxd.InstanceServer, error) {
	remoteName = p.selectRemote(remoteName)

	server, err := p.server(remoteName)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve server for remote %q: %w", remoteName, err)
	}

	connInfo, err := server.GetConnectionInfo()
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection info for remote %q: %w", remoteName, err)
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
	remoteName = p.selectRemote(remoteName)

	server, err := p.server(remoteName)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve server for remote %q: %w", remoteName, err)
	}

	connInfo, err := server.GetConnectionInfo()
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection info for remote %q: %w", remoteName, err)
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

	var server lxd.Server
	var err error

	remoteName = p.selectRemote(remoteName)

	remote, ok := p.remotes[remoteName]
	if !ok {
		return nil, fmt.Errorf("Unknown remote %q", remoteName)
	}

	if remote.server != nil {
		// Return cached server for the main provider remote.
		return remote.server, nil
	}

	// Validate LXD server version for lxd protocol remotes.
	userAgent := "terraform-provider-lxd/" + p.version

	connArgs, err := p.buildConnectionArgs(remote, userAgent)
	if err != nil {
		return nil, err
	}

	// Connect to the server based on the specified protocol.
	// If remoteName is provided, the caller is asking for the image server.
	switch remote.Protocol {
	case "simplestreams":
		// For simplestreams protocol, we only support HTTPS connections.
		server, err = lxd.ConnectSimpleStreams(remote.Address, connArgs)
		if err != nil {
			return nil, fmt.Errorf("Failed to connect to simplestreams server: %w", err)
		}
	case "", "lxd":
		address, ok := strings.CutPrefix(remote.Address, "unix://")
		if ok {
			// Unix connection.
			server, err = lxd.ConnectLXDUnix(address, connArgs)
		} else {
			// HTTPS connection.
			server, err = lxd.ConnectLXD(address, connArgs)
		}

		if err != nil {
			return nil, fmt.Errorf("Failed to connect to LXD server: %w", err)
		}

		// Validate LXD server version.
		instServer, ok := server.(lxd.InstanceServer)
		if !ok {
			return nil, fmt.Errorf("Connected to LXD server, but it does not support the InstanceServer interface")
		}

		apiServer, _, err := instServer.GetServer()
		if err != nil {
			return nil, fmt.Errorf("Failed to get server info: %w", err)
		}

		// Authenticate against HTTPS remote if it is not already trusted.
		if apiServer.Auth != "trusted" {
			if remote.TrustToken == "" {
				return nil, fmt.Errorf("Unable to authenticate with remote server: Client not trusted")
			}

			// Trust token is provided, try to authenticate using the trust
			// token and client certificate.
			req := api.CertificatesPost{
				Type: "client",
			}

			if instServer.HasExtension("explicit_trust_token") {
				req.TrustToken = remote.TrustToken
			} else {
				req.Password = remote.TrustToken // nolint: staticcheck
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

		serverVersion := apiServer.Environment.ServerVersion
		versionOK, err := utils.CheckVersion(serverVersion, supportedLXDVersions)
		if err != nil {
			return nil, err
		}

		if !versionOK {
			return nil, fmt.Errorf("LXD server with version %q does not meet the required version constraint: %q", serverVersion, supportedLXDVersions)
		}
	default:
		return nil, fmt.Errorf("Invalid protocol %q: Value must be one of: [lxd, simplestreams]", remote.Protocol)
	}

	// Cache initialized server.
	remote.server = server
	p.remotes[remoteName] = remote

	return server, nil
}

// buildConnectionArgs constructs ConnectionArgs for an HTTPS LXD connection.
// It handles bearer token injection, mTLS, and server certificate verification.
func (p *LxdProviderConfig) buildConnectionArgs(remote LxdRemote, userAgent string) (*lxd.ConnectionArgs, error) {
	args := &lxd.ConnectionArgs{
		UserAgent: userAgent,
	}

	if strings.HasPrefix(remote.Address, "unix:") {
		// For LXD remote using unix socket, we set only user agent.
		return args, nil
	}

	if remote.BearerToken != "" && (remote.ClientCertificate != "" || remote.ClientKey != "") {
		return nil, fmt.Errorf("Cannot use both bearer token and TLS client certificate/key for authentication")
	}

	if remote.TrustToken != "" && (remote.ClientCertificate == "" || remote.ClientKey == "") {
		return nil, fmt.Errorf("Trust token can only be used with TLS client certificate and key for initial trust establishment")
	}

	if (remote.ClientCertificate != "" || remote.ClientKey != "") && (remote.ClientCertificate == "" || remote.ClientKey == "") {
		return nil, fmt.Errorf("Both client certificate and client key must be provided for TLS authentication")
	}

	if remote.ClientCertificate != "" {
		args.TLSClientCert = remote.ClientCertificate
		args.TLSClientKey = remote.ClientKey
	}

	if remote.BearerToken != "" {
		args.BearerToken = remote.BearerToken
	}

	if remote.TrustToken != "" {
		trustToken, err := shared.CertificateTokenDecode(remote.TrustToken)
		if err != nil {
			return nil, fmt.Errorf("Failed decoding trust token: %w", err)
		}

		if remote.ServerCertificateFingerprint == "" {
			remote.ServerCertificateFingerprint = trustToken.Fingerprint
		} else if !strings.EqualFold(trustToken.Fingerprint, remote.ServerCertificateFingerprint) {
			return nil, fmt.Errorf("Trust token fingerprint does not match the provided server certificate fingerprint: %q != %q", trustToken.Fingerprint, remote.ServerCertificateFingerprint)
		}
	}

	// Server certificate verification.
	if remote.ServerCertificateFingerprint != "" {
		// Fetch the server certificate (using InsecureSkipVerify to bootstrap)
		// and verify its fingerprint before trusting it.
		cert, err := shared.GetRemoteCertificate(context.Background(), remote.Address, userAgent)
		if err != nil {
			return nil, fmt.Errorf("Failed to retrieve server certificate: %w", err)
		}

		fingerprint := shared.CertFingerprint(cert)
		if fingerprint != remote.ServerCertificateFingerprint {
			return nil, fmt.Errorf(
				"Server certificate fingerprint mismatch: expected %q, got %q",
				remote.ServerCertificateFingerprint,
				fingerprint,
			)
		}

		// Pin the verified certificate.
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
		args.TLSServerCert = string(certPEM)
	}

	return args, nil
}

// selectRemote returns the provided remote name if it is not empty,
// otherwise it returns the default remote name.
func (p *LxdProviderConfig) selectRemote(remoteName string) string {
	if remoteName != "" {
		return remoteName
	}

	return p.defaultRemote
}

// ToHCL returns the provider configuration as an HCL provider block string.
func (p *LxdProviderConfig) ToHCL() string {
	p.mux.RLock()
	defer p.mux.RUnlock()

	var b strings.Builder
	b.WriteString(`provider "lxd" {` + "\n")
	fmt.Fprintf(&b, "  default_remote = %q\n\n", p.defaultRemote)

	builtinRemoteNames := []string{""}
	for name := range builtinRemotes() {
		builtinRemoteNames = append(builtinRemoteNames, name)
	}

	for name, remote := range p.remotes {
		if slices.Contains(builtinRemoteNames, name) {
			continue
		}

		b.WriteString("  remote {\n")
		fmt.Fprintf(&b, "    name    = %q\n", name)
		fmt.Fprintf(&b, "    address = %q\n", remote.Address)

		if remote.Protocol != "" && remote.Protocol != "lxd" {
			fmt.Fprintf(&b, "    protocol = %q\n", remote.Protocol)
		}

		if remote.BearerToken != "" {
			fmt.Fprintf(&b, "    bearer_token = %q\n", remote.BearerToken)
		}

		if remote.ClientCertificate != "" {
			fmt.Fprintf(&b, "    client_certificate = %q\n", remote.ClientCertificate)
		}

		if remote.ClientKey != "" {
			fmt.Fprintf(&b, "    client_key = %q\n", remote.ClientKey)
		}

		if remote.ServerCertificateFingerprint != "" {
			fmt.Fprintf(&b, "    server_certificate_fingerprint = %q\n", remote.ServerCertificateFingerprint)
		}

		b.WriteString("  }\n")
	}

	b.WriteString("}\n")
	return b.String()
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
		scheme, _, _ = strings.Cut(address, "://")
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

			host := url.Hostname()
			// If the hostname contains ':' it's likely an IPv6 address and
			// must be enclosed in brackets when adding a port.
			if strings.Contains(host, ":") {
				url.Host = "[" + host + "]:" + port
			} else {
				url.Host = host + ":" + port
			}
		}

		return url.String(), nil
	default:
		return "", fmt.Errorf("Invalid scheme %q: Value must be one of: [unix, https]", scheme)
	}
}

// builtinRemotes returns a map of remotes that are built into the provider.
func builtinRemotes() map[string]LxdRemote {
	remotes := make(map[string]LxdRemote)

	// Load pre-defined image remotes from default LXD config.
	for name, r := range lxdConfig.DefaultConfig().Remotes {
		if r.Protocol != "simplestreams" {
			continue
		}

		remotes[name] = LxdRemote{
			Protocol: r.Protocol,
			Address:  r.Addr,
		}
	}

	return remotes
}
