// podmanapi.go
package podmanapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	nettypes "github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/api/handlers"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
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
type EBPFServiceStatus struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func ContainerExec(ctx context.Context, containerID string, command []string) (string, error) {
	fmt.Println("Executing command in container...")

	contData, err := containersInspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		return "", err
	}

	fmt.Println(contData.State.Status)

	if contData.State.Status != define.ContainerStateRunning.String() {
		return "", fmt.Errorf("Container is not running")
	}

	execConfig := new(handlers.ExecCreateConfig)
	execConfig.Cmd = command
	execConfig.AttachStdout = true
	execConfig.AttachStderr = true
	execConfig.Tty = false
	execID, err := containers.ExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("creating exec session: %w", err)
	}
	var stdout, stderr bytes.Buffer
	var w_out io.Writer = &stdout
	var w_err io.Writer = &stderr
	startOpts := new(containers.ExecStartAndAttachOptions)
	startOpts.AttachOutput = utils.GetPtr(true)
	startOpts.AttachError = utils.GetPtr(true)
	startOpts.AttachInput = utils.GetPtr(false)
	startOpts.OutputStream = utils.GetPtr(w_out)
	startOpts.ErrorStream = utils.GetPtr(w_err)
	if err := containers.ExecStartAndAttach(ctx, execID, startOpts); err != nil {
		return "", fmt.Errorf("starting exec session: %w", err)
	}

	inspectOptions := new(containers.ExecInspectOptions)
	inspect, err := containers.ExecInspect(ctx, execID, inspectOptions)
	if err != nil {
		return "", fmt.Errorf("inspecting exec session: %w", err)
	}
	if inspect.ExitCode != 0 {
		return "", fmt.Errorf("exec failed (exit %d): %s", inspect.ExitCode, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

func GetEBPFSystemdUnits(ctx context.Context, containerID string) ([]EBPFServiceStatus, error) {
	fmt.Println("Getting systemd units...")

	contData, err := containersInspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		return nil, err
	}

	fmt.Println(contData.State.Status)

	if contData.State.Status != define.ContainerStateRunning.String() {
		return nil, fmt.Errorf("Container is not running")
	}

	command := []string{"find", "/etc/systemd/system", "-maxdepth", "1", "-type", "f", "-name", "ebpf_*", "-printf", "%f\n"}
	output, err := ContainerExec(ctx, containerID, command)
	if err != nil {
		return nil, fmt.Errorf("executing command in container: %w", err)
	}

	lines := strings.Split(output, "\n")
	var files []string
	for _, l := range lines {
		if l != "" {
			files = append(files, l)
		}
	}
	EBPFServices := make([]EBPFServiceStatus, len(files))
	for i, f := range files {
		command = []string{"systemctl", "is-active", f}
		output, err = ContainerExec(ctx, containerID, command)
		if err != nil {
			return nil, fmt.Errorf("executing command in container: %w", err)
		}
		fmt.Println(output)
		status := output == "active"
		EBPFServices[i] = EBPFServiceStatus{
			Name:   f,
			Active: status,
		}
	}
	return EBPFServices, nil
}

func StartEBPFService(ctx context.Context, containerID string, ebpfService string) (bool, error) {
	fmt.Println("Getting systemd units...")

	contData, err := containersInspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		return false, err
	}

	fmt.Println(contData.State.Status)

	if contData.State.Status != define.ContainerStateRunning.String() {
		return false, fmt.Errorf("Container is not running")
	}

	command := []string{"systemctl", "start", ebpfService}
	_, err = ContainerExec(ctx, containerID, command)
	if err != nil {
		return false, fmt.Errorf("executing command in container: %w", err)
	}

	// Check if the service is running
	command = []string{"systemctl", "is-active", ebpfService}
	output, err := ContainerExec(ctx, containerID, command)
	if err != nil {
		return false, fmt.Errorf("executing command in container: %w", err)
	}
	if output == "active" {
		return true, nil
	} else {
		return false, fmt.Errorf("service is not running")
	}
}

func StopEBPFService(ctx context.Context, containerID string, ebpfService string) (bool, error) {
	fmt.Println("Stopping systemd units...")

	contData, err := containersInspect(ctx, containerID, &containers.InspectOptions{})
	if err != nil {
		return false, err
	}

	fmt.Println(contData.State.Status)

	if contData.State.Status != define.ContainerStateRunning.String() {
		return false, fmt.Errorf("Container is not running")
	}

	command := []string{"systemctl", "stop", ebpfService}
	_, err = ContainerExec(ctx, containerID, command)
	if err != nil {
		return false, fmt.Errorf("executing command in container: %w", err)
	}

	// Check if the service is stopped
	command = []string{"systemctl", "is-active", ebpfService}
	output, err := ContainerExec(ctx, containerID, command)
	if err != nil {
		return false, fmt.Errorf("executing command in container: %w", err)
	}
	if output == "inactive" {
		return true, nil
	} else {
		return false, fmt.Errorf("service is not stopped")
	}
}

func ListPodmanContainers(ctx context.Context) ([]PodmanContainer, error) {
	// fmt.Println("Listing containers...")
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

	//fmt.Println(contData)
	fmt.Println(contData.State.Status)

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

func CreateFromImage(ctx context.Context, imageName string, containerName string) (string, error) {
	fmt.Println("Creating container...")
	spec := new(specgen.SpecGenerator)
	spec.Name = containerName
	spec.Image = imageName

	// get node hostname
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	// Create log dirs /var/log/$hostname/$containerName
	baseDir := "/var/log/"
	jobLogDir := filepath.Join(baseDir, hostname, containerName)
	if err := os.MkdirAll(jobLogDir, 0755); err != nil {
		log.Fatalf("failed to create log dir: %v", err)
	}

	spec.Mounts = []specs.Mount{
		{
			Source:      jobLogDir,
			Destination: "/var/log/",
			Type:        "bind",
			Options:     []string{"rw"},
		},
	}

	ctrData, err := containersCreate(ctx, spec, nil)
	if err != nil {
		return "", err
	}

	return ctrData.ID, nil
}

func CreateFromImageWithStaticIP(ctx context.Context, imageName string, containerName string, static_ip net.IP) (string, error) {
	fmt.Println("Creating container with static IP...")
	spec := new(specgen.SpecGenerator)
	spec.Name = containerName
	spec.Image = imageName
	if static_ip != nil {
		spec.Networks = map[string]nettypes.PerNetworkOptions{
			"podman": {
				StaticIPs: []net.IP{static_ip},
			},
		}
	} else {
		return "", fmt.Errorf("Static IP cannot be nil")
	}

	// get node hostname
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	// Create log dirs /var/log/$hostname/$containerName
	baseDir := "/var/log/"
	jobLogDir := filepath.Join(baseDir, hostname, containerName)
	if err := os.MkdirAll(jobLogDir, 0755); err != nil {
		log.Fatalf("failed to create log dir: %v", err)
	}

	spec.Mounts = []specs.Mount{
		{
			Source:      jobLogDir,
			Destination: "/var/log/",
			Type:        "bind",
			Options:     []string{"rw"},
		},
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

func CreateEBPFContainer(ctx context.Context, imageName string, containerName string, static_ip net.IP) (string, error) {
	image := "base_ebpf:latest"
	if len(imageName) > 1 && imageName != "" {
		image = imageName
	}

	// Get kernel version
	kernelVer, err := exec.Command("uname", "-r").Output()
	if err != nil {
		log.Fatalf("failed to get kernel version: %v", err)
	}
	// Remove trailing newline character
	kernel := string(kernelVer[:len(kernelVer)-1])

	jobID := ""
	if len(containerName) > 1 && containerName != "" {
		jobID = containerName
	} else {
		// Generate 12 character random hex string for job ID
		bytes := make([]byte, 12)
		if _, err := rand.Read(bytes); err != nil {
			log.Fatalf("failed to generate job ID: %v", err)
		}
		jobID = hex.EncodeToString(bytes)
	}

	// get node hostname
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	// Create log dirs /var/log/$hostname/$jobID
	baseDir := "/var/log/"
	jobLogDir := filepath.Join(baseDir, hostname, jobID)
	if err := os.MkdirAll(jobLogDir, 0755); err != nil {
		log.Fatalf("failed to create log dir: %v", err)
	}

	spec := new(specgen.SpecGenerator)
	spec.Name = jobID
	spec.Image = image
	if static_ip != nil {
		spec.Networks = map[string]nettypes.PerNetworkOptions{
			"podman": {
				StaticIPs: []net.IP{static_ip},
			},
		}
	}
	spec.Privileged = utils.GetPtr(true)

	spec.Mounts = []specs.Mount{
		{
			Source:      fmt.Sprintf("/usr/src/kernels/%s", kernel),
			Destination: fmt.Sprintf("/usr/src/kernels/%s", kernel),
			Type:        "bind",
			Options:     []string{"ro"},
		},
		{
			Source:      fmt.Sprintf("/lib/modules/%s", kernel),
			Destination: fmt.Sprintf("/lib/modules/%s", kernel),
			Type:        "bind",
			Options:     []string{"ro"},
		},
		{
			Source:      jobLogDir,
			Destination: "/var/log/",
			Type:        "bind",
			Options:     []string{"rw"},
		},
		{
			Source:      "/sys/fs/cgroup",
			Destination: "/sys/fs/cgroup",
			Type:        "bind",
			Options:     []string{"ro"},
		},
		{
			Type:        "tmpfs",
			Source:      "tmpfs",
			Destination: "/run",
			Options:     []string{"rw", "nosuid", "nodev", "noexec"},
		},
		{
			Type:        "tmpfs",
			Source:      "tmpfs",
			Destination: "/run/lock",
			Options:     []string{"rw", "nosuid", "nodev", "noexec"},
		},
	}
	// spec.PortMappings = []nettypes.PortMapping{
	// 	{HostPort: 5801, ContainerPort: 5801},
	// 	{HostPort: 7681, ContainerPort: 7681},
	// }
	spec.CapAdd = []string{"CAP_BPF", "CAP_SYS_ADMIN"}
	spec.Terminal = utils.GetPtr(false)

	ctrData, err := containersCreate(ctx, spec, nil)
	if err != nil {
		return "", err
	}

	return ctrData.ID, nil
}
