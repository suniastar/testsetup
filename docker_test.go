package testsetup_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/stretchr/testify/assert"
	"github.com/suniastar/testsetup"
	"github.com/suniastar/testsetup/container"
)

func TestDocker_RunDockerContainer(t *testing.T) {
	pool, err := dockertest.NewPool("")
	assert.NoError(t, err)
	networkID := "TestDocker_RunDockerContainer" + uuid.New().String()
	network, _ := testsetup.CreateNetwork(pool, networkID)
	defer func(pool *dockertest.Pool, name string) { _ = testsetup.RemoveNetwork(pool, name) }(pool, networkID)
	assert.NoError(t, err)
	opts := testsetup.DockerContainerOpts{
		ContainerName: "postgres-container",
		Repository:    "postgres",
		Tag:           "13.1",
		PortBinding:   map[string]string{"5431": "5432"},
		Env: map[string]string{
			"POSTGRES_DB":       "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_USER":     "test",
		},
		HealthCheck: func(pool *dockertest.Pool, _ *dockertest.Resource) error {
			if err := waitForPostgres(pool,
				container.AutoGuessHostname(),
				"5431",
				"test",
				"test",
				"test",
				"disable"); err != nil {
				return err
			}
			return nil
		},
		ExpireTime: 5,
		NetworkID:  network.ID,
	}
	resource, podName, err := testsetup.RunDockerContainer(docker.AuthConfiguration{}, pool, opts)
	assert.NoError(t, err)
	assert.Equal(t, opts.ContainerName, *podName)
	err = resource.Close()
	assert.NoError(t, err)
}

func TestDocker_RunDockerContainerHealthCheckFails(t *testing.T) {
	pool, err := dockertest.NewPool("")
	assert.NoError(t, err)
	opts := testsetup.DockerContainerOpts{
		Repository:  "postgres",
		Tag:         "15",
		PortBinding: map[string]string{"5432": "5432"},
		Env: map[string]string{
			"POSTGRES_DB":       "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_USER":     "test",
		},
		HealthCheck: func(pool *dockertest.Pool, _ *dockertest.Resource) error {
			if err := waitForPostgres(pool,
				"192.0.0.1",
				"5432",
				"network",
				"test",
				"some other password that is incorrect to cause this healthcheck failing",
				"disable"); err != nil {
				return err
			}
			return nil
		},
		ExpireTime: 5,
	}
	_, _, err = testsetup.RunDockerContainer(docker.AuthConfiguration{}, pool, opts)
	assert.Error(t, err)
}

func TestDocker_RunDockerContainerInvalidImage(t *testing.T) {
	pool, err := dockertest.NewPool("")
	assert.NoError(t, err)
	opts := testsetup.DockerContainerOpts{
		Repository:  "hey-how-are-u",
		Tag:         "15",
		PortBinding: map[string]string{"5432": "5432"},
		Env:         nil,
		HealthCheck: nil,
		ExpireTime:  5,
	}
	_, _, err = testsetup.RunDockerContainer(docker.AuthConfiguration{}, pool, opts)
	assert.Error(t, err)
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
