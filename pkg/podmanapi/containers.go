// podmanapi.go
package podmanapi

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	nettypes "github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/sonarping/go-nodeapi/pkg/utils"
)

var timeout uint = 10
var Podmanctx context.Context

// Dependency injection variables for testing:
var (
	newConnectionFunc = bindings.NewConnection
	containersInspect = containers.Inspect
	containersList    = containers.List
	containersStats   = containers.Stats
	containersStart   = containers.Start
	containersStop    = containers.Stop
	containersWait    = containers.Wait
	containersCreate  = containers.CreateWithSpec
	containersRemove  = containers.Remove
)

type Container struct {
	ID string
}

func InitPodmanConnection() (context.Context, error) {
	sockDir := os.Getenv("XDG_RUNTIME_DIR")
	if sockDir == "" {
		sockDir = "/var/run"
	}
	socket := "unix:" + sockDir + "/podman/podman.sock"

	if Podmanctx != nil {
		return Podmanctx, nil
	}

	var err error
	Podmanctx, err = newConnectionFunc(context.Background(), socket)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return Podmanctx, nil
}

type PodmanContainer struct {
	ID            string   `json:"env_id"`
	Image         string   `json:"image"`
	Names         []string `json:"names"`
	State         string   `json:"state"`
	StartedAt     int64    `json:"started_at"`
	Ports         []uint16 `json:"ports"`
	IP            string   `json:"ip"`
	Networks      []string `json:"networks"`
	Exited        bool     `json:"exited"`
	ExitCode      int32    `json:"exit_code"`
	ExitedAt      int64    `json:"exited_at"`
	Status        string   `json:"status"`
	CPUPercentage float64  `json:"cpu_percentage"`
	MemoryPercent float64  `json:"memory_percent"`
	Uptime        int64    `json:"uptime"`
}

type PodmanContainerStatus struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

func ListPodmanContainers(ctx context.Context) ([]PodmanContainer, error) {
	fmt.Println("Listing containers...")
	ctrList, err := containersList(ctx, &containers.ListOptions{All: utils.GetPtr(true)})
	if err != nil {
		return nil, err
	}
	var ctrStatusList []PodmanContainer
	for _, ctr := range ctrList {
		// Retrieve IP
		ip, err := GetIPAddress(ctx, ctr.ID)
		if err != nil {
			ip = ""
		}
		var stats types.ContainerStatsReport
		if ctr.State == define.ContainerStateRunning.String() {
			// Get stats for running containers
			statsChan, err := containersStats(ctx, []string{ctr.ID}, &containers.StatsOptions{Stream: utils.GetPtr(false)})
			if err != nil {
				fmt.Println(err)
			} else {
				stats = <-statsChan
			}
		} else {
			stats = types.ContainerStatsReport{
				Stats: []define.ContainerStats{
					{
						CPU:     0,
						MemPerc: 0,
						UpTime:  0,
					},
				},
			}
		}

		ctrStatusList = append(ctrStatusList, PodmanContainer{
			ID:            ctr.ID,
			Image:         ctr.Image,
			Names:         ctr.Names,
			State:         ctr.State,
			StartedAt:     ctr.StartedAt,
			Ports:         utils.GetMapKeys(ctr.ExposedPorts),
			Networks:      ctr.Networks,
			IP:            ip,
			Exited:        ctr.Exited,
			ExitCode:      ctr.ExitCode,
			ExitedAt:      ctr.ExitedAt,
			Status:        ctr.Status,
			CPUPercentage: stats.Stats[0].CPU,
			MemoryPercent: stats.Stats[0].MemPerc,
			Uptime:        int64(stats.Stats[0].UpTime),
		})
	}

	return ctrStatusList, nil
}

func StartPodmanContainer(ctx context.Context, containerID string) (PodmanContainerStatus, error) {
	fmt.Println("Starting container...")

	contData, err := containersInspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	if contData.State.Status == define.ContainerStateRunning.String() {
		return PodmanContainerStatus{}, fmt.Errorf("Container is already running")
	}

	err = containersStart(ctx, containerID, &containers.StartOptions{})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	ret := make(chan bool)
	startContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() {
		_, err = containersWait(ctx, containerID, &containers.WaitOptions{
			Condition: []define.ContainerStatus{define.ContainerStateRunning},
		})
		if err != nil {
			fmt.Println(err)
		}
		ret <- true
	}()

	select {
	case <-ret:
	case <-startContext.Done():
		return PodmanContainerStatus{}, fmt.Errorf("Timeout waiting for container to start")
	}

	ctrData, err := containersInspect(ctx, containerID, &containers.InspectOptions{})
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

	contData, err := containersInspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	if contData.State.Status == define.ContainerStateStopped.String() {
		return PodmanContainerStatus{}, fmt.Errorf("Container is already stopped")
	}

	err = containersStop(ctx, containerID, &containers.StopOptions{
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
		_, err = containersWait(ctx, containerID, &containers.WaitOptions{
			Condition: []define.ContainerStatus{define.ContainerStateStopped},
		})
		if err != nil {
			fmt.Println(err)
		}
		ret <- true
	}()

	select {
	case <-ret:
	case <-stopContext.Done():
		return PodmanContainerStatus{}, fmt.Errorf("Timeout waiting for container to stop")
	}

	ctrData, err := containersInspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		return PodmanContainerStatus{}, err
	}

	return PodmanContainerStatus{
		ID:    containerID,
		State: ctrData.State.Status,
	}, nil
}

type Option func(*Config)

type Config struct {
	StaticIP net.IP
}

func WithStaticIP(ip net.IP) Option {
	return func(c *Config) {
		c.StaticIP = ip
	}
}

func CreateFromImage(ctx context.Context, imageName string, containerName string, create_opts ...Option) (string, error) {
	fmt.Println("Creating container...")
	config := &Config{}
	for _, opt := range create_opts {
		opt(config)
	}
	spec := new(specgen.SpecGenerator)
	spec.Name = containerName
	spec.Image = imageName

	if config.StaticIP != nil {
		spec.Networks = map[string]nettypes.PerNetworkOptions{
			"podman": {
				StaticIPs: []net.IP{config.StaticIP},
			},
		}
	}

	ctrData, err := containersCreate(ctx, spec, nil)
	if err != nil {
		return "", err
	}

	return ctrData.ID, nil
}

func RemovePodmanContainer(ctx context.Context, containerID string) error {
	inspectData, err := containersInspect(ctx, containerID, nil)
	if err != nil {
		return err
	}
	if inspectData.State.Status == define.ContainerStateRunning.String() {
		fmt.Println("Stopping container first...")
		_, err := StopPodmanContainer(ctx, containerID)
		if err != nil {
			return err
		}
	}

	fmt.Println("Removing container...")
	rmReports, err := containersRemove(ctx, containerID, &containers.RemoveOptions{
		Force:   utils.GetPtr(true),
		Timeout: utils.GetPtr(uint(30)),
	})
	for _, report := range rmReports {
		if report.Err != nil {
			return report.Err
		}
	}
	return err
}

func GetContainerName(ctx context.Context, containerID string) (string, error) {
	ctr, err := containersInspect(ctx, containerID, nil)
	if err != nil {
		return "", nil
	}
	return ctr.Name, nil
}

func GetIPAddress(ctx context.Context, containerID string) (string, error) {
	inspectData, err := containersInspect(ctx, containerID, nil)
	if err != nil {
		return "", err
	}

	if inspectData.NetworkSettings == nil {
		return "", fmt.Errorf("No network settings found for container")
	}
	return inspectData.NetworkSettings.IPAddress, nil
}

// package podmanapi

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"time"

// 	"github.com/containers/podman/v5/libpod/define"
// 	"github.com/containers/podman/v5/pkg/bindings"
// 	"github.com/containers/podman/v5/pkg/bindings/containers"
// 	"github.com/containers/podman/v5/pkg/domain/entities/types"
// 	"github.com/containers/podman/v5/pkg/specgen"
// 	"github.com/sonarping/go-nodeapi/pkg/utils"
// )

// var timeout uint = 10
// var Podmanctx context.Context

// type Container struct {
// 	ID string
// }

// func InitPodmanConnection() (context.Context, error) {
// 	sock_dir := os.Getenv("XDG_RUNTIME_DIR")
// 	if sock_dir == "" {
// 		sock_dir = "/var/run"
// 	}
// 	socket := "unix:" + sock_dir + "/podman/podman.sock"

// 	if Podmanctx != nil {
// 		return Podmanctx, nil
// 	}

// 	Podmanctx, err := bindings.NewConnection(context.Background(), socket)
// 	if err != nil {
// 		fmt.Println(err)
// 		return nil, err
// 	}
// 	return Podmanctx, nil
// }

// type PodmanContainer struct {
// 	ID            string   `json:"env_id"`
// 	Image         string   `json:"image"`
// 	Names         []string `json:"names"`
// 	State         string   `json:"state"`
// 	StartedAt     int64    `json:"started_at"`
// 	Ports         []uint16 `json:"ports"`
// 	IP            string   `json:"ip"`
// 	Networks      []string `json:"networks"`
// 	Exited        bool     `json:"exited"`
// 	ExitCode      int32    `json:"exit_code"`
// 	ExitedAt      int64    `json:"exited_at"`
// 	Status        string   `json:"status"`
// 	CPUPercentage float64  `json:"cpu_percentage"`
// 	MemoryPercent float64  `json:"memory_percent"`
// 	Uptime        int64    `json:"uptime"`
// }
// type PodmanContainerStatus struct {
// 	ID    string `json:"id"`
// 	State string `json:"state"`
// }

// func ListPodmanContainers(ctx context.Context) ([]PodmanContainer, error) {
// 	fmt.Println("Listing containers...")
// 	ctrList, err := containers.List(ctx, &containers.ListOptions{All: utils.GetPtr(true)})
// 	if err != nil {
// 		return nil, err
// 	}
// 	var ctrStatusList []PodmanContainer
// 	for _, ctr := range ctrList {
// 		if err != nil {
// 			return nil, err
// 		}
// 		ip, err := GetIPAddress(ctx, ctr.ID)
// 		if err != nil {
// 			ip = ""
// 		}
// 		var stats types.ContainerStatsReport
// 		if ctr.State == define.ContainerStateRunning.String() {
// 			// get stats for running containers
// 			statsChan, err := containers.Stats(ctx, []string{ctr.ID}, &containers.StatsOptions{Stream: utils.GetPtr(false)})
// 			if err != nil {
// 				fmt.Println(err)
// 			}
// 			stats = <-statsChan
// 		}
// 		// get keys from ExposedPorts map as Ports list
// 		ctrStatusList = append(ctrStatusList, PodmanContainer{
// 			ID:            ctr.ID,
// 			Image:         ctr.Image,
// 			Names:         ctr.Names,
// 			State:         ctr.State,
// 			StartedAt:     ctr.StartedAt,
// 			Ports:         utils.GetMapKeys(ctr.ExposedPorts),
// 			Networks:      ctr.Networks,
// 			IP:            ip,
// 			Exited:        ctr.Exited,
// 			ExitCode:      ctr.ExitCode,
// 			ExitedAt:      ctr.ExitedAt,
// 			Status:        ctr.Status,
// 			CPUPercentage: stats.Stats[0].CPU,
// 			MemoryPercent: stats.Stats[0].MemPerc,
// 			Uptime:        int64(stats.Stats[0].UpTime),
// 		})
// 	}

// 	return ctrStatusList, nil
// }

// func StartPodmanContainer(ctx context.Context, containerID string) (PodmanContainerStatus, error) {
// 	fmt.Println("Starting container...")

// 	contData, testContainerErr := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
// 	if testContainerErr != nil {
// 		return PodmanContainerStatus{}, testContainerErr
// 	}

// 	if contData.State.Status == define.ContainerStateRunning.String() {
// 		return PodmanContainerStatus{}, fmt.Errorf("Container is already running")
// 	}

// 	err := containers.Start(ctx, containerID, &containers.StartOptions{})
// 	if err != nil {
// 		return PodmanContainerStatus{}, err
// 	}

// 	ret := make(chan bool)
// 	startContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
// 	go func() {
// 		_, err = containers.Wait(ctx, containerID, &containers.WaitOptions{
// 			Condition: []define.ContainerStatus{define.ContainerStateRunning},
// 		})
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		ret <- true
// 	}()

// 	select {
// 	case <-ret:
// 		break
// 	case <-startContext.Done():
// 		return PodmanContainerStatus{}, fmt.Errorf("Timeout waiting for container to start")
// 	}

// 	ctrData, err := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
// 	if err != nil {
// 		return PodmanContainerStatus{}, err
// 	}

// 	return PodmanContainerStatus{
// 		ID:    containerID,
// 		State: ctrData.State.Status,
// 	}, nil
// }

// func StopPodmanContainer(ctx context.Context, containerID string) (PodmanContainerStatus, error) {
// 	fmt.Println("Stopping container...")

// 	contData, testContainerErr := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
// 	if testContainerErr != nil {
// 		return PodmanContainerStatus{}, testContainerErr
// 	}

// 	if contData.State.Status == define.ContainerStateStopped.String() {
// 		return PodmanContainerStatus{}, fmt.Errorf("Container is already stopped")
// 	}

// 	err := containers.Stop(ctx, containerID, &containers.StopOptions{
// 		Ignore:  utils.GetPtr(false),
// 		Timeout: &timeout,
// 	})
// 	if err != nil {
// 		return PodmanContainerStatus{}, err
// 	}

// 	ret := make(chan bool)
// 	stopContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
// 	go func() {
// 		_, err = containers.Wait(ctx, containerID, &containers.WaitOptions{
// 			Condition: []define.ContainerStatus{define.ContainerStateStopped},
// 		})
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		ret <- true
// 	}()

// 	select {
// 	case <-ret:
// 		break
// 	case <-stopContext.Done():
// 		return PodmanContainerStatus{}, fmt.Errorf("Timeout waiting for container to stop")
// 	}

// 	if err != nil {
// 		return PodmanContainerStatus{}, err
// 	}

// 	ctrData, err := containers.Inspect(ctx, containerID, &containers.InspectOptions{})
// 	if err != nil {
// 		return PodmanContainerStatus{}, err
// 	}

// 	return PodmanContainerStatus{
// 		ID:    containerID,
// 		State: ctrData.State.Status,
// 	}, nil
// }

// func CreateFromImage(ctx context.Context, imageName string, containerName string) (string, error) {
// 	fmt.Println("Creating container...")
// 	spec := new(specgen.SpecGenerator)
// 	spec.Name = containerName
// 	spec.Image = imageName
// 	// basic container settings

// 	ctrData, err := containers.CreateWithSpec(ctx, spec, nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	return ctrData.ID, nil
// }

// func RemovePodmanContainer(ctx context.Context, containerID string) error {
// 	inspectData, err := containers.Inspect(ctx, containerID, nil)
// 	if err != nil {
// 		return err
// 	}
// 	if inspectData.State.Status == define.ContainerStateRunning.String() {
// 		fmt.Println("Stopping container first...")
// 		_, err := StopPodmanContainer(ctx, containerID)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	fmt.Println("Removing container...")
// 	rmReports, err := containers.Remove(ctx, containerID, &containers.RemoveOptions{
// 		Force:   utils.GetPtr(true),
// 		Timeout: utils.GetPtr(uint(30)),
// 	})
// 	for _, report := range rmReports {
// 		if report.Err != nil {
// 			return report.Err
// 		}
// 	}
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func GetIPAddress(ctx context.Context, containerID string) (string, error) {
// 	inspectData, err := containers.Inspect(ctx, containerID, nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	// get the IP address of the container
// 	// only top-level networks are returned

// 	if inspectData.NetworkSettings == nil {
// 		return "", fmt.Errorf("No network settings found for container")
// 	}
// 	return inspectData.NetworkSettings.IPAddress, nil
// }
