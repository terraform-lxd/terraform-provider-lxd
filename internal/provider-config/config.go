package config

import (
	"context"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	lxd "github.com/canonical/lxd/client"
	lxdConfig "github.com/canonical/lxd/lxc/config"
	"github.com/canonical/lxd/shared"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

// supportedLXDVersions defines LXD versions that are supported by the provider.
const supportedLXDVersions = ">= 4.0.0"

// LxdRemote contains the configuration for a single LXD remote.
type LxdRemote struct {
	Protocol string
	Address  string

	// Server certificate verification (fingerprint of the server's TLS certificate).
	ServerCertificateFingerprint string

	// mTLS authentication.
	ClientCertificate string
	ClientKey         string

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

	// remote contains the configuration of a target remote.
	remote LxdRemote

	// imageServers contains default simplestream remotes sourced from the LXD itself.
	imageServers map[string]LxdRemote

	// mux is a lock that handles concurrent reads/writes.
	mux sync.RWMutex
}

// NewLxdProviderConfig initializes a new provider configuration from the given
// remotes and options. At least one remote must be provided.
func NewLxdProviderConfig(version string, remote LxdRemote) (*LxdProviderConfig, error) {
	if remote.Protocol == "" {
		remote.Protocol = "lxd"
	}

	if remote.Address == "" {
		return nil, fmt.Errorf("Provider address must be set")
	}

	// Load pre-defined image remotes from LXD config.
	imageServers := make(map[string]LxdRemote)
	for name, r := range lxdConfig.DefaultConfig().Remotes {
		if r.Protocol == "simplestreams" {
			imageServers[name] = LxdRemote{
				Protocol: r.Protocol,
				Address:  r.Addr,
			}
		}
	}

	return &LxdProviderConfig{
		version:      version,
		remote:       remote,
		imageServers: imageServers,
	}, nil
}

// InstanceServer returns a LXD InstanceServer client for the given remote.
// An error is returned if the remote is not an InstanceServer.
func (p *LxdProviderConfig) InstanceServer(project string, target string) (lxd.InstanceServer, error) {
	server, err := p.server("")
	if err != nil {
		return nil, err
	}

	connInfo, err := server.GetConnectionInfo()
	if err != nil {
		return nil, err
	}

	if connInfo.Protocol != "lxd" {
		return nil, fmt.Errorf("Server (%s) is not an InstanceServer", connInfo.Protocol)
	}

	instServer, ok := server.(lxd.InstanceServer)
	if !ok {
		return nil, fmt.Errorf("Server is not an InstanceServer")
	}

	instServer = instServer.UseProject(project)
	instServer = instServer.UseTarget(target)

	return instServer, nil
}

// ImageServer returns a LXD ImageServer client for the given remote.
// Empty remoteName corresponds to the main provider remote, while non-empty remoteName
// corresponds to a pre-defined image server remote.
// An error is returned if the remote is not an [lxd.ImageServer].
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
		return nil, fmt.Errorf("Server (%s / %s) is not an ImageServer", connInfo.Protocol, connInfo.Addresses[0])
	}

	imageServer, ok := server.(lxd.ImageServer)
	if !ok {
		return nil, fmt.Errorf("Server is not an ImageServer")
	}

	return imageServer, nil
}

// server returns a server for the named remote. The returned server
// can be either of type ImageServer or InstanceServer.
func (p *LxdProviderConfig) server(remoteName string) (lxd.Server, error) {
	p.mux.Lock()
	defer p.mux.Unlock()

	var (
		server lxd.Server
		err    error
	)

	// Validate LXD server version for lxd protocol remotes.
	protocol := p.remote.Protocol
	address := p.remote.Address
	userAgent := "terraform-provider-lxd/" + p.version

	connArgs, err := p.buildConnectionArgs(p.remote, userAgent)
	if err != nil {
		return nil, err
	}

	if remoteName != "" {
		// Use pre-defined image server remote from LXD config.
		imageRemote, ok := p.imageServers[remoteName]
		if !ok {
			return nil, fmt.Errorf("Unknown image remote %q", remoteName)
		}

		if imageRemote.server != nil {
			// Return cached server for the image remote.
			return imageRemote.server, nil
		}

		server, err = lxd.ConnectSimpleStreams(imageRemote.Address, connArgs)
		if err != nil {
			return nil, fmt.Errorf("Failed to connect to simplestreams server for remote %q: %w", remoteName, err)
		}

		// Cache initialized server for the remote.
		imageRemote.server = server
		p.imageServers[remoteName] = imageRemote
		return server, nil
	}

	if p.remote.server != nil {
		// Return cached server for the main provider remote.
		return p.remote.server, nil
	}

	// Connect to the server based on the specified protocol.
	// If remoteName is provided, the caller is asking for the image server.
	switch protocol {
	case "simplestreams":
		// For simplestreams protocol, we only support HTTPS connections.
		server, err = lxd.ConnectSimpleStreams(address, connArgs)
		if err != nil {
			return nil, fmt.Errorf("Failed to connect to simplestreams server: %w", err)
		}
	case "lxd":
		// Check if there is an already initialized LXD server.
		if p.remote.server != nil {
			return p.remote.server, nil
		}

		if strings.HasPrefix(address, "unix:") {
			// Unix connection.
			path := strings.TrimPrefix(address, "unix://")
			server, err = lxd.ConnectLXDUnix(path, connArgs)
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

		serverVersion := apiServer.Environment.ServerVersion
		versionOK, err := utils.CheckVersion(serverVersion, supportedLXDVersions)
		if err != nil {
			return nil, err
		}

		if !versionOK {
			return nil, fmt.Errorf("LXD server with version %q does not meet the required version constraint: %q", serverVersion, supportedLXDVersions)
		}
	default:
		return nil, fmt.Errorf("Invalid protocol %q: Value must be one of: [lxd, simplestreams]", protocol)
	}

	// Cache initialized server.
	p.remote.server = server

	return server, nil
}

// buildConnectionArgs constructs ConnectionArgs for an HTTPS LXD connection.
// It handles bearer token injection, mTLS, and server certificate verification.
func (p *LxdProviderConfig) buildConnectionArgs(remote LxdRemote, userAgent string) (*lxd.ConnectionArgs, error) {
	args := &lxd.ConnectionArgs{
		UserAgent: userAgent,
	}

	if remote.Protocol != "lxd" || strings.HasPrefix(remote.Address, "unix:") {
		// For LXD remote using unix socket and simplestreams protocol,
		// we set only user agent.
		return args, nil
	}

	if remote.BearerToken != "" && (remote.ClientCertificate != "" || remote.ClientKey != "") {
		return nil, fmt.Errorf("Cannot use both bearer token and TLS client certificate/key for authentication")
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

// DefaultTimeout returns the default time period after which a resource
// action (read/create/update/delete) is expected to time out.
func (p *LxdProviderConfig) DefaultTimeout() time.Duration {
	return 5 * time.Minute
}

// DetermineLXDAddress is a helper function that constructs the server
// address from the provided protocol and address string.
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

			url.Host = url.Hostname() + ":" + port
		}

		return url.String(), nil
	default:
		return "", fmt.Errorf("Invalid scheme %q: Value must be one of: [unix, https]", scheme)
	}
}
