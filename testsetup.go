package testsetup

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"log"
	"sync"
)

var (
	ErrAborted  = errors.New("test setup was erroneous")
	ErrNotReady = errors.New("still not ready")
)

type Container interface {
	GetHostname() string
	GetPorts() []int
	SetLabel(map[string]string)
	Start(auth docker.AuthConfiguration, pool *dockertest.Pool) error
	Stop() error
}

type TestSetup struct {
	testSetupID string
	aborted     error
	started     sync.Once
	stopped     sync.Once
	services    []Container
	network     *docker.Network
	pool        *dockertest.Pool
	auth        docker.AuthConfiguration
}

func NewTestSetup(auth docker.AuthConfiguration, networkID string, container ...Container) *TestSetup {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not create new pool: %s", err)
	}
	network, err := CreateNetwork(pool, networkID)
	if err != nil {
		log.Fatalf("Could not create network: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	testSetupID := "testSetup-" + uuid.New().String()
	for _, c := range container {
		c.SetLabel(map[string]string{
			testSetupID: "event-emitter",
		})
	}
	return &TestSetup{
		testSetupID: testSetupID,
		services:    container,
		network:     network,
		pool:        pool,
		auth:        auth,
		aborted:     nil,
	}
}

func (t *TestSetup) Start() {
	defer handleCleanupContainer(t)
	t.started.Do(func() {
		for _, service := range t.services {
			err := service.Start(t.auth, t.pool)
			if err != nil {
				panic(err)
			}
		}
	})
}

func handleCleanupContainer(t *TestSetup) {
	if r := recover(); r != nil {
		containers, _ := t.pool.Client.ListContainers(docker.ListContainersOptions{All: true})
		for _, container := range containers {
			_ = t.pool.Client.KillContainer(docker.KillContainerOptions{ID: container.ID})
		}
		_ = t.pool.Client.RemoveNetwork(t.network.ID)
		t.aborted = fmt.Errorf("recovered from panic: %v", r)
	}

}

func (t *TestSetup) Stop() {
	defer handleCleanupContainer(t)
	t.stopped.Do(func() {
		for _, service := range t.services {
			if err := service.Stop(); err != nil {
				panic("unable to stop a resource: " + err.Error())
			}
		}
		if err := RemoveNetwork(t.pool, t.network.ID); err != nil {
			panic("unable to delete network: " + err.Error())
		}
	})
}

func (t *TestSetup) WaitUntilStarted() error {
	err := t.pool.Retry(func() error {
		if t.aborted != nil {
			return errors.Join(ErrAborted, t.aborted)
		}
		c, _ := t.pool.Client.ListContainers(docker.ListContainersOptions{Filters: map[string][]string{
			"label": {t.testSetupID},
		}})
		if len(c) == len(t.services) {
			return nil
		}
		return ErrNotReady
	})
	if err != nil {
		return err
	}
	return nil
}
