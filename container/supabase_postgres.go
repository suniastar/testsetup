package container

import (
	"strconv"

	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/suniastar/testsetup"
)

type supabasePostgres struct {
	hostName string
	Port     int
	Opts     testsetup.DockerContainerOpts
	r        *dockertest.Resource
}

type SupabasePostgresContainerOpts struct {
	ContainerName string
	NetworkID     string
	DBName        string
	DBPass        string
	// ExternalDBHost is the hostname the container is reachable at.
	// For DinD environments this is "docker", for local testing it
	// is "localhost". If empty "docker" will be set if running in
	// a CI environment and "localhost" otherwise.
	ExternalDBHost string
	DBExternalPort string
	DBInternalPort string
}

func WithSupabasePostgres(opts SupabasePostgresContainerOpts) testsetup.Container {
	opts.ExternalDBHost = validateHost(opts.ExternalDBHost)
	port, _ := strconv.Atoi(opts.ExternalDBHost)
	return &supabasePostgres{
		Port: port,
		Opts: testsetup.DockerContainerOpts{
			ContainerName: opts.ContainerName,
			Repository:    "supabase/postgres",
			Tag:           "15.6.1.121",
			NetworkID:     opts.NetworkID,
			PortBinding:   map[string]string{opts.DBExternalPort: opts.DBInternalPort},
			Env: map[string]string{
				"POSTGRES_PORT":     opts.DBInternalPort,
				"PGPORT":            opts.DBInternalPort,
				"POSTGRES_DB":       opts.DBName,
				"PGDATABASE":        opts.DBName,
				"POSTGRES_PASSWORD": opts.DBPass,
				"PGPASSWORD":        opts.DBPass,
			},
			ExpireTime: 5,
			HealthCheck: func(pool *dockertest.Pool, resource *dockertest.Resource) error {
				if err := waitForPostgres(pool,
					opts.ExternalDBHost,
					opts.DBExternalPort,
					opts.DBName,
					"postgres",
					opts.DBPass,
					"disable"); err != nil {
					return err
				}
				return nil
			},
		},
	}
}

func (s *supabasePostgres) GetHostname() string {
	return s.hostName
}

func (s *supabasePostgres) GetPorts() []int {
	return []int{s.Port}
}

func (s *supabasePostgres) Start(auth docker.AuthConfiguration, pool *dockertest.Pool) error {
	resource, hostname, err := testsetup.RunDockerContainer(auth, pool, s.Opts)
	if err != nil {
		return err
	}
	s.hostName = *hostname
	s.r = resource
	return nil
}

func (s *supabasePostgres) Stop() error {
	return s.r.Close()
}

func (s *supabasePostgres) SetLabel(label map[string]string) {
	s.Opts.Labels = label
}
