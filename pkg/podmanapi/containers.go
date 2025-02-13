package podmanapi

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/sonarping/go-nodeapi/pkg/utils"
)

var timeout uint = 10
var Podmanctx context.Context

type Container struct {
	ID string
}

func InitPodmanConnection() (context.Context, error) {
	sock_dir := os.Getenv("XDG_RUNTIME_DIR")
	if sock_dir == "" {
		sock_dir = "/var/run"
	}
	socket := "unix:" + sock_dir + "/podman/podman.sock"

	if Podmanctx != nil {
		return Podmanctx, nil
	}

	Podmanctx, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return Podmanctx, nil
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

	contData, testContainerErr := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
	if testContainerErr != nil {
		return PodmanContainerStatus{}, testContainerErr
	}

	if contData.State.Status == define.ContainerStateRunning.String() {
		return PodmanContainerStatus{}, fmt.Errorf("Container is already running")
	}

	err := containers.Start(ctx, containerID, &containers.StartOptions{})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	ret := make(chan bool)
	startContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	case <-startContext.Done():
		return PodmanContainerStatus{}, fmt.Errorf("Timeout waiting for container to start")
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

	contData, testContainerErr := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
	if testContainerErr != nil {
		return PodmanContainerStatus{}, testContainerErr
	}

	if contData.State.Status == define.ContainerStateStopped.String() {
		return PodmanContainerStatus{}, fmt.Errorf("Container is already stopped")
	}

	err := containers.Stop(ctx, containerID, &containers.StopOptions{
		Ignore:  utils.GetPtr(false),
		Timeout: &timeout,
	})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	ret := make(chan bool)
	stopContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() {
		_, err = containers.Wait(ctx, containerID, &containers.WaitOptions{
			Condition: []define.ContainerStatus{define.ContainerStateStopped},
		})
		if err != nil {
			fmt.Println(err)
		}
		ret <- true
	}()

	select {
	case <-ret:
		break
	case <-stopContext.Done():
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

func CreateFromImage(ctx context.Context, imageName string, containerName string) (string, error) {
	fmt.Println("Creating container...")
	spec := new(specgen.SpecGenerator)
	spec.Name = containerName
	spec.Image = imageName
	// basic container settings

	ctrData, err := containers.CreateWithSpec(ctx, spec, nil)
	if err != nil {
		return "", err
	}

	return ctrData.ID, nil
}

func InspectNetwork(ctx context.Context, containerID string) (map[string]string, error) {
	inspectData, err := containers.Inspect(ctx, containerID, nil)
	if err != nil {
		return nil, err
	}

	networks := make(map[string]string, 0)

	if inspectData.NetworkSettings != nil && inspectData.NetworkSettings.Networks != nil {
		for networkName, networkData := range inspectData.NetworkSettings.Networks {
			if networkData.IPAddress != "" {
				fmt.Printf("Container IP Address: %s\n", networkData.IPAddress)
				networks[networkName] = networkData.IPAddress
			}
		}
	} else if inspectData.NetworkSettings != nil {
		fmt.Printf("Container IP Address: %s\n", inspectData.NetworkSettings.IPAddress)
		networks["default"] = inspectData.NetworkSettings.IPAddress
	} else {
		fmt.Println("No network settings found for container")
		return nil, nil
	}
	return networks, nil
}
