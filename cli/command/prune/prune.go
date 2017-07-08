package prune

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/command/network"
	"github.com/docker/cli/cli/command/volume"
	"github.com/docker/cli/opts"
)

// RunContainerPrune executes a prune command for containers
func RunContainerPrune(dockerCli command.Cli, dryRun bool, filter opts.FilterOpt) (uint64, string, error) {
	return container.RunPrune(dockerCli, dryRun, filter)
}

// RunVolumePrune executes a prune command for volumes
func RunVolumePrune(dockerCli command.Cli, dryRun bool, filter opts.FilterOpt) (uint64, string, error) {
	return volume.RunPrune(dockerCli, dryRun, filter)
}

// RunImagePrune executes a prune command for images
func RunImagePrune(dockerCli command.Cli, all bool, dryRun bool, filter opts.FilterOpt) (uint64, string, error) {
	return image.RunPrune(dockerCli, all, dryRun, filter)
}

// RunNetworkPrune executes a prune command for networks
func RunNetworkPrune(dockerCli command.Cli, dryRun bool, filter opts.FilterOpt) (uint64, string, error) {
	return network.RunPrune(dockerCli, dryRun, filter)
}
