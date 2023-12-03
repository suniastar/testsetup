# Test Setup
This library contains useful (integration) test functionality.

## Usage
### Start single container
This example is with postgres
````go
opts := testsetup.DockerContainerOpts{
    ContainerName: "postgres-container",
    Repository:    "postgres",
    Tag:           "15",
    ExposedPorts:  []string{"5432", "5430"},
    Env: map[string]string{
        "POSTGRES_DB":       "test",
        "POSTGRES_PASSWORD": "test",
        "POSTGRES_USER":     "test",
    },
    HealthCheck: func(pool *dockertest.Pool, exposedPorts []string, env map[string]string) error {
        // Perform your checks
        return nil
    },
    ExpireTime: 5,
    NetworkID:  network.ID,
}
resource, podName, err := testsetup.RunDockerContainer(docker.AuthConfiguration{}, pool, opts)
````
Its also possible to provide authentication for private repositories by extending `testsetup.DockerContainerOpts{}`
with the `auth` parameter.

### Use the test suite with pre-defined container
````go
networkID := "TestTestSetup_Start-" + uuid.New().String()
zookeeper := container.ZookeeperOpts{
    ContainerName: "my-zookeeper",
    Port:          "2181",
    NetworkID:     networkID,
}

kafka := container.KafkaOpts{
    ContainerName:     "my-kafka",
    Port:              "9092",
    ZookeeperHostName: zookeeperContainerName,
    ZookeeperPort:     "2181",
    NetworkID:         networkID,
}

testSetup := testsetup.NewTestSetup(docker.AuthConfiguration{},
    networkID,
    container.WithZookeeper(zookeeper),
    container.WithKafka(kafka, "your.topic"),
)
testSetup.Start()
testSetup.WaitUntilStarted()
testSetup.Stop()

````
The test setup provides an Auth Parameter which is necessary if you want to pull images from private repositories.

Available pre-defined container:
- Kafka (+ Init Kafka)
- Postgres
- Zookeeper

#### For MAC users
If you use [colima](https://github.com/abiosoft/colima), you have to create a symlink to make it run on MacOS:

```bash
sudo ln -sf $HOME/.colima/default/docker.sock /var/run/docker.sock
```

### Hostnames in different environments

Depending on your environment the hostname of the containers might be different. Use `container.AutoGuessHostname()` to get
the applicable hostname for your environment.
