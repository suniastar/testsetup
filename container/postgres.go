package container

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/4ND3R50N/testsetup"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

type postgres struct {
	hostName string
	Port     int
	Opts     testsetup.DockerContainerOpts
	r        *dockertest.Resource
}

type PostgresContainerOpts struct {
	ContainerName string
	NetworkID     string
	DBName        string
	DBUser        string
	DBPass        string
	// ExternalDBHost is the hostname the container is reachable at.
	// For DinD environments this is "docker", for local testing it
	// is "localhost". If empty "docker" will be set if running in
	// a CI environment and "localhost" otherwise.
	ExternalDBHost string
	DBExternalPort string
	DBInternalPort string
}

// WithPostgres returns a Container in order to spawn a postgres container
func WithPostgres(opts PostgresContainerOpts) testsetup.Container {
	opts.ExternalDBHost = validateHost(opts.ExternalDBHost)
	port, _ := strconv.Atoi(opts.DBExternalPort)
	return &postgres{
		Port: port,
		Opts: testsetup.DockerContainerOpts{
			ContainerName: opts.ContainerName,
			Repository:    "postgres",
			Tag:           "13.1",
			PortBinding:   map[string]string{opts.DBExternalPort: opts.DBInternalPort},
			Env: map[string]string{
				"POSTGRES_DB":       opts.DBName,
				"POSTGRES_PASSWORD": opts.DBPass,
				"POSTGRES_USER":     opts.DBUser,
				"POSTGRES_PORT":     opts.DBInternalPort,
			},
			ExpireTime: 5,
			HealthCheck: func(pool *dockertest.Pool, _ *dockertest.Resource) error {
				if err := waitForPostgres(pool,
					opts.ExternalDBHost,
					opts.DBExternalPort,
					opts.DBName,
					opts.DBUser,
					opts.DBPass,
					"disable"); err != nil {
					return err
				}
				return nil
			},
			NetworkID: opts.NetworkID,
		},
	}
}

func waitForPostgres(pool *dockertest.Pool,
	dbHost string,
	dbPort string,
	dbName string,
	dbUser string,
	dbPass string,
	dbSSLMode string) error {
	err := pool.Retry(func() error {
		dbClient, err := sql.Open("postgres",
			fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
				dbHost, dbPort, dbUser, dbName, dbPass, dbSSLMode))
		if err != nil {
			return err
		}
		if err := dbClient.Ping(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *postgres) GetHostname() string {
	return p.hostName
}

func (p *postgres) GetPorts() []int {
	return []int{p.Port}
}

func (p *postgres) Start(_ docker.AuthConfiguration, pool *dockertest.Pool) error {
	resource, hostname, err := testsetup.RunDockerContainer(docker.AuthConfiguration{}, pool, p.Opts)
	if err != nil {
		return err
	}
	p.hostName = *hostname
	p.r = resource
	return nil
}

func (p *postgres) Stop() error {
	return p.r.Close()
}

func (p *postgres) SetLabel(label map[string]string) {
	p.Opts.Labels = label
}
