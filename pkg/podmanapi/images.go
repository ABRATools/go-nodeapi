package podmanapi

import (
	"context"
	"fmt"

	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/sonarping/go-nodeapi/pkg/utils"
)

func GetImageList(ctx context.Context) ([]*types.ImageSummary, error) {
	images, err := images.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting images: %v", err)
	}
	return images, nil
}

// RemoveImage removes an image from the local storage
func RemoveImage(ctx context.Context, imageID string) (int, error) {
	imagesToRemove := []string{imageID}
	removeOpts := new(images.RemoveOptions)
	removeOpts.Force = utils.GetPtr(true)
	removeOpts.Ignore = utils.GetPtr(true)
	exitReport, err := images.Remove(ctx, imagesToRemove, removeOpts)
	if err != nil {
		return -1, fmt.Errorf("error removing image: %v", err)
	}
	return exitReport.ExitCode, nil
}

func BuildFromDockerFile(ctx context.Context, dockerFile string, imageLabel string) (string, error) {
	containerFiles := []string{dockerFile}
	buildOpts := new(types.BuildOptions)
	buildOpts.Labels = []string{imageLabel}
	buildOpts.Squash = true
	buildReport, err := images.Build(ctx, containerFiles, *buildOpts)
	if err != nil {
		return "", fmt.Errorf("error building image: %v", err)
	}
	return buildReport.ID, nil
}
