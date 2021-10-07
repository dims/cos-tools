// Package commands implements subcommands of cos_gpu_installer.
package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"flag"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/internal/installer"
	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/internal/signing"
	"cos.googlesource.com/cos/tools.git/src/pkg/cos"
	"cos.googlesource.com/cos/tools.git/src/pkg/modules"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"

	log "github.com/golang/glog"
	"github.com/google/subcommands"
	"github.com/pkg/errors"
)

const (
	grepFound       = 0
	hostRootPath    = "/root"
	kernelSrcDir    = "/build/usr/src/linux"
	toolchainPkgDir = "/build/cos-tools"
)

// InstallCommand is the subcommand to install GPU drivers.
type InstallCommand struct {
	driverVersion      string
	hostInstallDir     string
	unsignedDriver     bool
	gcsDownloadBucket  string
	gcsDownloadPrefix  string
	nvidiaInstallerURL string
	debug              bool
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
		"The GPU driver verion to install. "+
			"It will install the default GPU driver if the flag is not set explicitly. "+
			"Set the flag to 'latest' to install the latest GPU driver version.")
	f.StringVar(&c.hostInstallDir, "host-dir", "",
		"Host directory that GPU drivers should be installed to. "+
			"It tries to read from the env NVIDIA_INSTALL_DIR_HOST if the flag is not set explicitly.")
	f.BoolVar(&c.unsignedDriver, "allow-unsigned-driver", false,
		"Whether to allow load unsigned GPU drivers. "+
			"If this flag is set to true, module signing security features must be disabled on the host for driver installation to succeed. "+
			"This flag is only for debugging.")
	f.StringVar(&c.gcsDownloadBucket, "gcs-download-bucket", "cos-tools",
		"The GCS bucket to download COS artifacts from. "+
			"For example, the default value is 'cos-tools' which is the public COS artifacts bucket.")
	f.StringVar(&c.gcsDownloadPrefix, "gcs-download-prefix", "",
		"The GCS path prefix when downloading COS artifacts."+
			"If not set then the COS version build number (e.g. 13310.1041.38) will be used.")
	f.StringVar(&c.nvidiaInstallerURL, "nvidia-installer-url", "",
		"A URL to an nvidia-installer to use for driver installation. This flag is mutually exclusive with `-version`. This flag must be used with `-allow-unsigned-driver`. This flag is only for debugging.")
	f.BoolVar(&c.debug, "debug", false,
		"Enable debug mode.")
}

func (c *InstallCommand) validateFlags() error {
	if c.nvidiaInstallerURL != "" && c.driverVersion != "" {
		return stderrors.New("-nvidia-installer-url and -version are both set; these flags are mutually exclusive")
	}
	if c.nvidiaInstallerURL != "" && c.unsignedDriver == false {
		return stderrors.New("-nvidia-installer-url is set, and -allow-unsigned-driver is not; -nvidia-installer-url must be used with -allow-unsigned-driver")
	}
	return nil
}

// Execute implements subcommands.Command.Execute.
func (c *InstallCommand) Execute(ctx context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := c.validateFlags(); err != nil {
		c.logError(err)
		return subcommands.ExitFailure
	}
	envReader, err := cos.NewEnvReader(hostRootPath)
	if err != nil {
		c.logError(errors.Wrapf(err, "failed to create envReader with host root path %s", hostRootPath))
		return subcommands.ExitFailure
	}

	if c.debug {
		if err := flag.Set("v", "2"); err != nil {
			log.Errorf("Unable to set debug logging: %v", err)
		}
	}

	log.V(2).Infof("Running on COS build id %s", envReader.BuildNumber())

	if releaseTrack := envReader.ReleaseTrack(); releaseTrack == "dev-channel" {
		c.logError(fmt.Errorf("GPU installation is not supported on dev images for now; Please use LTS image."))
		return subcommands.ExitFailure
	}

	var isGpuConfigured bool
	if isGpuConfigured, err = c.isGpuConfigured(); err != nil {
		c.logError(errors.Wrapf(err, "failed to check if GPU is configured"))
		return subcommands.ExitFailure
	}

	if !isGpuConfigured {
		c.logError(fmt.Errorf("Please have GPU device configured"))
		return subcommands.ExitFailure
	}

	downloader := cos.NewGCSDownloader(envReader, c.gcsDownloadBucket, c.gcsDownloadPrefix)
	if c.nvidiaInstallerURL == "" {
		c.driverVersion, err = getDriverVersion(downloader, c.driverVersion)
		if err != nil {
			c.logError(errors.Wrap(err, "failed to get default driver version"))
			return subcommands.ExitFailure
		}
		log.Infof("Installing GPU driver version %s", c.driverVersion)
	} else {
		log.Infof("Installing GPU driver from %q", c.nvidiaInstallerURL)
	}

	if c.unsignedDriver {
		kernelCmdline, err := ioutil.ReadFile("/proc/cmdline")
		if err != nil {
			c.logError(fmt.Errorf("failed to read kernel command line: %v", err))
		}
		if !cos.CheckKernelModuleSigning(string(kernelCmdline)) {
			log.Warning("Current kernel command line does not support unsigned kernel modules. Not enforcing kernel module signing may cause installation fail.")
		}
	}

	// Read value from env NVIDIA_INSTALL_DIR_HOST if the flag is not set. This is to be compatible with old interface.
	if c.hostInstallDir == "" {
		c.hostInstallDir = os.Getenv("NVIDIA_INSTALL_DIR_HOST")
	}
	hostInstallDir := filepath.Join(hostRootPath, c.hostInstallDir)
	var cacher *installer.Cacher
	// We only want to cache drivers installed from official sources.
	if c.nvidiaInstallerURL == "" {
		cacher = installer.NewCacher(hostInstallDir, envReader.BuildNumber(), c.driverVersion)
		if isCached, err := cacher.IsCached(); isCached && err == nil {
			log.V(2).Info("Found cached version, NOT building the drivers.")
			if err := installer.ConfigureCachedInstalltion(hostInstallDir, !c.unsignedDriver); err != nil {
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
	}

	log.V(2).Info("Did not find cached version, installing the drivers...")
	if err := installDriver(c, cacher, envReader, downloader); err != nil {
		c.logError(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func getDriverVersion(downloader *cos.GCSDownloader, argVersion string) (string, error) {
	if argVersion == "" {
		return installer.GetDefaultGPUDriverVersion(downloader)
	} else if argVersion == "latest" {
		return installer.GetLatestGPUDriverVersion(downloader)
	}
	// argVersion is an acutal verson, return it as-is.
	return argVersion, nil
}

func remountExecutable(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create dir %q: %v", dir, err)
	}
	if err := syscall.Mount(dir, dir, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to create bind mount at %q: %v", dir, err)
	}
	if err := syscall.Mount("", dir, "", syscall.MS_REMOUNT|syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_RELATIME, ""); err != nil {
		return fmt.Errorf("failed to remount %q: %v", dir, err)
	}
	return nil
}

func installDriver(c *InstallCommand, cacher *installer.Cacher, envReader *cos.EnvReader, downloader *cos.GCSDownloader) error {
	callback, err := installer.ConfigureDriverInstallationDirs(filepath.Join(hostRootPath, c.hostInstallDir), envReader.KernelRelease())
	if err != nil {
		return errors.Wrap(err, "failed to configure GPU driver installation dirs")
	}
	defer func() { callback <- 0 }()

	if !c.unsignedDriver {
		if err := signing.DownloadDriverSignatures(downloader, c.driverVersion); err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				return fmt.Errorf("The GPU driver is not available for the COS version. Please wait for half a day and retry.")
			}
			return errors.Wrap(err, "failed to download driver signature")
		}
	}

	var installerFile string
	if c.nvidiaInstallerURL == "" {
		installerFile, err = installer.DownloadDriverInstaller(
			c.driverVersion, envReader.Milestone(), envReader.BuildNumber())
		if err != nil {
			return errors.Wrap(err, "failed to download GPU driver installer")
		}
	} else {
		installerFile, err = installer.DownloadToInstallDir(c.nvidiaInstallerURL, "Unofficial GPU driver installer")
		if err != nil {
			return err
		}
	}

	if err := cos.SetCompilationEnv(downloader); err != nil {
		return errors.Wrap(err, "failed to set compilation environment variables")
	}
	if err := remountExecutable(toolchainPkgDir); err != nil {
		return fmt.Errorf("failed to remount %q as executable: %v", filepath.Dir(toolchainPkgDir), err)
	}
	if err := cos.InstallCrossToolchain(downloader, toolchainPkgDir); err != nil {
		return errors.Wrap(err, "failed to install toolchain")
	}

	if err := installer.RunDriverInstaller(toolchainPkgDir, installerFile, !c.unsignedDriver, false); err != nil {
		if errors.Is(err, installer.ErrDriverLoad) {
			// Drivers were linked, but couldn't load; try again with legacy linking
			log.Info("Retrying driver installation with legacy linking")
			if err := installer.RunDriverInstaller(toolchainPkgDir, installerFile, !c.unsignedDriver, true); err != nil {
				return fmt.Errorf("failed to run GPU driver installer: %v", err)
			}
		} else {
			return errors.Wrap(err, "failed to run GPU driver installer")
		}
	}
	if cacher != nil {
		if err := cacher.Cache(); err != nil {
			return errors.Wrap(err, "failed to cache installation")
		}
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

func (c *InstallCommand) isGpuConfigured() (bool, error) {
	cmd := "lspci | grep -i \"nvidia\""
	returnCode, err := utils.RunCommandWithExitCode([]string{"/bin/bash", "-c", cmd}, "", nil)
	if err != nil {
		return false, err
	}
	isConfigured := returnCode == grepFound
	return isConfigured, nil
}
