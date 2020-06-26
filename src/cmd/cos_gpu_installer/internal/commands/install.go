// Package commands implements subcommands of cos_gpu_installer.
package commands

import (
	"context"
	"path/filepath"

	"cmd/cos_gpu_installer/internal/installer"
	"cmd/cos_gpu_installer/internal/signing"
	"flag"
	"pkg/cos"
	"pkg/modules"

	log "github.com/golang/glog"
	"github.com/google/subcommands"
	"github.com/pkg/errors"
)

const (
	hostRootPath    = "/root"
	kernelSrcDir    = "/build/usr/src/linux"
	kernelHeaderDir = "/build/usr/src/linux-headers"
	toolchainPkgDir = "/build/cos-tools"
)

// InstallCommand is the subcommand to install GPU drivers.
type InstallCommand struct {
	driverVersion    string
	hostInstallDir   string
	enforceSigning   bool
	internalDownload bool
	debug            bool
}

// Name implements subcommands.Command.Name.
func (*InstallCommand) Name() string { return "install" }

// Synopsis implements subcommands.Command.Synopsis.
func (*InstallCommand) Synopsis() string { return "Install GPU drivers." }

// Usage implements subcommands.Command.Usage.
func (*InstallCommand) Usage() string { return "install [-dir <filepath>]\n" }

// SetFlags implements subcommands.Command.SetFlags.
func (c *InstallCommand) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.driverVersion, "version", "",
		"The GPU driver verion to install. It will install the default GPU driver if the flag is not set explicitly.")
	f.StringVar(&c.hostInstallDir, "dir", "/var/lib/nvidia",
		"Host directory that GPU drivers should be installed to")
	f.BoolVar(&c.enforceSigning, "enforce-signing", true,
		"Whether to enforce GPU drivers being signed. Setting to false will disable kernel module signing security feature.")
	// TODO(mikewu): change this flag to a bucket prefix string.
	f.BoolVar(&c.internalDownload, "internal-download", false,
		"Whether to try to download files from Google internal server. This is only useful for internal developing.")
	f.BoolVar(&c.debug, "debug", false,
		"Enable debug mode.")
}

// Execute implements subcommands.Command.Execute.
func (c *InstallCommand) Execute(ctx context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	envReader, err := cos.NewEnvReader(hostRootPath)
	if err != nil {
		c.logError(errors.Wrapf(err, "failed to create envReader with host root path %s", hostRootPath))
		return subcommands.ExitFailure
	}

	log.Infof("Running on COS build id %s", envReader.BuildNumber())

	downloader := &cos.GCSDownloader{envReader, c.internalDownload}
	if c.driverVersion == "" {
		defaultVersion, err := installer.GetDefaultGPUDriverVersion(downloader)
		if err != nil {
			c.logError(errors.Wrap(err, "failed to get default driver version"))
			return subcommands.ExitFailure
		}
		c.driverVersion = defaultVersion
	}
	log.Infof("Installing GPU driver version %s", c.driverVersion)

	if !c.enforceSigning {
		log.Info("Doesn't enforce signing. Need to disable module locking.")
		if err := cos.DisableKernelModuleLocking(); err != nil {
			c.logError(errors.Wrap(err, "failed to configure kernel module locking"))
			return subcommands.ExitFailure
		}
	}

	hostInstallDir := filepath.Join(hostRootPath, c.hostInstallDir)
	cacher := installer.NewCacher(hostInstallDir, envReader.BuildNumber(), c.driverVersion)
	if isCached, err := cacher.IsCached(); isCached && err == nil {
		log.Info("Found cached version, NOT building the drivers.")
		if err := installer.ConfigureCachedInstalltion(hostInstallDir, c.enforceSigning); err != nil {
			c.logError(errors.Wrap(err, "failed to configure cached installation"))
			return subcommands.ExitFailure
		}
		if err := installer.VerifyDriverInstallation(); err != nil {
			c.logError(errors.Wrap(err, "failed to verify GPU driver installation"))
			return subcommands.ExitFailure
		}
		if err := modules.UpdateHostLdCache(hostRootPath, filepath.Join(c.hostInstallDir, "lib64")); err != nil {
			c.logError(errors.Wrap(err, "failed to update host ld cache"))
			return subcommands.ExitFailure
		}
		return subcommands.ExitSuccess
	}

	log.Info("Did not find cached version, installing the drivers...")
	if err := installDriver(c, cacher, envReader, downloader); err != nil {
		c.logError(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func installDriver(c *InstallCommand, cacher *installer.Cacher, envReader *cos.EnvReader, downloader *cos.GCSDownloader) error {
	callback, err := installer.ConfigureDriverInstallationDirs(filepath.Join(hostRootPath, c.hostInstallDir), envReader.KernelRelease())
	if err != nil {
		return errors.Wrap(err, "failed to configure GPU driver installation dirs")
	}
	defer func() { callback <- 0 }()

	if c.enforceSigning {
		if err := signing.DownloadDriverSignatures(downloader, c.driverVersion); err != nil {
			return errors.Wrap(err, "failed to download driver signature")
		}
	}

	installerFile, err := installer.DownloadDriverInstaller(
		c.driverVersion, envReader.Milestone(), envReader.BuildNumber())
	if err != nil {
		return errors.Wrap(err, "failed to download GPU driver installer")
	}

	if err := cos.SetCompilationEnv(downloader); err != nil {
		return errors.Wrap(err, "failed to set compilation environment variables")
	}
	if err := cos.InstallCrossToolchain(downloader, toolchainPkgDir); err != nil {
		return errors.Wrap(err, "failed to install toolchain")
	}

	if err := installer.RunDriverInstaller(installerFile, c.enforceSigning); err != nil {
		return errors.Wrap(err, "failed to run GPU driver installer")
	}
	if err := cacher.Cache(); err != nil {
		return errors.Wrap(err, "failed to cache installation")
	}
	if err := installer.VerifyDriverInstallation(); err != nil {
		return errors.Wrap(err, "failed to verify installation")
	}
	if err := modules.UpdateHostLdCache(hostRootPath, filepath.Join(c.hostInstallDir, "lib64")); err != nil {
		return errors.Wrap(err, "failed to update host ld cache")
	}
	log.Info("Finished installing the drivers.")
	return nil
}

func (c *InstallCommand) logError(err error) {
	if c.debug {
		log.Errorf("%+v", err)
	} else {
		log.Errorf("%v", err)
	}
}
