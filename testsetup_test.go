package testsetup_test

import (
	"context"
	"github.com/segmentio/kafka-go"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/4ND3R50N/testsetup"
	"github.com/4ND3R50N/testsetup/container"
	"github.com/google/uuid"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestSetup_Start(t *testing.T) {
	networkID := "TestTestSetup_Start-" + uuid.New().String()
	auth := docker.AuthConfiguration{
		Username:      os.Getenv("CI_REGISTRY_USER"),
		Password:      os.Getenv("CI_REGISTRY_PASSWORD"),
		Email:         os.Getenv("CI_REGISTRY_EMAIL"),
		ServerAddress: os.Getenv("CI_REGISTRY"),
	}
	postgresContainerName := "postgres-" + uuid.New().String()
	postgres := container.PostgresContainerOpts{
		ContainerName:  postgresContainerName,
		NetworkID:      networkID,
		DBName:         "network",
		DBUser:         "test",
		DBPass:         "test",
		ExternalDBHost: container.AutoGuessHostname(),
		DBExternalPort: "5431",
		DBInternalPort: "5432",
	}

	zookeeperContainerName := "zookeeper-" + uuid.New().String()
	zookeeper := container.ZookeeperOpts{
		ContainerName: zookeeperContainerName,
		Port:          "2181",
		NetworkID:     networkID,
	}

	kafkaContainerName := "kafka-" + uuid.New().String()
	kafkaContainerOpts := container.KafkaOpts{
		ContainerName:     kafkaContainerName,
		ContainerNamePort: "9091",
		ExternalHostName:  container.AutoGuessHostname(),
		ExternalPort:      "9092",
		ZookeeperHostName: zookeeperContainerName,
		ZookeeperPort:     zookeeper.Port,
		NetworkID:         networkID,
	}
	testSetup := testsetup.NewTestSetup(auth,
		networkID,
		container.WithPostgres(postgres),
		container.WithZookeeper(zookeeper),
		container.WithKafka(kafkaContainerOpts, "hetzner.cloud.network.v1.changes",
			"hetzner.cloud.network.v1.changes.reply"))
	testSetup.Start()
	err := testSetup.WaitUntilStarted()
	require.NoError(t, err)

	// Start a second test setup with same containers to check port configuration.
	networkID2 := "TestTestSetup_Start-" + uuid.New().String()
	postgres.ContainerName = postgresContainerName + "-2"
	postgres.DBExternalPort = "5433"

	zookeeper.ContainerName = zookeeperContainerName + "-2"
	zookeeper.Port = "2182"

	kafkaContainerOpts.ContainerName = kafkaContainerName + "-2"
	kafkaContainerOpts.ContainerNamePort = "9094"
	kafkaContainerOpts.ExternalPort = "9093"
	kafkaContainerOpts.ZookeeperHostName = zookeeper.ContainerName
	kafkaContainerOpts.ZookeeperPort = zookeeper.Port

	testSetup2 := testsetup.NewTestSetup(auth,
		networkID2,
		container.WithPostgres(postgres),
		container.WithZookeeper(zookeeper),
		container.WithKafka(kafkaContainerOpts, "hetzner.cloud.network.v1.changes",
			"hetzner.cloud.network.v1.changes.reply"))
	testSetup2.Start()
	err = testSetup2.WaitUntilStarted()
	require.NoError(t, err)

	// Try connecting to both of them.
	reader1 := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{container.AutoGuessHostname() + ":9092"},
		Topic:   "hetzner.cloud.network.v1.changes",
	})
	reader2 := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{container.AutoGuessHostname() + ":9093"},
		Topic:   "hetzner.cloud.network.v1.changes",
	})

	ctxWithDeadline, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		msg, err := reader1.FetchMessage(ctxWithDeadline)
		assert.NoError(t, err)
		assert.Equal(t, `hello world from writer 1`, string(msg.Value))
		wg.Done()
	}()
	go func() {
		msg, err := reader2.FetchMessage(ctxWithDeadline)
		assert.NoError(t, err)
		assert.Equal(t, `hello world from writer 2`, string(msg.Value))
		wg.Done()
	}()

	writer1 := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{container.AutoGuessHostname() + ":9092"},
		Topic:        "hetzner.cloud.network.v1.changes",
		BatchSize:    1,
		BatchTimeout: time.Millisecond * 5,
	})
	writer2 := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{container.AutoGuessHostname() + ":9093"},
		Topic:        "hetzner.cloud.network.v1.changes",
		BatchSize:    1,
		BatchTimeout: time.Millisecond * 5,
	})
	assert.NoError(t, writer1.WriteMessages(context.Background(), kafka.Message{
		Value: []byte("hello world from writer 1"),
	}), "failed writing to kafka 1")
	assert.NoError(t, writer2.WriteMessages(context.Background(), kafka.Message{
		Value: []byte("hello world from writer 2"),
	}), "failed writing to kafka 2")
	wg.Wait()

	// Stop them.
	testSetup.Stop()
	testSetup2.Stop()
}

func TestTestSetup_TestPanicRecovery(t *testing.T) {
	networkID := "TestTestSetup_TestPanicRecovery-" + uuid.New().String()
	postgresContainerName := "postgres-" + uuid.New().String()
	postgres := container.PostgresContainerOpts{
		ContainerName:  postgresContainerName,
		NetworkID:      "wedfwfwefwef",
		DBName:         "network",
		DBUser:         "test",
		DBPass:         "test",
		DBExternalPort: "5431",
		DBInternalPort: "5432",
	}

	zookeeperContainerName := "zookeeper-" + uuid.New().String()
	zookeeper := container.ZookeeperOpts{
		ContainerName: zookeeperContainerName,
		Port:          "2181",
		NetworkID:     networkID,
	}

	// Spawn 2 containers.
	testSetup := testsetup.NewTestSetup(docker.AuthConfiguration{},
		networkID,
		container.WithZookeeper(zookeeper), container.WithPostgres(postgres))
	// The second container will panic, because the network does not exist.
	// Due to the recovery function, the whole process will not panic. Instead, all containers + the testsetuo network
	// gets deleted.
	testSetup.Start()

	// WaitUntilStarted will also abort due to the panic with a controlled error message that also contains the root
	// reason.
	err := testSetup.WaitUntilStarted()
	require.ErrorAs(t, err, &testsetup.ErrAborted)

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)
	containers, err := pool.Client.ListContainers(docker.ListContainersOptions{All: true})
	require.NoError(t, err)
	require.Len(t, containers, 0)
	_, err = pool.Client.NetworkInfo(networkID)
	require.Error(t, err)
}
