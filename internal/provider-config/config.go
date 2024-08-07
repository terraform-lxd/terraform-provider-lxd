package config

import (
	"encoding/pem"
	"fmt"
	"os"
	"sync"
	"time"

	lxd "github.com/canonical/lxd/client"
	lxd_config "github.com/canonical/lxd/lxc/config"
	lxd_shared "github.com/canonical/lxd/shared"
	lxd_api "github.com/canonical/lxd/shared/api"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

// supportedLXDVersions defines LXD versions that are supported by the provider.
const supportedLXDVersions = ">= 4.0.0"

// A global mutex.
var mutex sync.RWMutex

// LxdProviderRemoteConfig represents LXD remote/server data as defined
// in a user's Terraform schema/configuration.
type LxdProviderRemoteConfig struct {
	Name         string
	Address      string
	Port         string
	Password     string
	Protocol     string
	Token        string
	Scheme       string
	Bootstrapped bool
}

// LxdProviderConfig contains the Provider configuration and initialized
// remote servers.
type LxdProviderConfig struct {
	// AcceptServerCertificates toggles if an LXD remote SSL certificate
	// should be accepted.
	acceptServerCertificate bool

	// LXDConfig is the converted form of terraformLXDConfig
	// in LXD's native data structure. This is lazy-loaded / created
	// only when a connection to an LXD remote/server happens.
	// https://github.com/canonical/lxd/blob/main/lxc/config/config.go
	lxdConfig *lxd_config.Config

	// remotes is a map of LXD remotes which the user has defined in
	// the Terraform schema/configuration.
	remotes map[string]LxdProviderRemoteConfig

	// servers is a map of client connections to LXD remote servers.
	// These are lazy-loaded / created only when a connection to an LXD
	// remote/server is established.
	//
	// While a client can also be retrieved from LXDConfig, this map serves
	// an additional purpose of ensuring Terraform has successfully
	// connected and authenticated to each defined LXD server/remote.
	servers map[string]lxd.Server

	// This is a mutex used to handle concurrent reads/writes.
	mux sync.RWMutex
}

// NewLxdProvider returns initialized LXD provider structure. This struct is
// used to store information about this Terraform provider's configuration for
// reference throughout the lifecycle.
func NewLxdProvider(lxdConfig *lxd_config.Config, acceptServerCert bool) *LxdProviderConfig {
	return &LxdProviderConfig{
		acceptServerCertificate: acceptServerCert,
		lxdConfig:               lxdConfig,
		remotes:                 make(map[string]LxdProviderRemoteConfig),
		servers:                 make(map[string]lxd.Server),
	}
}

// InstanceServer returns a LXD InstanceServer client for the given remote.
// An error is returned if the remote is not a InstanceServer.
func (p *LxdProviderConfig) InstanceServer(remoteName string, project string, target string) (lxd.InstanceServer, error) {
	server, err := p.server(remoteName)
	if err != nil {
		return nil, err
	}

	p.mux.RLock()
	defer p.mux.RUnlock()

	connInfo, err := server.GetConnectionInfo()
	if err != nil {
		return nil, err
	}

	if connInfo.Protocol != "lxd" {
		return nil, fmt.Errorf("Remote %q (%s) is not an InstanceServer", remoteName, connInfo.Protocol)
	}

	instServer := server.(lxd.InstanceServer)
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

	p.mux.RLock()
	defer p.mux.RUnlock()

	connInfo, err := server.GetConnectionInfo()
	if err != nil {
		return nil, err
	}

	if connInfo.Protocol == "simplestreams" || connInfo.Protocol == "lxd" {
		return server.(lxd.ImageServer), nil
	}

	err = fmt.Errorf("Remote %q (%s / %s) is not an ImageServer", remoteName, connInfo.Protocol, connInfo.Addresses[0])
	return nil, err
}

// getServer returns a server for the named remote. The returned server
// can be either of type ImageServer or InstanceServer.
func (p *LxdProviderConfig) server(remoteName string) (lxd.Server, error) {
	// If remoteName is empty, use default LXD remote (most likely "local").
	if remoteName == "" {
		remoteName = p.lxdConfig.DefaultRemote
	}

	// Check if there is an already initialized LXD server.
	p.mux.Lock()
	server, ok := p.servers[remoteName]
	p.mux.Unlock()
	if ok {
		return server, nil
	}

	// If the server is not already created, create a new one.
	remote := p.remote(remoteName)
	if remote != nil && !remote.Bootstrapped {
		err := p.createLxdServerClient(*remote)
		if err != nil {
			return nil, fmt.Errorf("Unable to create server client for remote %q: %v", remoteName, err)
		}
	}

	var err error

	lxdRemoteConfig := p.getLxdConfigRemote(remoteName)

	switch lxdRemoteConfig.Protocol {
	case "simplestreams":
		server, err = p.getLxdConfigImageServer(remoteName)
		if err != nil {
			return nil, err
		}
	default:
		server, err = p.getLxdConfigInstanceServer(remoteName)
		if err != nil {
			return nil, err
		}

		// Ensure that LXD version meets the provider's version constraint.
		err := verifyLxdServerVersion(server.(lxd.InstanceServer))
		if err != nil {
			return nil, fmt.Errorf("Remote %q: %v", remoteName, err)
		}
	}

	// Add the server to the lxdServer map (cache).
	p.mux.Lock()
	defer p.mux.Unlock()

	p.servers[remoteName] = server

	return server, nil
}

// createLxdServerClient will create an LXD client for a given remote.
// The client is then stored in the lxdProvider.Config collection of clients.
func (p *LxdProviderConfig) createLxdServerClient(remote LxdProviderRemoteConfig) error {
	if remote.Address == "" {
		return nil
	}

	lxdRemote := lxd_config.Remote{
		Addr:     determineLxdDaemonAddr(remote),
		Protocol: remote.Protocol,
	}

	p.setLxdConfigRemote(remote.Name, lxdRemote)

	if remote.Scheme == "https" && remote.Protocol == "lxd" {
		// If the LXD remote's certificate does not exist on the client...
		p.mux.RLock()
		certPath := p.lxdConfig.ServerCertPath(remote.Name)
		p.mux.RUnlock()

		if !lxd_shared.PathExists(certPath) {
			// Try to obtain an early connection to the remote server.
			// If it succeeds, then either the certificates between
			// the remote and the client have already been exchanged
			// or PKI is being used.
			instServer, _ := p.getLxdConfigInstanceServer(remote.Name)

			err := connectToLxdServer(instServer)
			if err != nil {
				// Either PKI isn't being used or certificates haven't been
				// exchanged. Try to add the remote server certificate.
				if p.acceptServerCertificate {
					err := p.fetchLxdServerCertificate(remote.Name)
					if err != nil {
						return fmt.Errorf("Failed to get remote server certificate: %v", err)
					}
				} else {
					return fmt.Errorf("Unable to communicate with remote server. Either set " +
						"accept_remote_certificate to true or add the remote out of band " +
						"of Terraform and try again.")
				}
			}
		}

		// Set bootstrapped to true to prevent an infinite loop.
		// This is required for situations when a remote might be
		// defined in a config.yml file but the client has not yet
		// exchanged certificates with the remote.
		remote.Bootstrapped = true
		p.SetRemote(remote, false)

		// Finally, make sure the client is authenticated.
		instServer, err := p.InstanceServer(remote.Name, "", "")
		if err != nil {
			return err
		}

		p.mux.Lock()
		defer p.mux.Unlock()

		server, _, err := instServer.GetServer()
		if err != nil {
			return err
		}

		// Authenticate to the remote LXD server. If successful, the LXD server becomes
		// trusted to the LXD client, and vice-versa.
		if server.Auth != "trusted" {
			req := lxd_api.CertificatesPost{}
			req.Type = "client"

			if remote.Token != "" {
				if instServer.HasExtension("explicit_trust_token") {
					req.TrustToken = remote.Token
				} else {
					req.Password = remote.Token
				}
			} else if remote.Password != "" {
				ok, err := utils.CheckVersion(server.Environment.ServerVersion, ">= 6.0.0")
				if err != nil {
					return err
				}

				if !ok {
					return fmt.Errorf("LXD server does not support password authentication from version 6.0 onwards")
				}

				req.Password = remote.Password
			}

			// Create new certificate.
			errCert := instServer.CreateCertificate(req)
			if errCert != nil {
				// If request to create a certificate failed, refresh the
				// server and check again whether the server is trusted.
				server, _, err = instServer.GetServer()
				if err != nil {
					return err
				}

				if server.Auth != "trusted" {
					return fmt.Errorf("Unable to authenticate with remote server: %v", errCert)
				}
			}
		}
	}

	return nil
}

// fetchServerCertificate will attempt to retrieve a remote LXD server's
// certificate and save it to the servercerts path.
func (p *LxdProviderConfig) fetchLxdServerCertificate(remoteName string) error {
	lxdRemote := p.getLxdConfigRemote(remoteName)

	certificate, err := lxd_shared.GetRemoteCertificate(lxdRemote.Addr, "terraform-provider-lxd/2.0")
	if err != nil {
		return err
	}

	certDir := p.lxdConfig.ConfigPath("servercerts")
	err = os.MkdirAll(certDir, 0750)
	if err != nil {
		return err
	}

	certPath := fmt.Sprintf("%s/%s.crt", certDir, remoteName)
	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}

	defer certFile.Close()

	return pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})
}

// verifyLXDVersion verifies whether the version of target LXD server matches the
// provider's required version contraint.
func verifyLxdServerVersion(instServer lxd.InstanceServer) error {
	server, _, err := instServer.GetServer()
	if err != nil {
		return err
	}

	serverVersion := server.Environment.ServerVersion
	if serverVersion == "" {
		// If server version is empty, it means that authentication
		// has failed, therefore we can ignore version check.
		return nil
	}

	ok, err := utils.CheckVersion(serverVersion, supportedLXDVersions)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("LXD server with version %q does not meet the required version constraint: %q", serverVersion, supportedLXDVersions)
	}

	return nil
}

// connectToLxdServer makes a simple GET request to the servers API to ensure
// connection can be successfully established.
func connectToLxdServer(instServer lxd.InstanceServer) error {
	if instServer == nil {
		return fmt.Errorf("Instance server is nil")
	}

	_, _, err := instServer.GetServer()
	if err != nil {
		return err
	}

	return nil
}

// determineLxdDaemonAddr determines address of the LXD server daemon.
func determineLxdDaemonAddr(remote LxdProviderRemoteConfig) string {
	var daemonAddr string

	if remote.Address != "" {
		switch remote.Scheme {
		case "unix", "":
			daemonAddr = fmt.Sprintf("unix:%s", remote.Address)
		case "https":
			daemonAddr = fmt.Sprintf("https://%s:%s", remote.Address, remote.Port)
		}
	}

	return daemonAddr
}

/* Getters & Setters */

// remote returns LXD remote with the given name or default otherwise.
func (p *LxdProviderConfig) remote(name string) *LxdProviderRemoteConfig {
	p.mux.RLock()
	defer p.mux.RUnlock()

	remote, ok := p.remotes[name]
	if !ok {
		remote, ok = p.remotes[p.lxdConfig.DefaultRemote]
		if !ok {
			return nil
		}
	}

	return &remote
}

// SetRemote set LXD remote for the given name.
func (p *LxdProviderConfig) SetRemote(remote LxdProviderRemoteConfig, isDefault bool) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if isDefault {
		p.lxdConfig.DefaultRemote = remote.Name
	}

	p.remotes[remote.Name] = remote
}

// SelectRemote returns the specified remote name if it exists, or the default
// remote name otherwise.
func (p *LxdProviderConfig) SelectRemote(name string) string {
	p.mux.RLock()
	defer p.mux.RUnlock()

	_, ok := p.remotes[name]
	if ok {
		return name
	}

	return p.lxdConfig.DefaultRemote
}

// setLxdServer set LXD server for the given name.
func (p *LxdProviderConfig) getLxdConfigRemote(name string) lxd_config.Remote {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.lxdConfig.Remotes[name]
}

// setLxdServer set LXD server for the given name.
func (p *LxdProviderConfig) setLxdConfigRemote(name string, remote lxd_config.Remote) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.lxdConfig.Remotes[name] = remote
}

// getLxdConfigInstanceServer will retrieve an LXD InstanceServer client
// in a conncurrent-safe way.
func (p *LxdProviderConfig) getLxdConfigInstanceServer(remoteName string) (lxd.InstanceServer, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.lxdConfig.GetInstanceServer(remoteName)
}

// getLxdConfigImageServer will retrieve an LXD ImageServer client
// in a conncurrent-safe way.
func (p *LxdProviderConfig) getLxdConfigImageServer(remoteName string) (lxd.ImageServer, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.lxdConfig.GetImageServer(remoteName)
}

// DefaultTimeout returns the default time period after which a resource
// action (read/create/update/delete) is expected to time out.
func (p *LxdProviderConfig) DefaultTimeout() time.Duration {
	return 5 * time.Minute
}
