package podmanapi

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/pkg/bindings/network"
)

func InitNewNetwork(ctx context.Context, name string, subnet net.IPNet, gateway net.IP) error {
	ipNet := types.IPNet{
		subnet,
	}
	netOptions := new(types.Network)
	netOptions.Name = name
	netOptions.ID = name
	netOptions.Subnets = []types.Subnet{
		{
			Subnet:  ipNet,
			Gateway: gateway,
		},
	}
	netOptions.Internal = false
	// The network.Create function returns a response containing details about the created network.
	netResponse, err := network.Create(ctx, netOptions)
	if err != nil {
		return fmt.Errorf("error creating network: %v", err)
	}

	log.Printf("Network created: %v", netResponse)
	return nil
}

func RemoveNetwork(ctx context.Context, name string) error {
	report, err := network.Remove(ctx, name, nil)
	if err != nil {
		return fmt.Errorf("error deleting network: %v", err)
	}
	log.Printf("Network deleted: %v", report)
	return nil
}

func ListNetworks(ctx context.Context) ([]types.Network, error) {
	networks, err := network.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error listing networks: %v", err)
	}
	return networks, nil
}

// returns IP address of the container after attaching it to the network
func AttachContainerToNetwork(ctx context.Context, containerID, networkName string) (string, error) {
	err := network.Connect(ctx, containerID, networkName, nil)
	if err != nil {
		return "", fmt.Errorf("error attaching container to network: %v", err)
	}
	ipAddr, err := GetIPAddress(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("error inspecting network: %v", err)
	}
	return ipAddr, nil
}
