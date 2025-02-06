package podmanapi

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
)

// ptr is a helper function to return a pointer to a value
func ptr[T any](t T) *T {
	return &t
}

var timeout uint = 10

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
	Ports     []uint16 `json:"ports"`
	Networks  []string `json:"networks"`
	Exited    bool     `json:"exited"`
	ExitCode  int32    `json:"exit_code"`
	ExitedAt  int64    `json:"exited_at"`
	Status    string   `json:"status"`
}
type PodmanContainerStatus struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

func getMapKeys(m map[uint16][]string) []uint16 {
	keys := make([]uint16, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func ListPodmanContainers(ctx context.Context) ([]PodmanContainer, error) {
	fmt.Println("Listing containers...")
	ctrList, err := containers.List(ctx, &containers.ListOptions{})
	if err != nil {
		return nil, err
	}
	var ctrStatusList []PodmanContainer
	for _, ctr := range ctrList {
		if err != nil {
			return nil, err
		}

		// get keys from ExposedPorts map as Ports list
		ctrStatusList = append(ctrStatusList, PodmanContainer{
			ID:        ctr.ID,
			Image:     ctr.Image,
			Names:     ctr.Names,
			State:     ctr.State,
			StartedAt: ctr.StartedAt,
			Ports:     getMapKeys(ctr.ExposedPorts),
			Networks:  ctr.Networks,
			Exited:    ctr.Exited,
			ExitCode:  ctr.ExitCode,
			ExitedAt:  ctr.ExitedAt,
			Status:    ctr.Status,
		})
	}

	return ctrStatusList, nil
}

func StartPodmanContainer(ctx context.Context, containerID string) (PodmanContainerStatus, error) {
	fmt.Println("Starting container...")
	err := containers.Start(ctx, containerID, &containers.StartOptions{})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	_, err = containers.Wait(ctx, containerID, &containers.WaitOptions{
		Condition: []define.ContainerStatus{define.ContainerStateRunning},
	})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	ctrData, err := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	return PodmanContainerStatus{
		ID:    containerID,
		State: ctrData.State.Status,
	}, nil
}

func StopPodmanContainer(ctx context.Context, containerID string) (PodmanContainerStatus, error) {
	fmt.Println("Stopping container...")

	err := containers.Stop(ctx, containerID, &containers.StopOptions{
		Ignore:  ptr(false),
		Timeout: &timeout,
	})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	ret := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() {
		_, err = containers.Wait(ctx, containerID, &containers.WaitOptions{
			Condition: []define.ContainerStatus{define.ContainerStateRunning},
		})
		if err != nil {
			fmt.Println(err)
		}
		ret <- true
	}()

	select {
	case <-ret:
		break
	case <-ctx.Done():
		return PodmanContainerStatus{}, fmt.Errorf("Timeout waiting for container to stop")
	}

	if err != nil {
		return PodmanContainerStatus{}, err
	}

	ctrData, err := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	return PodmanContainerStatus{
		ID:    containerID,
		State: ctrData.State.Status,
	}, nil
}
