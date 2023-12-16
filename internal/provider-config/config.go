package config

import (
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	incus "github.com/lxc/incus/client"
	incus_api "github.com/lxc/incus/shared/api"
	incus_config "github.com/lxc/incus/shared/cliconfig"
	incus_tls "github.com/lxc/incus/shared/tls"
	incus_shared "github.com/lxc/incus/shared/util"
	"github.com/lxc/terraform-provider-incus/internal/utils"
)

// supportedIncusVersions defines Incus versions that are supported by the provider.
const supportedIncusVersions = ">= 0.1"

// A global mutex.
var mutex sync.RWMutex

// IncusProviderRemoteConfig represents Incus remote/server data as defined
// in a user's Terraform schema/configuration.
type IncusProviderRemoteConfig struct {
	Name         string
	Address      string
	Port         string
	Token        string
	Scheme       string
	Bootstrapped bool
}

// IncusProviderConfig contains the Provider configuration and initialized
// remote servers.
type IncusProviderConfig struct {
	// AcceptServerCertificates toggles if an Incus remote SSL certificate
	// should be accepted.
	acceptServerCertificate bool

	// LXDConfig is the converted form of terraformLXDConfig
	// in Incus's native data structure. This is lazy-loaded / created
	// only when a connection to an Incus remote/server happens.
	// https://github.com/lxc/incus/blob/main/lxc/config/config.go
	lxdConfig *incus_config.Config

	// remotes is a map of Incus remotes which the user has defined in
	// the Terraform schema/configuration.
	remotes map[string]IncusProviderRemoteConfig

	// servers is a map of client connections to Incus remote servers.
	// These are lazy-loaded / created only when a connection to an Incus
	// remote/server is established.
	//
	// While a client can also be retrieved from LXDConfig, this map serves
	// an additional purpose of ensuring Terraform has successfully
	// connected and authenticated to each defined Incus server/remote.
	servers map[string]incus.Server

	// This is a mutex used to handle concurrent reads/writes.
	mux sync.RWMutex
}

// NewIncusProvider returns initialized Incus provider structure. This struct is
// used to store information about this Terraform provider's configuration for
// reference throughout the lifecycle.
func NewIncusProvider(lxdConfig *incus_config.Config, acceptServerCert bool) *IncusProviderConfig {
	return &IncusProviderConfig{
		acceptServerCertificate: acceptServerCert,
		lxdConfig:               lxdConfig,
		remotes:                 make(map[string]IncusProviderRemoteConfig),
		servers:                 make(map[string]incus.Server),
	}
}

// InstanceServer returns an IncusInstanceServer client for the given remote.
// An error is returned if the remote is not a InstanceServer.
func (p *IncusProviderConfig) InstanceServer(remoteName string, project string, target string) (incus.InstanceServer, error) {
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

	if connInfo.Protocol != "incus" {
		return nil, fmt.Errorf("Remote %q (%s) is not an InstanceServer", remoteName, connInfo.Protocol)
	}

	instServer := server.(incus.InstanceServer)
	instServer = instServer.UseProject(project)
	instServer = instServer.UseTarget(target)

	return instServer, nil
}

// ImageServer returns an IncusImageServer client for the given remote.
// An error is returned if the remote is not an ImageServer.
func (p *IncusProviderConfig) ImageServer(remoteName string) (incus.ImageServer, error) {
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

	if connInfo.Protocol == "simplestreams" || connInfo.Protocol == "incus" {
		return server.(incus.ImageServer), nil
	}

	err = fmt.Errorf("Remote %q (%s / %s) is not an ImageServer", remoteName, connInfo.Protocol, connInfo.Addresses[0])
	return nil, err
}

// getServer returns a server for the named remote. The returned server
// can be either of type ImageServer or InstanceServer.
func (p *IncusProviderConfig) server(remoteName string) (incus.Server, error) {
	// If remoteName is empty, use default Incusremote (most likely "local").
	if remoteName == "" {
		remoteName = p.lxdConfig.DefaultRemote
	}

	// Check if there is an already initialized Incusserver.
	p.mux.Lock()
	server, ok := p.servers[remoteName]
	p.mux.Unlock()
	if ok {
		return server, nil
	}

	// If the server is not already created, create a new one.
	remote := p.remote(remoteName)
	if remote != nil && !remote.Bootstrapped {
		err := p.createIncusServerClient(*remote)
		if err != nil {
			return nil, fmt.Errorf("Unable to create server client for remote %q: %v", remoteName, err)
		}
	}

	lxdRemoteConfig := p.getIncusConfigRemote(remoteName)

	// If remote address is not provided or is only set to the prefix for
	// Unix sockets (`unix://`) then determine which Incus directory
	// contains a writable unix socket.
	if lxdRemoteConfig.Addr == "" || lxdRemoteConfig.Addr == "unix://" {
		lxdDir, err := determineIncusDir()
		if err != nil {
			return nil, err
		}

		_ = os.Setenv("INCUS_DIR", lxdDir)
	}

	var err error

	switch lxdRemoteConfig.Protocol {
	case "simplestreams":
		server, err = p.getIncusConfigImageServer(remoteName)
		if err != nil {
			return nil, err
		}
	default:
		server, err = p.getIncusConfigInstanceServer(remoteName)
		if err != nil {
			return nil, err
		}

		// Ensure that Incusversion meets the provider's version constraint.
		err := verifyIncusServerVersion(server.(incus.InstanceServer))
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

// createIncusServerClient will create an Incusclient for a given remote.
// The client is then stored in the lxdProvider.Config collection of clients.
func (p *IncusProviderConfig) createIncusServerClient(remote IncusProviderRemoteConfig) error {
	if remote.Address == "" {
		return nil
	}

	daemonAddr, err := determineIncusDaemonAddr(remote)
	if err != nil {
		return fmt.Errorf("Unable to determine daemon address for remote %q: %v", remote.Name, err)
	}

	lxdRemote := incus_config.Remote{Addr: daemonAddr}
	p.setIncusConfigRemote(remote.Name, lxdRemote)

	if remote.Scheme == "https" {
		// If the Incusremote's certificate does not exist on the client...
		p.mux.RLock()
		certPath := p.lxdConfig.ServerCertPath(remote.Name)
		p.mux.RUnlock()

		if !incus_shared.PathExists(certPath) {
			// Try to obtain an early connection to the remote server.
			// If it succeeds, then either the certificates between
			// the remote and the client have already been exchanged
			// or PKI is being used.
			instServer, _ := p.getIncusConfigInstanceServer(remote.Name)

			err := connectToIncusServer(instServer)
			if err != nil {
				// Either PKI isn't being used or certificates haven't been
				// exchanged. Try to add the remote server certificate.
				if p.acceptServerCertificate {
					err := p.fetchIncusServerCertificate(remote.Name)
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

		err = authenticateToIncusServer(instServer, remote.Token)
		if err != nil {
			return err
		}
	}

	return nil
}

// fetchServerCertificate will attempt to retrieve a remote Incusserver's
// certificate and save it to the servercerts path.
func (p *IncusProviderConfig) fetchIncusServerCertificate(remoteName string) error {
	lxdRemote := p.getIncusConfigRemote(remoteName)

	certificate, err := incus_tls.GetRemoteCertificate(lxdRemote.Addr, "terraform-provider-incus/1.0")
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

// verifyLXDVersion verifies whether the version of target Incusserver matches the
// provider's required version contraint.
func verifyIncusServerVersion(instServer incus.InstanceServer) error {
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

	ok, err := utils.CheckVersion(serverVersion, supportedIncusVersions)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("Incusserver with version %q does not meet the required version constraint: %q", serverVersion, supportedIncusVersions)
	}

	return nil
}

// authenticateToLXDServer authenticates to a given remote Incusserver.
// If successful, the Incusserver becomes trusted to the Incusclient,
// and vice-versa.
func authenticateToIncusServer(instServer incus.InstanceServer, token string) error {
	server, _, err := instServer.GetServer()
	if err != nil {
		return err
	}

	if server.Auth == "trusted" {
		return nil
	}

	req := incus_api.CertificatesPost{}
	req.TrustToken = token
	req.Type = "client"

	err = instServer.CreateCertificate(req)
	if err != nil {
		return fmt.Errorf("Unable to authenticate with remote server: %v", err)
	}

	_, _, err = instServer.GetServer()
	if err != nil {
		return err
	}

	return nil
}

// connectToIncusServer makes a simple GET request to the servers API to ensure
// connection can be successfully established.
func connectToIncusServer(instServer incus.InstanceServer) error {
	if instServer == nil {
		return fmt.Errorf("Instance server is nil")
	}

	_, _, err := instServer.GetServer()
	if err != nil {
		return err
	}

	return nil
}

// determineIncusDaemonAddr determines address of the Incusserver daemon.
func determineIncusDaemonAddr(remote IncusProviderRemoteConfig) (string, error) {
	var daemonAddr string

	if remote.Address != "" {
		switch remote.Scheme {
		case "unix", "":
			daemonAddr = fmt.Sprintf("unix:%s", remote.Address)
		case "https":
			daemonAddr = fmt.Sprintf("https://%s:%s", remote.Address, remote.Port)
		}
	}

	return daemonAddr, nil
}

// determineIncusDir determines which standard Incus directory contains a writable UNIX socket.
// If environment variable INCUS_DIR or INCUS_SOCKET is set, the function will return Incus directory
// based on the value from any of those variables.
func determineIncusDir() (string, error) {
	lxdSocket, ok := os.LookupEnv("INCUS_SOCKET")
	if ok {
		if utils.IsSocketWritable(lxdSocket) {
			return filepath.Dir(lxdSocket), nil
		}

		return "", fmt.Errorf("Environment variable INCUS_SOCKET points to either a non-existing or non-writable unix socket")
	}

	lxdDir, ok := os.LookupEnv("INCUS_DIR")
	if ok {
		socketPath := filepath.Join(lxdDir, "unix.socket")
		if utils.IsSocketWritable(socketPath) {
			return lxdDir, nil
		}

		return "", fmt.Errorf("Environment variable INCUS_DIR points to an Incus directory that does not contain a writable unix socket")
	}

	lxdDirs := []string{
		"/var/lib/incus",
	}

	// Iterate over Incusdirectories and find a writable unix socket.
	for _, lxdDir := range lxdDirs {
		socketPath := filepath.Join(lxdDir, "unix.socket")
		if utils.IsSocketWritable(socketPath) {
			return lxdDir, nil
		}
	}

	return "", fmt.Errorf("Incussocket with write permissions not found. Searched Incus directories: %v", lxdDirs)
}

/* Getters & Setters */

// remote returns Incusremote with the given name or default otherwise.
func (p *IncusProviderConfig) remote(name string) *IncusProviderRemoteConfig {
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

// SetRemote set Incusremote for the given name.
func (p *IncusProviderConfig) SetRemote(remote IncusProviderRemoteConfig, isDefault bool) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if isDefault {
		p.lxdConfig.DefaultRemote = remote.Name
	}

	p.remotes[remote.Name] = remote
}

// SelectRemote returns the specified remote name if it exists, or the default
// remote name otherwise.
func (p *IncusProviderConfig) SelectRemote(name string) string {
	p.mux.RLock()
	defer p.mux.RUnlock()

	_, ok := p.remotes[name]
	if ok {
		return name
	}

	return p.lxdConfig.DefaultRemote
}

// setIncusServer set Incusserver for the given name.
func (p *IncusProviderConfig) getIncusConfigRemote(name string) incus_config.Remote {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.lxdConfig.Remotes[name]
}

// setIncusServer set Incusserver for the given name.
func (p *IncusProviderConfig) setIncusConfigRemote(name string, remote incus_config.Remote) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.lxdConfig.Remotes[name] = remote
}

// getIncusConfigInstanceServer will retrieve an IncusInstanceServer client
// in a conncurrent-safe way.
func (p *IncusProviderConfig) getIncusConfigInstanceServer(remoteName string) (incus.InstanceServer, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.lxdConfig.GetInstanceServer(remoteName)
}

// getIncusConfigImageServer will retrieve an IncusImageServer client
// in a conncurrent-safe way.
func (p *IncusProviderConfig) getIncusConfigImageServer(remoteName string) (incus.ImageServer, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.lxdConfig.GetImageServer(remoteName)
}
