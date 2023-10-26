package lxd

import (
	"fmt"
	"log"
	"strings"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLxdNetworkLB() *schema.Resource {
	return &schema.Resource{
		Read:   resourceLxdNetworkLBRead,
		Create: resourceLxdNetworkLBCreate,
		Update: resourceLxdNetworkLBUpdate,
		Delete: resourceLxdNetworkLBDelete,
		Exists: resourceLxdNetworkLBExists,

		Schema: map[string]*schema.Schema{
			"listen_address": {
				Type:     schema.TypeString,
				Required: true,
			},

			"network": {
				Type:     schema.TypeString,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"backend": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"description": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"target_address": {
							Type:     schema.TypeString,
							Required: true,
						},

						"target_port": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"port": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"protocol": {
							Type:     schema.TypeString,
							Default:  "tcp",
							Optional: true,
							ValidateFunc: func(v interface{}, res string) ([]string, []error) {
								protocols := []string{"tcp", "udp"}

								protocol, ok := v.(string)
								if ok && !ValueInSlice(protocol, protocols) {
									err := fmt.Errorf("Argument %q must be one of the following values: %s", res, strings.Join(protocols, ", "))
									return nil, []error{err}
								}

								return nil, nil
							},
						},

						"listen_port": {
							Type:     schema.TypeString,
							Required: true,
						},

						"target_backend": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Required: true,
							MinItems: 1,
						},
					},
				},
			},
		},
	}
}

func resourceLxdNetworkLBRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)

	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	project, ok := d.GetOk("project")
	if ok && project != "" {
		server = server.UseProject(project.(string))
	}

	network := d.Get("network").(string)
	listenAddr := d.Get("listen_address").(string)

	// Id is a combination of network name and listen address.
	d.SetId(fmt.Sprintf("%s/%s", network, listenAddr))

	lb, _, err := server.GetNetworkLoadBalancer(network, listenAddr)
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}

		return err
	}

	backends := make([]map[string]interface{}, 0, len(lb.Backends))
	for _, b := range lb.Backends {
		backend := make(map[string]interface{})
		backend["name"] = b.Name
		backend["description"] = b.Description
		backend["target_address"] = b.TargetAddress
		backend["target_port"] = b.TargetPort

		backends = append(backends, backend)
	}

	ports := make([]map[string]interface{}, 0, len(lb.Ports))
	for _, p := range lb.Ports {
		port := make(map[string]interface{})
		port["description"] = p.Description
		port["listen_port"] = p.ListenPort
		port["target_backend"] = p.TargetBackend
		port["protocol"] = p.Protocol

		ports = append(ports, port)
	}

	d.Set("description", lb.Description)
	d.Set("listen_address", lb.ListenAddress)
	d.Set("config", lb.Config)
	d.Set("backend", backends)
	d.Set("port", ports)

	log.Printf("[DEBUG] Retrieved network load balancer %q: %+v", network, lb)

	return nil
}

func resourceLxdNetworkLBCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)

	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	network := d.Get("network").(string)

	req := api.NetworkLoadBalancersPost{}
	req.ListenAddress = d.Get("listen_address").(string)
	req.Description = d.Get("description").(string)
	req.Config = resourceLxdConfigMap(d.Get("config"))
	req.Backends = resourceLxdNetworkLBBackends(d.Get("backend"))
	req.Ports = resourceLxdNetworkLBPorts(d.Get("port"))

	// log.Printf("[DEBUG] Creating network load balancer %q with config: %#v", network, config)

	mutex.Lock()
	err = server.CreateNetworkLoadBalancer(network, req)
	mutex.Unlock()

	if err != nil {
		return err
	}

	d.SetId(network)

	return resourceLxdNetworkLBRead(d, meta)
}

func resourceLxdNetworkLBUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	network := d.Get("network").(string)
	listenAddr := d.Get("listen_address").(string)

	_, etag, err := server.GetNetworkLoadBalancer(network, listenAddr)
	if err != nil {
		return err
	}

	req := api.NetworkLoadBalancerPut{}
	req.Description = d.Get("description").(string)
	req.Config = resourceLxdConfigMap(d.Get("config"))
	req.Backends = resourceLxdNetworkLBBackends(d.Get("backend"))
	req.Ports = resourceLxdNetworkLBPorts(d.Get("port"))

	err = server.UpdateNetworkLoadBalancer(network, listenAddr, req, etag)
	if err != nil {
		return err
	}

	return resourceLxdNetworkLBRead(d, meta)
}

func resourceLxdNetworkLBDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	network := d.Get("network").(string)
	listenAddr := d.Get("listen_address").(string)

	err = server.DeleteNetworkLoadBalancer(network, listenAddr)
	if err != nil && isNotFoundError(err) {
		err = nil
	}

	return err
}

func resourceLxdNetworkLBExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	p := meta.(*lxdProvider)
	server, err := p.GetInstanceServer(p.selectRemote(d))
	if err != nil {
		return false, err
	}

	if v, ok := d.GetOk("project"); ok && v != "" {
		project := v.(string)
		server = server.UseProject(project)
	}

	network := d.Get("network").(string)
	listenAddr := d.Get("listen_address").(string)

	_, _, err = server.GetNetworkLoadBalancer(network, listenAddr)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func resourceLxdNetworkLBBackends(d interface{}) []api.NetworkLoadBalancerBackend {
	rawBackends := d.(*schema.Set).List()
	backends := make([]api.NetworkLoadBalancerBackend, 0, len(rawBackends))

	for _, v := range rawBackends {
		b := v.(map[string]interface{})

		backend := api.NetworkLoadBalancerBackend{}
		backend.Name = b["name"].(string)
		backend.Description = b["description"].(string)
		backend.TargetPort = b["target_port"].(string)
		backend.TargetAddress = b["target_address"].(string)

		backends = append(backends, backend)
	}

	log.Printf("[DEBUG] LXD Backends: %#v", backends)

	return backends
}

func resourceLxdNetworkLBPorts(d interface{}) []api.NetworkLoadBalancerPort {
	rawPorts := d.(*schema.Set).List()
	ports := make([]api.NetworkLoadBalancerPort, 0, len(rawPorts))

	for _, v := range rawPorts {
		p := v.(map[string]interface{})

		port := api.NetworkLoadBalancerPort{}
		port.Description = p["description"].(string)
		port.ListenPort = p["listen_port"].(string)
		port.Protocol = p["protocol"].(string)

		backends, ok := p["target_backend"].([]interface{})
		if ok {
			portBackends := make([]string, 0, len(backends))
			for _, b := range backends {
				portBackends = append(portBackends, b.(string))
			}

			port.TargetBackend = portBackends
		}

		ports = append(ports, port)
	}

	log.Printf("[DEBUG] LXD Ports: %#v", ports)

	return ports
}
