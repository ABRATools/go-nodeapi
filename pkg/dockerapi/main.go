package dockerapi

import (
	"context"
	"encoding/json"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerContainer struct {
	ID     string   `json:"id"`
	Names  []string `json:"names"`
	Status string   `json:"status"`
	CPU    float64  `json:"cpu"`
	Memory uint64   `json:"memory"`
}

var dockerClient *client.Client

// InitDockerConnection initializes a connection to the Docker daemon, returning an error if it fails.
func InitDockerConnection() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	dockerClient = cli
	return nil
}

// StartDocker starts a Docker container by its ID, returning an error if it fails, it has a timeout of 100 seconds.
func StartDocker(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	if dockerClient == nil {
		err := InitDockerConnection()
		if err != nil {
			return err
		}
	}

	options := container.StartOptions{}
	err := dockerClient.ContainerStart(ctx, containerID, options)
	if err != nil {
		panic(err)
	}
	return err
}

// StopDocker stops a Docker container by its ID, returning an error if it fails, it has a timeout of 100 seconds.
func StopDocker(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	if dockerClient == nil {
		err := InitDockerConnection()
		if err != nil {
			return err
		}
	}

	var stopTimeout int = 100
	options := container.StopOptions{
		Timeout: &stopTimeout,
	}
	err := dockerClient.ContainerStop(ctx, containerID, options)
	if err != nil {
		panic(err)
	}
	return err
}

// ListContainers returns a slice of DockerContainer for all running containers.
func ListContainers() ([]DockerContainer, error) {
	ctx := context.Background()

	if dockerClient == nil {
		err := InitDockerConnection()
		if err != nil {
			return nil, err
		}
	}

	// List running containers (by default, ContainerList returns only running ones).
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, err
	}

	var infos []DockerContainer

	// Loop over each container to get its stats.
	for _, container := range containers {
		// Get container stats without streaming (one-shot)
		statsResp, err := dockerClient.ContainerStats(ctx, container.ID, false)
		if err != nil {
			return nil, err
		}
		// close the response body when finished.
		defer statsResp.Body.Close()

		// types.StatsJSON is deprecated, use container.StatsResponse instead.
		var stats types.StatsJSON
		// var stats container.StatsResponse
		if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
			return nil, err
		}

		// Calculate CPU usage percentage.
		cpuPercent := calculateCPUPercent(&stats)
		memUsage := stats.MemoryStats.Usage

		infos = append(infos, DockerContainer{
			ID:     container.ID,
			Names:  container.Names,
			Status: container.Status,
			CPU:    cpuPercent,
			Memory: memUsage,
		})
	}

	return infos, nil
}

// calculateCPUPercent calculates the CPU usage percentage based on the current and previous stats.
// This calculation is adapted from Docker's own CLI implementation.
func calculateCPUPercent(v *types.StatsJSON) float64 {
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)

	var cpuPercent = 0.0
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		// Multiply by the number of CPUs and scale to percentage.
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}
