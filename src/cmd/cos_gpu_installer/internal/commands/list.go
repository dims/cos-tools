package commands

import (
	"context"
	"fmt"
	"strings"

	"flag"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/internal/installer"
	"cos.googlesource.com/cos/tools.git/src/pkg/cos"

	log "github.com/golang/glog"
	"github.com/google/subcommands"
	"github.com/pkg/errors"
)

// ListCommand is the subcommand to list supported GPU drivers.
type ListCommand struct {
	internalDownload bool
	debug            bool
}

// Name implements subcommands.Command.Name.
func (*ListCommand) Name() string { return "list" }

// Synopsis implements subcommands.Command.Synopsis.
func (*ListCommand) Synopsis() string { return "List supported GPU drivers for this version." }

// Usage implements subcommands.Command.Usage.
func (*ListCommand) Usage() string { return "list\n" }

// SetFlags implements subcommands.Command.SetFlags.
func (c *ListCommand) SetFlags(f *flag.FlagSet) {
	// TODO(mikewu): change this flag to a bucket prefix string.
	f.BoolVar(&c.internalDownload, "internal-download", false,
		"Whether to try to download files from Google internal server. This is only useful for internal developing.")
	f.BoolVar(&c.debug, "debug", false,
		"Enable debug mode.")
}

// Execute implements subcommands.Command.Execute.
func (c *ListCommand) Execute(ctx context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	envReader, err := cos.NewEnvReader(hostRootPath)
	if err != nil {
		c.logError(errors.Wrap(err, "failed to create envReader"))
		return subcommands.ExitFailure
	}
	log.Infof("Running on COS build id %s", envReader.BuildNumber())
	downloader := cos.NewGCSDownloader(envReader, c.internalDownload)
	artifacts, err := downloader.ListExtensionArtifacts("gpu")
	if err != nil {
		c.logError(errors.Wrap(err, "failed to list gpu extension artifacts"))
		return subcommands.ExitFailure
	}
	defaultVersion, err := installer.GetDefaultGPUDriverVersion(downloader)
	if err != nil {
		c.logError(errors.Wrap(err, "failed to get default driver version"))
		return subcommands.ExitFailure
	}
	for _, artifact := range artifacts {
		if strings.HasSuffix(artifact, ".signature.tar.gz") {
			driverVersion := strings.TrimSuffix(artifact, ".signature.tar.gz")
			if defaultVersion == driverVersion {
				fmt.Printf("%s [default]\n", driverVersion)
			} else {
				fmt.Printf("%s\n", driverVersion)
			}
		}
	}
	return subcommands.ExitSuccess
}

func (c *ListCommand) logError(err error) {
	if c.debug {
		log.Errorf("%+v", err)
	} else {
		log.Errorf("%v", err)
	}
}
