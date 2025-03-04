// podmanapi_test.go
package podmanapi

import (
	"context"
	"testing"
	"time"

	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings/containers"
)

// saveOriginals helps to restore original function variables after each test.
func saveOriginals() func() {
	origInspect := containersInspect
	origStart := containersStart
	origWait := containersWait

	return func() {
		containersInspect = origInspect
		containersStart = origStart
		containersWait = origWait
	}
}

func TestStartPodmanContainer_AlreadyRunning(t *testing.T) {
	restore := saveOriginals()
	defer restore()

	// Simulate container already running.
	containersInspect = func(ctx context.Context, containerID string, options *containers.InspectOptions) (*define.InspectContainerData, error) {
		thing := new(define.InspectContainerData)
		thing.ID = containerID
		newstate := new(define.InspectContainerState)
		newstate.Status = "running"
		newstate.Running = true
		thing.State = newstate
		return thing, nil
	}

	_, err := StartPodmanContainer(context.Background(), "testID")
	if err == nil || err.Error() != "Container is already running" {
		t.Fatalf("expected 'Container is already running' error, got: %v", err)
	}
}

func TestStartPodmanContainer_Success(t *testing.T) {
	restore := saveOriginals()
	defer restore()

	callCount := 0
	// First inspect returns not running; second inspect returns running.
	containersInspect = func(ctx context.Context, containerID string, options *containers.InspectOptions) (*define.InspectContainerData, error) {
		callCount++
		if callCount == 1 {
			thing := new(define.InspectContainerData)
			thing.ID = containerID
			newstate := new(define.InspectContainerState)
			newstate.Status = "exited"
			newstate.Running = false
			thing.State = newstate
			return thing, nil
		}
		// After start, inspect should show running.
		thing := new(define.InspectContainerData)
		thing.ID = containerID
		newstate := new(define.InspectContainerState)
		newstate.Status = "running"
		newstate.Running = true
		thing.State = newstate
		return thing, nil
	}

	// Simulate Start call succeeds.
	containersStart = func(ctx context.Context, containerID string, options *containers.StartOptions) error {
		return nil
	}

	// Simulate Wait call returns promptly.
	containersWait = func(ctx context.Context, containerID string, options *containers.WaitOptions) (int32, error) {
		ch := make(chan int32, 1)
		go func() {
			// Simulate a small delay.
			time.Sleep(100 * time.Millisecond)
			ch <- 100
		}()
		return <-ch, nil
	}

	status, err := StartPodmanContainer(context.Background(), "testID")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status.ID != "testID" || status.State != define.ContainerStateRunning.String() {
		t.Fatalf("expected running container status, got: %#v", status)
	}
}

func TestStartPodmanContainer_WaitTimeout(t *testing.T) {
	restore := saveOriginals()
	defer restore()

	// Inspect: first call returns not running.
	containersInspect = func(ctx context.Context, containerID string, options *containers.InspectOptions) (*define.InspectContainerData, error) {
		thing := new(define.InspectContainerData)
		thing.ID = containerID
		newstate := new(define.InspectContainerState)
		newstate.Running = false
		thing.State = newstate
		return thing, nil
	}
	containersStart = func(ctx context.Context, containerID string, options *containers.StartOptions) error {
		return nil
	}
	// Simulate Wait call never returning a report.
	containersWait = func(ctx context.Context, containerID string, options *containers.WaitOptions) (int32, error) {
		ch := make(chan int32)
		// Never send on channel.
		return <-ch, nil
	}

	_, err := StartPodmanContainer(context.Background(), "testID")
	if err == nil || err.Error() != "Timeout waiting for container to start" {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}
