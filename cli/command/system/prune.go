package system

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/container"
	"github.com/docker/cli/cli/command/image"
	"github.com/docker/cli/cli/command/network"
	"github.com/docker/cli/cli/command/volume"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/versions"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type pruneOptions struct {
	force           bool
	all             bool
	pruneBuildCache bool
	pruneVolumes    bool
	filter          opts.FilterOpt
	dryRun bool
}

// newPruneCommand creates a new cobra.Command for `docker prune`
func newPruneCommand(dockerCli command.Cli) *cobra.Command {
	options := pruneOptions{filter: opts.NewFilterOpt(), pruneBuildCache: true}

	cmd := &cobra.Command{
		Use:   "prune [OPTIONS]",
		Short: "Remove unused data",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrune(dockerCli, options)
		},
		Tags: map[string]string{"version": "1.25"},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.force, "force", "f", false, "Do not prompt for confirmation")
	flags.BoolVarP(&options.all, "all", "a", false, "Remove all unused images not just dangling ones")
	flags.BoolVarP(&options.dryRun, "dry-run", "n", false, "Display prune report without removing anything")
	flags.BoolVar(&options.pruneVolumes, "volumes", false, "Prune volumes")
	flags.Var(&options.filter, "filter", "Provide filter values (e.g. 'label=<key>=<value>')")
	// "filter" flag is available in 1.28 (docker 17.04) and up
	flags.SetAnnotation("filter", "version", []string{"1.28"})

	return cmd
}

const confirmationTemplate = `WARNING! This will remove:
{{- range $_, $warning := . }}
        - {{ $warning }}
{{- end }}
Are you sure you want to continue?`

// runContainerPrune executes a prune command for containers
func runContainerPrune(dockerCli command.Cli, dryRun bool, filter opts.FilterOpt) (uint64, string, error) {
	return container.RunPrune(dockerCli, filter)
}

// runNetworkPrune executes a prune command for networks
func runNetworkPrune(dockerCli command.Cli, dryRun bool, filter opts.FilterOpt) (uint64, string, error) {
	return network.RunPrune(dockerCli, filter)
}

// runVolumePrune executes a prune command for volumes
func runVolumePrune(dockerCli command.Cli, dryRun bool, filter opts.FilterOpt) (uint64, string, error) {
	return volume.RunPrune(dockerCli, filter)
}

// runImagePrune executes a prune command for images
func runImagePrune(dockerCli command.Cli, all bool, dryRun bool, filter opts.FilterOpt) (uint64, string, error) {
	return image.RunPrune(dockerCli, all, filter)
}

// runBuildCachePrune executes a prune command for build cache
func runBuildCachePrune(dockerCli command.Cli, _ opts.FilterOpt) (uint64, string, error) {
	report, err := dockerCli.Client().BuildCachePrune(context.Background())
	if err != nil {
		return 0, "", err
	}
	return report.SpaceReclaimed, "", nil
}

func runPrune(dockerCli command.Cli, options pruneOptions) error {
	if versions.LessThan(dockerCli.Client().ClientVersion(), "1.31") {
		options.pruneBuildCache = false
	}
	if !options.force && !options.dryRun && !command.PromptForConfirmation(dockerCli.In(), dockerCli.Out(), confirmationMessage(options)) {
		return nil
	}
	imagePrune := func(dockerCli command.Cli, filter opts.FilterOpt) (uint64, string, error) {
		return runImagePrune(dockerCli, options.all, options.dryRun, options.filter)
	}
	pruneFuncs := []func(dockerCli command.Cli, dryRun bool, filter opts.FilterOpt) (uint64, string, error){
		runContainerPrune,
		runNetworkPrune,
	}
	if options.pruneVolumes {
		pruneFuncs = append(pruneFuncs, runVolumePrune)
	}
	pruneFuncs = append(pruneFuncs, imagePrune)
	if options.pruneBuildCache {
		pruneFuncs = append(pruneFuncs, runBuildCachePrune)
	}

	var spaceReclaimed uint64
	for _, pruneFn := range pruneFuncs {
		spc, output, err := pruneFn(dockerCli, options.dryRun, options.filter)
		if err != nil {
			return err
		}
		spaceReclaimed += spc
		if output != "" {
			fmt.Fprintln(dockerCli.Out(), output)
		}
	}

	spc, output, err := runImagePrune(dockerCli, options.all, options.dryRun, options.filter)
	if err != nil {
		return err
	}
	if spc > 0 {
		spaceReclaimed += spc
		fmt.Fprintln(dockerCli.Out(), output)
	}

	if options.pruneBuildCache {
		report, err := dockerCli.Client().BuildCachePrune(context.Background())
		if err != nil {
			return err
		}
		spaceReclaimed += report.SpaceReclaimed
	}

	spaceReclaimedLabel := "Total reclaimed space:"
	if options.dryRun {
		spaceReclaimedLabel = "Estimated reclaimable space:"
	}
	fmt.Fprintln(dockerCli.Out(), spaceReclaimedLabel, units.HumanSize(float64(spaceReclaimed)))

	return nil
}

// confirmationMessage constructs a confirmation message that depends on the cli options.
func confirmationMessage(options pruneOptions) string {
	t := template.Must(template.New("confirmation message").Parse(confirmationTemplate))

	warnings := []string{
		"all stopped containers",
		"all networks not used by at least one container",
	}
	if options.pruneVolumes {
		warnings = append(warnings, "all volumes not used by at least one container")
	}
	if options.all {
		warnings = append(warnings, "all images without at least one container associated to them")
	} else {
		warnings = append(warnings, "all dangling images")
	}
	if options.pruneBuildCache {
		warnings = append(warnings, "all build cache")
	}

	var buffer bytes.Buffer
	t.Execute(&buffer, &warnings)
	return buffer.String()
}
