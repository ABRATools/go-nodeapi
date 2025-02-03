package podmanapi

import (
	"context"
	"fmt"
	"os"

	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
)

// functions to implement:
// [x] start
// [x] stop
// [x] list containers
// [x] inspect container

// ptr is a helper function to return a pointer to a value
func ptr[T any](t T) *T {
	return &t
}

type Container struct {
	ID string
}

func InitPodmanConnection() (context.Context, error) {
	sock_dir := os.Getenv("XDG_RUNTIME_DIR")
	if sock_dir == "" {
		sock_dir = "/var/run"
	}
	socket := "unix:" + sock_dir + "/podman/podman.sock"

	podmanctx, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return podmanctx, nil
}

type PodmanContainer struct {
	ID        string   `json:"id"`
	Image     string   `json:"image"`
	Names     []string `json:"names"`
	State     string   `json:"state"`
	StartedAt int64    `json:"started_at"`
}

type PodmanContainerStatus struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

func ListPodmanContainers(ctx context.Context) []PodmanContainer {
	fmt.Println("Listing containers...")
	ctrList, err := containers.List(ctx, &containers.ListOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var ctrStatusList []PodmanContainer
	for _, ctr := range ctrList {
		// ctrData, err := containers.Inspect(ctx, ctr.ID, &containers.InspectOptions{})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		ctrStatusList = append(ctrStatusList, PodmanContainer{
			ID:        ctr.ID,
			Image:     ctr.Image,
			Names:     ctr.Names,
			State:     ctr.State,
			StartedAt: ctr.StartedAt,
		})
	}

	return ctrStatusList
}

func StartPodmanContainer(ctx context.Context, containerID string) PodmanContainerStatus {
	fmt.Println("Starting container...")
	err := containers.Start(ctx, containerID, &containers.StartOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = containers.Wait(ctx, containerID, &containers.WaitOptions{
		Condition: []define.ContainerStatus{define.ContainerStateRunning},
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctrData, err := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return PodmanContainerStatus{
		ID:    containerID,
		State: ctrData.State.Status,
	}
}

func StopPodmanContainer(ctx context.Context, containerID string) PodmanContainerStatus {
	fmt.Println("Stopping container...")
	err := containers.Stop(ctx, containerID, &containers.StopOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = containers.Wait(ctx, containerID, &containers.WaitOptions{
		Condition: []define.ContainerStatus{define.ContainerStateRunning},
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctrData, err := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return PodmanContainerStatus{
		ID:    containerID,
		State: ctrData.State.Status,
	}
}
