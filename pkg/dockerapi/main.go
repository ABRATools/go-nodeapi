package dockerapi

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// functions to implement:
// [x] start
// [x] stop
// [x] list containers
// [x] inspect container

// not done yet

type DockerContainer struct {
	id     string
	status string
	cpu    container.CPUUsage
	memory container.MemoryStats
}

func ListContainer() []DockerContainer {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		panic(err)
	}

	var dockerContainers []DockerContainer
	if len(containers) > 0 {
		for _, type_container := range containers {
			container, err := cli.ContainerStats(context.Background(), type_container.ID, false)
			dockerContainers = append(dockerContainers, DockerContainer{
				id:     type_container.ID,
				status: type_container.State,
				cpu:    container.CPUUsage,
				memory: container.MemoryStats,
			})
		}
	} else {
		fmt.Println("There are no containers running")
	}
	return nil
}

func StopDocker(containerID string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	options := types.ContainerStopOptions{}
	err = cli.ContainerStop(context.Background(), containerID, nil)
	if err != nil {
		panic(err)
	}
	return err
}
