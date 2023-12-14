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
	incus_localtls "github.com/lxc/incus/shared/tls"
	incus_shared "github.com/lxc/incus/shared/util"
	"github.com/maveonair/terraform-provider-incus/internal/utils"
)

// supportedIncusVersions defines Incus versions that are supported by the provider.
const supportedIncusVersions = ">= 0.3.0"

// A global mutex.
var mutex sync.RWMutex

// IncusProviderRemoteConfig represents Incus remote/server data as defined
// in a user's Terraform schema/configuration.
type IncusProviderRemoteConfig struct {
	Name         string
	Address      string
	Port         string
	Password     string
	Scheme       string
	Bootstrapped bool
}

// IncusProviderConfig contains the Provider configuration and initialized
// remote servers.
type IncusProviderConfig struct {
	// AcceptServerCertificates toggles if an Incus remote SSL certificate
	// should be accepted.
	acceptServerCertificate bool

	// IncusConfig is the converted form of terraformIncusConfig
	// in Incus's native data structure. This is lazy-loaded / created
	// only when a connection to an Incus remote/server happens.
	// https://github.com/lxc/incus/blob/main/shared/cliconfig/config.go
	incusConfig *incus_config.Config

	// remotes is a map of Incus remotes which the user has defined in
	// the Terraform schema/configuration.
	remotes map[string]IncusProviderRemoteConfig

	// servers is a map of client connections to Incus remote servers.
	// These are lazy-loaded / created only when a connection to an Incus
	// remote/server is established.
	//
	// While a client can also be retrieved from IncusConfig, this map serves
	// an additional purpose of ensuring Terraform has successfully
	// connected and authenticated to each defined Incus server/remote.
	servers map[string]incus.Server

	// This is a mutex used to handle concurrent reads/writes.
	mux sync.RWMutex
}

// NewIncusProvider returns initialized Incus provider structure. This struct is
// used to store information about this Terraform provider's configuration for
// reference throughout the lifecycle.
func NewIncusProvider(incusConfig *incus_config.Config, acceptServerCert bool) *IncusProviderConfig {
	return &IncusProviderConfig{
		acceptServerCertificate: acceptServerCert,
		incusConfig:             incusConfig,
		remotes:                 make(map[string]IncusProviderRemoteConfig),
		servers:                 make(map[string]incus.Server),
	}
}

// InstanceServer returns a Incus InstanceServer client for the given remote.
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

// ImageServer returns a Incus ImageServer client for the given remote.
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
	// If remoteName is empty, use default Incus remote (most likely "local").
	if remoteName == "" {
		remoteName = p.incusConfig.DefaultRemote
	}

	// Check if there is an already initialized Incus server.
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

	incusRemoteConfig := p.getIncusConfigRemote(remoteName)

	// If remote address is not provided or is only set to the prefix for
	// Unix sockets (`unix://`) then determine which Incus directory
	// contains a writable unix socket.
	if incusRemoteConfig.Addr == "" || incusRemoteConfig.Addr == "unix://" {
		incusDir, err := determineIncusDir()
		if err != nil {
			return nil, err
		}

		_ = os.Setenv("Incus_DIR", incusDir)
	}

	var err error

	switch incusRemoteConfig.Protocol {
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

		// Ensure that Incus version meets the provider's version constraint.
		err := verifyIncusServerVersion(server.(incus.InstanceServer))
		if err != nil {
			return nil, fmt.Errorf("Remote %q: %v", remoteName, err)
		}
	}

	// Add the server to the incusServer map (cache).
	p.mux.Lock()
	defer p.mux.Unlock()

	p.servers[remoteName] = server

	return server, nil
}

// createIncusServerClient will create an Incus client for a given remote.
// The client is then stored in the incusProvider.Config collection of clients.
func (p *IncusProviderConfig) createIncusServerClient(remote IncusProviderRemoteConfig) error {
	if remote.Address == "" {
		return nil
	}

	daemonAddr, err := determineIncusDaemonAddr(remote)
	if err != nil {
		return fmt.Errorf("Unable to determine daemon address for remote %q: %v", remote.Name, err)
	}

	incusRemote := incus_config.Remote{Addr: daemonAddr}
	p.setIncusConfigRemote(remote.Name, incusRemote)

	if remote.Scheme == "https" {
		// If the Incus remote's certificate does not exist on the client...
		p.mux.RLock()
		certPath := p.incusConfig.ServerCertPath(remote.Name)
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

		err = authenticateToIncusServer(instServer, remote.Password)
		if err != nil {
			return err
		}
	}

	return nil
}

// fetchServerCertificate will attempt to retrieve a remote Incus server's
// certificate and save it to the servercerts path.
func (p *IncusProviderConfig) fetchIncusServerCertificate(remoteName string) error {
	incusRemote := p.getIncusConfigRemote(remoteName)

	certificate, err := incus_localtls.GetRemoteCertificate(incusRemote.Addr, "terraform-provider-incus/2.0")
	if err != nil {
		return err
	}

	certDir := p.incusConfig.ConfigPath("servercerts")
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

// verifyIncusVersion verifies whether the version of target Incus server matches the
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
		return fmt.Errorf("Incus server with version %q does not meet the required version constraint: %q", serverVersion, supportedIncusVersions)
	}

	return nil
}

// authenticateToIncusServer authenticates to a given remote Incus server.
// If successful, the Incus server becomes trusted to the Incus client,
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

// determineIncusDaemonAddr determines address of the Incus server daemon.
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
// If environment variable Incus_DIR or Incus_SOCKET is set, the function will return Incus directory
// based on the value from any of those variables.
func determineIncusDir() (string, error) {
	incusSocket, ok := os.LookupEnv("Incus_SOCKET")
	if ok {
		if utils.IsSocketWritable(incusSocket) {
			return filepath.Dir(incusSocket), nil
		}

		return "", fmt.Errorf("Environment variable Incus_SOCKET points to either a non-existing or non-writable unix socket")
	}

	incusDir, ok := os.LookupEnv("Incus_DIR")
	if ok {
		socketPath := filepath.Join(incusDir, "unix.socket")
		if utils.IsSocketWritable(socketPath) {
			return incusDir, nil
		}

		return "", fmt.Errorf("Environment variable Incus_DIR points to a Incus directory that does not contain a writable unix socket")
	}

	incusDirs := []string{
		"/var/lib/incus",
	}

	// Iterate over Incus directories and find a writable unix socket.
	for _, incusDir := range incusDirs {
		socketPath := filepath.Join(incusDir, "unix.socket")
		if utils.IsSocketWritable(socketPath) {
			return incusDir, nil
		}
	}

	return "", fmt.Errorf("Incus socket with write permissions not found. Searched Incus directories: %v", incusDirs)
}

/* Getters & Setters */

// remote returns Incus remote with the given name or default otherwise.
func (p *IncusProviderConfig) remote(name string) *IncusProviderRemoteConfig {
	p.mux.RLock()
	defer p.mux.RUnlock()

	remote, ok := p.remotes[name]
	if !ok {
		remote, ok = p.remotes[p.incusConfig.DefaultRemote]
		if !ok {
			return nil
		}
	}

	return &remote
}

// SetRemote set Incus remote for the given name.
func (p *IncusProviderConfig) SetRemote(remote IncusProviderRemoteConfig, isDefault bool) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if isDefault {
		p.incusConfig.DefaultRemote = remote.Name
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

	return p.incusConfig.DefaultRemote
}

// setIncusServer set Incus server for the given name.
func (p *IncusProviderConfig) getIncusConfigRemote(name string) incus_config.Remote {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.incusConfig.Remotes[name]
}

// setIncusServer set Incus server for the given name.
func (p *IncusProviderConfig) setIncusConfigRemote(name string, remote incus_config.Remote) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.incusConfig.Remotes[name] = remote
}

// getIncusConfigInstanceServer will retrieve an Incus InstanceServer client
// in a conncurrent-safe way.
func (p *IncusProviderConfig) getIncusConfigInstanceServer(remoteName string) (incus.InstanceServer, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.incusConfig.GetInstanceServer(remoteName)
}

// getIncusConfigImageServer will retrieve an Incus ImageServer client
// in a conncurrent-safe way.
func (p *IncusProviderConfig) getIncusConfigImageServer(remoteName string) (incus.ImageServer, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.incusConfig.GetImageServer(remoteName)
}
