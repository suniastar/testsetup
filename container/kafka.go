package container

import (
	"math/rand"

	"strconv"

	"github.com/4ND3R50N/testsetup"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	kafkaClient "github.com/segmentio/kafka-go"
)

type kafka struct {
	topics        []string
	hostName      string
	port          string
	kafkaInitPort string
	Opts          testsetup.DockerContainerOpts
	r             *dockertest.Resource
}

// KafkaOpts configures the kafka container
type KafkaOpts struct {
	ContainerName string
	// ContainerNamePort accepts connections in combination with ContainerName.
	ContainerNamePort string

	// ExternalHostName is the only accepted DNS name if you want to connect from the outside.
	// For DinD environments this is "docker", for local testing it is "localhost". If empty
	// "docker" will be set if running in a CI environment and "localhost" otherwise.
	ExternalHostName string
	// ExternalPort accepts connections in combination with ExternalHostName.
	ExternalPort      string
	ZookeeperHostName string
	ZookeeperPort     string
	NetworkID         string
}

// WithKafka returns a Container in order to spawn a kafka container
// it can be deployed with zookeeper (recommended) to use monitoring tools
func WithKafka(opts KafkaOpts, topics ...string) testsetup.Container {
	opts.ExternalHostName = validateHost(opts.ExternalHostName)
	kafkaInitConnectPort := strconv.Itoa(29000 + rand.Intn(100))
	env := map[string]string{
		"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":           "PLAINTEXT:PLAINTEXT,PLAINTEXT_DOCKER:PLAINTEXT,INTERNAL:PLAINTEXT",
		"KAFKA_LISTENERS":                                "PLAINTEXT://:9092,PLAINTEXT_DOCKER://:" + opts.ContainerNamePort + ",INTERNAL://:" + kafkaInitConnectPort,
		"KAFKA_ADVERTISED_LISTENERS":                     "PLAINTEXT://" + opts.ExternalHostName + ":" + opts.ExternalPort + ",PLAINTEXT_DOCKER://" + opts.ContainerName + ":" + opts.ContainerNamePort + ",INTERNAL://" + opts.ContainerName + ":" + kafkaInitConnectPort,
		"KAFKA_INTER_BROKER_LISTENER_NAME":               "INTERNAL",
		"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR":         "1",
		"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":            "1",
		"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR": "1",
	}
	if opts.ZookeeperHostName != "" {
		env["KAFKA_ZOOKEEPER_CONNECT"] = opts.ZookeeperHostName + ":" + opts.ZookeeperPort
	}
	kafkaContainer := kafka{
		hostName:      opts.ContainerName,
		topics:        topics,
		port:          opts.ExternalPort,
		kafkaInitPort: kafkaInitConnectPort,
		Opts: testsetup.DockerContainerOpts{
			Repository:    "confluentinc/cp-kafka",
			ContainerName: opts.ContainerName,
			Tag:           "7.2.1",
			PortBinding:   map[string]string{opts.ExternalPort: "9092"},
			Env:           env,
			ExpireTime:    5,
			HealthCheck: func(pool *dockertest.Pool, _ *dockertest.Resource) error {
				if err := kafkaHealthCheck(pool, opts.ExternalHostName, opts.ExternalPort); err != nil {
					return err
				}
				return nil
			},
			NetworkID: opts.NetworkID,
		},
	}
	return &kafkaContainer
}

func (k *kafka) GetHostname() string {
	return k.hostName
}

func (k *kafka) GetPorts() []int {
	port, _ := strconv.Atoi(k.port)
	return []int{port}
}

func (k *kafka) Start(_ docker.AuthConfiguration, pool *dockertest.Pool) error {
	auth := docker.AuthConfiguration{}
	resource, hostname, err := testsetup.RunDockerContainer(auth, pool, k.Opts)
	if err != nil {
		return err
	}
	k.hostName = *hostname
	k.r = resource
	if len(k.topics) > 0 {
		err := initKafka(auth, pool, *k)
		if err != nil {
			return err
		}
	}
	return nil
}

func initKafka(auth docker.AuthConfiguration, pool *dockertest.Pool, k kafka) error {
	command := "kafka-topics --bootstrap-server " + k.hostName + ":" + k.kafkaInitPort + " --list"
	for _, topic := range k.topics {
		command += " && "
		command += "kafka-topics --bootstrap-server " +
			k.hostName +
			":" + k.kafkaInitPort + " --create --if-not-exists --topic " + topic + " --replication-factor 1 --partitions 1"
	}
	kafkaInit := kafka{
		Opts: testsetup.DockerContainerOpts{
			Repository: "confluentinc/cp-kafka",
			Tag:        "7.2.1",
			ExpireTime: 5,
			EntryPoint: []string{"/bin/sh", "-c"},
			Commands:   []string{command},
			HealthCheck: func(pool *dockertest.Pool, resource *dockertest.Resource) error {
				_, err := pool.Client.WaitContainer(resource.Container.ID)
				if err != nil {
					return err
				}
				return nil
			},
			NetworkID: k.Opts.NetworkID,
			Labels: map[string]string{
				"testsetup": "kafka-init",
			},
		},
	}
	_, _, err := testsetup.RunDockerContainer(auth, pool, kafkaInit.Opts)
	if err != nil {
		return err
	}
	return nil
}

func (k *kafka) Stop() error {
	return k.r.Close()
}

func (k *kafka) SetLabel(label map[string]string) {
	k.Opts.Labels = label
}

func kafkaHealthCheck(pool *dockertest.Pool, host string, port string) error {
	if err := pool.Retry(func() error {
		conn, err := kafkaClient.Dial("tcp", host+":"+port)
		if err != nil {
			return err
		}
		_, err = conn.Brokers()
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
