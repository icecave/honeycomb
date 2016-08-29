package di

import "github.com/docker/engine-api/client"

// DockerClient returns the docker client used to access the swarm.
func (con *Container) DockerClient() client.APIClient {
	return con.get(
		"docker.client",
		func() (interface{}, error) {
			return client.NewEnvClient()
		},
		nil,
	).(client.APIClient)
}
