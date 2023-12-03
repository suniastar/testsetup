package container

import (
	"github.com/4ND3R50N/testsetup"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

type zookeeper struct {
	hostName string
	Opts     testsetup.DockerContainerOpts
	r        *dockertest.Resource
}

type ZookeeperOpts struct {
	Port          string
	NetworkID     string
	ContainerName string
}

// WithZookeeper returns a container in order to spawn a zookeeper
func WithZookeeper(opts ZookeeperOpts) testsetup.Container {
	return &zookeeper{
		Opts: testsetup.DockerContainerOpts{
			Repository:    "confluentinc/cp-zookeeper",
			ContainerName: opts.ContainerName,
			Tag:           "7.3.1",
			PortBinding:   map[string]string{opts.Port: "2181"},
			Env: map[string]string{
				"ZOOKEEPER_CLIENT_PORT": opts.Port,
				"ZOOKEEPER_TICK_TIME":   "2000",
			},
			ExpireTime: 5,
			HealthCheck: func(pool *dockertest.Pool, _ *dockertest.Resource) error {
				return nil
			},
			NetworkID: opts.NetworkID,
		},
	}
}

func (z *zookeeper) GetHostname() string {
	return z.hostName
}

func (z *zookeeper) GetPorts() []int {
	return []int{}
}

func (z *zookeeper) Start(_ docker.AuthConfiguration, pool *dockertest.Pool) error {
	resource, hostname, err := testsetup.RunDockerContainer(docker.AuthConfiguration{}, pool, z.Opts)
	if err != nil {
		return err
	}
	z.hostName = *hostname
	z.r = resource
	return nil
}

func (z *zookeeper) Stop() error {
	return z.r.Close()
}

func (z *zookeeper) SetLabel(label map[string]string) {
	z.Opts.Labels = label
}
