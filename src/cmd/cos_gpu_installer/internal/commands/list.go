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
	gcsDownloadBucket string
	gcsDownloadPrefix string
	debug             bool
}

// Name implements subcommands.Command.Name.
func (*ListCommand) Name() string { return "list" }

// Synopsis implements subcommands.Command.Synopsis.
func (*ListCommand) Synopsis() string { return "List supported GPU drivers for this version." }

// Usage implements subcommands.Command.Usage.
func (*ListCommand) Usage() string { return "list\n" }

// SetFlags implements subcommands.Command.SetFlags.
func (c *ListCommand) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.gcsDownloadBucket, "gcs-download-bucket", "cos-tools",
		"The GCS bucket to download COS artifacts from. "+
			"For example, the default value is 'cos-tools' which is the public COS artifacts bucket.")
	f.StringVar(&c.gcsDownloadPrefix, "gcs-download-prefix", "",
		"The GCS path prefix when downloading COS artifacts."+
			"If not set then the COS build number and board (e.g. 13310.1041.38/lakitu) will be used.")
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
	downloader := cos.NewGCSDownloader(envReader, c.gcsDownloadBucket, c.gcsDownloadPrefix)
	artifacts, err := downloader.ListGPUExtensionArtifacts()
	if err != nil {
		c.logError(errors.Wrap(err, "failed to list gpu extension artifacts"))
		return subcommands.ExitFailure
	}
	defaultVersion, err := installer.GetGPUDriverVersion(downloader, installer.DefaultVersion)
	if err != nil {
		c.logError(errors.Wrap(err, "failed to get default driver version"))
		return subcommands.ExitFailure
	}
	latestVersion, err := installer.GetGPUDriverVersion(downloader, installer.LatestVersion)
	if err != nil {
		c.logWarning(errors.Wrap(err, "failed to get latest driver version"))
	}
	for _, artifact := range artifacts {
		driverVersion := ""
		if strings.HasSuffix(artifact, ".signature.tar.gz") {
			driverVersion = strings.TrimSuffix(artifact, ".signature.tar.gz")
		} else if strings.HasPrefix(artifact, "nvidia-drivers-") && strings.HasSuffix(artifact, "-signature.tar.gz") {
			driverVersion = strings.TrimPrefix(artifact, "nvidia-drivers-")
			driverVersion = strings.TrimSuffix(driverVersion, "-signature.tar.gz")
		}
		if driverVersion != "" {
			if defaultVersion == driverVersion {
				if latestVersion == driverVersion {
					fmt.Printf("%s [default][latest]\n", driverVersion)
				} else {
					fmt.Printf("%s [default]\n", driverVersion)
				}
			} else if latestVersion == driverVersion {
				fmt.Printf("%s [latest]\n", driverVersion)
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

func (c *ListCommand) logWarning(err error) {
	if c.debug {
		log.Warningf("%+v", err)
	} else {
		log.Warningf("%v", err)
	}
}
