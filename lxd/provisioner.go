package lxd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/armon/circbuf"
	"github.com/mitchellh/go-linereader"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	lxd "github.com/lxc/lxd/client"
	lxd_config "github.com/lxc/lxd/lxc/config"
	lxd_api "github.com/lxc/lxd/shared/api"
)

const (
	// maxBufSize limits how much output we collect from a local
	// invocation. This is to prevent TF memory usage from growing
	// to an enormous amount due to a faulty process.
	maxBufSize = 8 * 1024
)

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		ConnSchema: map[string]*schema.Schema{
			"remote": {
				Type:     schema.TypeString,
				Required: true,
			},

			"address": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"config_dir": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: func() (interface{}, error) {
					return os.ExpandEnv("$HOME/.config/lxc"), nil
				},
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"port": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"scheme": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},

		Schema: map[string]*schema.Schema{
			"inline": &schema.Schema{
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				PromoteSingle: true,
				Required:      true,
			},

			"interpreter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/bin/sh",
			},
		},

		ApplyFunc: applyFn,
	}
}

// Apply executes the lxd exec provisioner
func applyFn(ctx context.Context) error {
	connData := ctx.Value(schema.ProvConnDataKey).(*schema.ResourceData)
	d := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)

	name := connData.Get("name").(string)
	remote := connData.Get("remote").(string)

	log.Printf("connData: %#v", connData)

	// Get the commands to run.
	var lines []string
	for _, l := range d.Get("inline").([]interface{}) {
		line, ok := l.(string)
		if !ok {
			return fmt.Errorf("Error parsing %v as a string", l)
		}
		lines = append(lines, line)
	}

	// Connect to the LXD remote.
	client, err := provisionerConfigureClient(connData)
	if err != nil {
		return fmt.Errorf("unable to obtain LXD client: %s", err)
	}

	// Set up the output.
	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to initialize pipe for output: %s", err)
	}

	output, _ := circbuf.NewBuffer(maxBufSize)
	tee := io.TeeReader(pr, output)

	copyDoneCh := make(chan struct{})
	go copyOutput(o, tee, copyDoneCh)

	// Get the interpreter
	cmd := d.Get("interpreter").(string)

	// Issue the requests.
	for _, line := range lines {
		o.Output(fmt.Sprintf("Executing %s on %s:%s", line, remote, name))

		args := lxd.ContainerExecArgs{
			Stdin:    ioutil.NopCloser(bytes.NewReader(nil)),
			Stderr:   pw,
			Stdout:   pw,
			DataDone: make(chan bool),
			Control:  nil,
		}

		req := lxd_api.ContainerExecPost{
			Command:     []string{cmd, "-c", line},
			WaitForWS:   true,
			Interactive: false,
		}

		op, err := client.ExecContainer(name, req, &args)
		if err != nil {
			return fmt.Errorf("failed to execute commands: %s", err)
		}

		// Wait for completion.
		err = op.Wait()
		if err != nil {
			return fmt.Errorf("failed to complete: %s", err)
		}

		<-args.DataDone
	}

	pw.Close()

	select {
	case <-copyDoneCh:
	case <-ctx.Done():
	}

	if err != nil {
		return fmt.Errorf("Error running commands on %s:%s: %s", remote, name, err)
	}

	return nil
}

func copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

func provisionerConfigureClient(d *schema.ResourceData) (lxd.ContainerServer, error) {
	var config *lxd_config.Config

	addr := d.Get("address").(string)
	configDir := d.Get("config_dir").(string)
	port := d.Get("port").(string)
	password := d.Get("password").(string)
	remote := d.Get("remote").(string)
	scheme := d.Get("scheme").(string)

	configPath := os.ExpandEnv(path.Join(configDir, "config.yml"))
	if conf, err := lxd_config.LoadConfig(configPath); err != nil {
		config = &lxd_config.DefaultConfig
		config.ConfigDir = configDir
	} else {
		config = conf
	}

	daemonAddr := ""
	switch scheme {
	case "unix", "":
		daemonAddr = fmt.Sprintf("unix:%s", addr)
	case "https":
		daemonAddr = fmt.Sprintf("https://%s:%s", addr, port)
	}

	config.Remotes[remote] = lxd_config.Remote{Addr: daemonAddr}
	rclient, err := config.GetContainerServer(remote)
	if err != nil {
		return nil, err
	}

	if err := authenticateToLXDServer(rclient, remote, password); err != nil {
		return nil, err
	}

	return rclient, nil
}
