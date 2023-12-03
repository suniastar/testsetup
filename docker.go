package testsetup

import (
	"fmt"
	"strings"
	"time"

	// necessary for sql
	_ "github.com/lib/pq"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

type DockerContainerAuthOpts struct {
	Username  string
	Password  string
	ServerURL string
	Email     string
}

type DockerContainerOpts struct {
	Repository    string
	ContainerName string
	Tag           string

	// PortBinding looks like this: {"5431": "5432"}
	// Key: port to access from outside, Value: port to expose
	PortBinding map[string]string

	NetworkID   string
	Env         map[string]string // key: env var name, Value: value
	Commands    []string          // Don´t set this option if there are no commands to perform!
	EntryPoint  []string          // Don´t set this option if there are no entry points to change!
	Labels      map[string]string
	ExpireTime  time.Duration
	HealthCheck func(pool *dockertest.Pool, resource *dockertest.Resource) error
}

// CreateNetwork creates a docker network used so container can communicate with each other
func CreateNetwork(pool *dockertest.Pool, name string) (*docker.Network, error) {
	nw, err := pool.Client.CreateNetwork(docker.CreateNetworkOptions{Name: name})
	if err != nil {
		return nil, err
	}
	return nw, nil
}

// RemoveNetwork removes a network by id
func RemoveNetwork(pool *dockertest.Pool, name string) error {
	if err := pool.Retry(func() error {
		if err := pool.Client.RemoveNetwork(name); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// RunDockerContainer can run any docker container.
// the cleanup function can be called to shut down the container
// the hostname can be used to access the pod from in a docker network
func RunDockerContainer(auth docker.AuthConfiguration, pool *dockertest.Pool, opts DockerContainerOpts) (
	r *dockertest.Resource,
	hostname *string,
	err error) {

	var envList []string
	for key, value := range opts.Env {
		envList = append(envList, key+"="+value)
	}

	portBindings := make(map[docker.Port][]docker.PortBinding, len(opts.PortBinding))
	for portToReach, portToExpose := range opts.PortBinding {
		portBindings[docker.Port(portToExpose+"/tcp")] = []docker.PortBinding{
			{
				HostIP:   opts.ContainerName,
				HostPort: portToReach,
			},
		}
	}

	runDockerOpt := &dockertest.RunOptions{
		Auth:         auth,
		Name:         opts.ContainerName,
		Repository:   opts.Repository,
		Tag:          opts.Tag,
		Env:          envList,
		Cmd:          opts.Commands,
		Entrypoint:   opts.EntryPoint,
		PortBindings: portBindings,
		NetworkID:    opts.NetworkID,
		Labels:       opts.Labels,
	}

	fnConfig := func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.NeverRestart()
	}
	resource, err := pool.RunWithOptions(runDockerOpt, fnConfig)
	if err != nil {
		return nil, nil, err
	}
	if err := resource.Expire(uint(time.Minute * opts.ExpireTime)); err != nil {
		return nil, nil, err
	}

	if err := opts.HealthCheck(pool, resource); err != nil {
		_ = resource.Close()
		return nil, nil, fmt.Errorf("waited too long for docker health check: %s", err.Error())
	}
	domainName := strings.Trim(resource.Container.Name, "/")

	return resource, &domainName, nil
}
