// Package commands implements subcommands of cos_gpu_installer.
package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"flag"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/internal/installer"
	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/internal/signing"
	"cos.googlesource.com/cos/tools.git/src/pkg/cos"
	"cos.googlesource.com/cos/tools.git/src/pkg/modules"

	log "github.com/golang/glog"
	"github.com/google/subcommands"
	"github.com/pkg/errors"
)

const (
	grepFound            = 0
	hostRootPath         = "/root"
	kernelSrcDir         = "/build/usr/src/linux"
	toolchainPkgDir      = "/build/cos-tools"
	installerURLTemplate = "https://us.download.nvidia.com/tesla/%[1]s/NVIDIA-Linux-x86_64-%[1]s.run"
)

type GPUType int

const (
	K80 GPUType = iota
	P4
	P100
	V100
	L4
	H100
	NO_GPU
	Others
)

func (g GPUType) String() string {
	switch g {
	case K80:
		return "K80"
	case P4:
		return "P4"
	case P100:
		return "P100"
	case V100:
		return "V100"
	case L4:
		return "L4"
	case H100:
		return "H100"
	case Others:
		return "Others"
	default:
		return "Unknown"
	}
}

func (g GPUType) OpenSupported() bool {
	switch g {
	case NO_GPU, K80, P4, P100, V100:
		return false
	default:
		return true
	}

}

type Fallback struct {
	minMajorVersion       int
	maxMajorVersion       int
	fallbackDriverVersion string
}

var fallbackMap = map[GPUType]Fallback{
	// R470 is the last driver family supporting K80 GPU devices.
	K80: {
		maxMajorVersion:       470,
		minMajorVersion:       450,
		fallbackDriverVersion: "R470",
	},
	L4: {
		minMajorVersion:       525,
		maxMajorVersion:       525,
		fallbackDriverVersion: "R525",
	},
	H100: {
		minMajorVersion:       525,
		maxMajorVersion:       525,
		fallbackDriverVersion: "R525",
	},
}

// InstallCommand is the subcommand to install GPU drivers.
type InstallCommand struct {
	driverVersion      string
	hostInstallDir     string
	unsignedDriver     bool
	gcsDownloadBucket  string
	gcsDownloadPrefix  string
	nvidiaInstallerURL string
	signatureURL       string
	debug              bool
	test               bool
	prepareBuildTools  bool
	kernelOpen         bool
	noVerify           bool
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
			"Set the flag to 'latest' to install the latest GPU driver version. "+
			"Please note that R470 is the last driver family supporting K80 GPU devices. "+
			"If a higher version is used with K80 GPU, the installer will automatically "+
			"choose an available R470 driver version.")
	f.StringVar(&c.hostInstallDir, "host-dir", "",
		"Host directory that GPU drivers should be installed to. "+
			"It tries to read from the env NVIDIA_INSTALL_DIR_HOST if the flag is not set explicitly.")
	f.BoolVar(&c.unsignedDriver, "allow-unsigned-driver", false,
		"Whether to allow load unsigned GPU drivers. "+
			"If this flag is set to true, module signing security features must be disabled on the host for driver installation to succeed. "+
			"This flag is only for debugging and testing.")
	f.StringVar(&c.gcsDownloadBucket, "gcs-download-bucket", "",
		"The GCS bucket to download COS artifacts from. "+
			"The default bucket is one of 'cos-tools', 'cos-tools-asia' and 'cos-tools-eu' based on where the VM is running. "+
			"Those are the public COS artifacts buckets.")
	f.StringVar(&c.gcsDownloadPrefix, "gcs-download-prefix", "",
		"The GCS path prefix when downloading COS artifacts."+
			"If not set then the COS version build number (e.g. 13310.1041.38) will be used.")
	f.StringVar(&c.nvidiaInstallerURL, "nvidia-installer-url", "",
		"A URL to an nvidia-installer to use for driver installation. This flag is mutually exclusive with `-version`. "+
			"This flag must be used with `-allow-unsigned-driver`. This flag is only for debugging and testing.")
	f.StringVar(&c.signatureURL, "signature-url", "",
		"A URL to the driver signature. This flag can only be used together with `-test` and `-nvidia-installer-url` for for debugging and testing.")
	f.BoolVar(&c.debug, "debug", false,
		"Enable debug mode.")
	f.BoolVar(&c.test, "test", false,
		"Enable test mode. "+
			"In test mode, `-nvidia-installer-url` can be used without `-allow-unsigned-driver`.")
	f.BoolVar(&c.prepareBuildTools, "prepare-build-tools", false, "Whether to populate the build tools cache, i.e. to download and install the toolchain and the kernel headers. Drivers are NOT installed when this flag is set and running with this flag does not require GPU attached to the instance.")
	f.BoolVar(&c.noVerify, "no-verify", false, "Skip kernel module loading and installation verification. Useful for preloading drivers without attached GPU.")

}

func (c *InstallCommand) validateFlags() error {
	if c.nvidiaInstallerURL != "" && c.driverVersion != "" {
		return stderrors.New("-nvidia-installer-url and -version are both set; these flags are mutually exclusive")
	}
	if c.nvidiaInstallerURL != "" && c.unsignedDriver == false && c.test == false {
		return stderrors.New("-nvidia-installer-url is set, and -allow-unsigned-driver is not; -nvidia-installer-url must be used with -allow-unsigned-driver if not in test mode")
	}
	if c.signatureURL != "" && (c.nvidiaInstallerURL == "" || c.test == false) {
		return stderrors.New("-signature-url must be used with -nvidia-installer-url and -test")
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

	// All prerelease builds are in dev-channel. For testing we don't need to check release track.
	// we can preload dependencies for dev-channel images too.
	if releaseTrack := envReader.ReleaseTrack(); !c.prepareBuildTools && !c.test && releaseTrack == "dev-channel" {
		c.logError(fmt.Errorf("GPU installation is not supported on dev images for now; Please use LTS image."))
		return subcommands.ExitFailure
	}

	var gpuType GPUType = NO_GPU

	if !c.prepareBuildTools {
		if gpuType, err = c.getGPUTypeInfo(); err != nil {
			if !c.noVerify {
				c.logError(errors.Wrapf(err, "failed to get GPU type information"))
				return subcommands.ExitFailure
			}
			log.Infof("No GPU device configured, continue driver preoloading without verification.")
		}
	}

	downloader := cos.NewGCSDownloader(envReader, c.gcsDownloadBucket, c.gcsDownloadPrefix)
	if c.nvidiaInstallerURL == "" {
		versionInput := c.driverVersion
		milestone, err := strconv.Atoi(envReader.Milestone())
		if err != nil {
			c.logError(errors.Wrap(err, "failed to parse milestone number"))
			return subcommands.ExitFailure
		}
		c.driverVersion, err = getDriverVersion(downloader, c.driverVersion)
		if err != nil {
			if versionInput == "latest" && milestone < 93 {
				c.logError(errors.Wrap(err, "'--version=latest' is only supported on COS M93 and onwards, please unset this flag"))
				return subcommands.ExitFailure
			} else {
				c.logError(errors.Wrap(err, "failed to get default driver version"))
				return subcommands.ExitFailure
			}
		}
		if err := c.checkDriverCompatibility(downloader, gpuType); err != nil {
			c.logError(errors.Wrap(err, "failed to check driver compatibility"))
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
		if cos.CheckKernelModuleSigning(string(kernelCmdline)) {
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
		if isCached, isOpen, err := cacher.IsCached(); isCached && err == nil {
			log.V(2).Info("Found cached version, NOT building the drivers.")
			if err := installer.ConfigureCachedInstalltion(hostInstallDir, !c.unsignedDriver, c.test, isOpen, c.noVerify); err != nil {
				c.logError(errors.Wrap(err, "failed to configure cached installation"))
				return subcommands.ExitFailure
			}
			if err := installer.VerifyDriverInstallation(c.noVerify); err != nil {
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

	// install OSS kernel modules (if available) if device supports
	if !c.unsignedDriver && gpuType.OpenSupported() {
		c.kernelOpen = gpuType.OpenSupported()
	}

	prebuiltModulesAvailable, err := installer.PrebuiltModulesAvailable(downloader, c.driverVersion, c.kernelOpen)

	if err != nil {
		c.logError(errors.Wrap(err, "failed to find prebuilt modules"))
		return subcommands.ExitFailure
	}

	// skip prebuilt module installation if preparing build tools
	if !c.prepareBuildTools && prebuiltModulesAvailable {
		log.V(2).Info("Found prebuilt kernel modules, installing additional components...")
		if err := installDriverPrebuiltModules(c, cacher, envReader, downloader); err != nil {
			c.logError(err)
			return subcommands.ExitFailure
		}
		return subcommands.ExitSuccess
	}

	if err := installDriver(c, cacher, envReader, downloader); err != nil {
		c.logError(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func getDriverVersion(downloader *cos.GCSDownloader, argVersion string) (string, error) {
	if argVersion == "" {
		return installer.GetGPUDriverVersion(downloader, installer.DefaultVersion)
	} else if argVersion == "latest" {
		return installer.GetGPUDriverVersion(downloader, installer.LatestVersion)
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

	if err := cos.SetCompilationEnv(downloader); err != nil {
		return errors.Wrap(err, "failed to set compilation environment variables")
	}
	if err := remountExecutable(toolchainPkgDir); err != nil {
		return fmt.Errorf("failed to remount %q as executable: %v", filepath.Dir(toolchainPkgDir), err)
	}
	if err := cos.InstallCrossToolchain(downloader, toolchainPkgDir); err != nil {
		return errors.Wrap(err, "failed to install toolchain")
	}

	// Skip driver installation if we are only populating build tools cache
	if c.prepareBuildTools {
		return nil
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

	if !c.unsignedDriver {
		if c.signatureURL != "" {
			if err := signing.DownloadDriverSignaturesFromURL(c.signatureURL); err != nil {
				return errors.Wrap(err, "failed to download driver signature")
			}
		} else if err := signing.DownloadDriverSignatures(downloader, c.driverVersion); err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				return fmt.Errorf("The GPU driver is not available for the COS version. Please wait for half a day and retry.")
			}
			return errors.Wrap(err, "failed to download driver signature")
		}
	}

	if err := installer.RunDriverInstaller(toolchainPkgDir, installerFile, c.driverVersion, !c.unsignedDriver, c.test, false, c.noVerify); err != nil {
		if errors.Is(err, installer.ErrDriverLoad) {
			// Drivers were linked, but couldn't load; try again with legacy linking
			log.Infof("Failed to load kernel module, err: %v. Retrying driver installation with legacy linking", err)
			if err := installer.RunDriverInstaller(toolchainPkgDir, installerFile, c.driverVersion, !c.unsignedDriver, c.test, true, c.noVerify); err != nil {
				return fmt.Errorf("failed to run GPU driver installer: %v", err)
			}
		} else {
			return errors.Wrap(err, "failed to run GPU driver installer")
		}
	}
	if cacher != nil {
		if err := cacher.Cache(false); err != nil {
			return errors.Wrap(err, "failed to cache installation")
		}
	}
	if err := installer.VerifyDriverInstallation(c.noVerify); err != nil {
		return errors.Wrap(err, "failed to verify installation")
	}
	if err := modules.UpdateHostLdCache(hostRootPath, filepath.Join(c.hostInstallDir, "lib64")); err != nil {
		return errors.Wrap(err, "failed to update host ld cache")
	}
	log.Info("Finished installing the drivers.")
	return nil
}

func installDriverPrebuiltModules(c *InstallCommand, cacher *installer.Cacher, envReader *cos.EnvReader, downloader *cos.GCSDownloader) error {
	callback, err := installer.ConfigureDriverInstallationDirs(filepath.Join(hostRootPath, c.hostInstallDir), envReader.KernelRelease())
	if err != nil {
		return errors.Wrap(err, "failed to configure GPU driver installation dirs")
	}
	defer func() { callback <- 0 }()

	installerURL := fmt.Sprintf(installerURLTemplate, c.driverVersion)
	installerFile, err := installer.DownloadToInstallDir(installerURL, "Downloading driver installer")
	if err != nil {
		return err
	}

	if err := installer.RunDriverInstallerPrebuiltModules(downloader, installerFile, c.driverVersion, c.noVerify); err != nil {
		return err
	}

	if cacher != nil {
		if err := cacher.Cache(true); err != nil {
			return errors.Wrap(err, "failed to cache installation")
		}
	}
	if err := installer.VerifyDriverInstallation(c.noVerify); err != nil {
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

func (c *InstallCommand) getGPUTypeInfo() (GPUType, error) {
	cmd := "lspci | grep -i \"nvidia\""
	outBytes, err := exec.Command("/bin/bash", "-c", cmd).Output()
	if err != nil {
		return NO_GPU, err
	}
	out := string(outBytes)
	switch {
	case strings.Contains(out, "[Tesla K80]"):
		return K80, nil
	case strings.Contains(out, "NVIDIA Corporation Device 15f8"), strings.Contains(out, "NVIDIA Corporation GP100GL"), strings.Contains(out, "[Tesla P100"):
		return P100, nil
	case strings.Contains(out, "NVIDIA Corporation Device 1db1"), strings.Contains(out, "NVIDIA Corporation GV100GL"), strings.Contains(out, "[Tesla V100"):
		return V100, nil
	case strings.Contains(out, "NVIDIA Corporation Device 1bb3"), strings.Contains(out, "NVIDIA Corporation GP104GL"), strings.Contains(out, "[Tesla P4"):
		return P4, nil
	case strings.Contains(out, "NVIDIA Corporation Device 27b8"), strings.Contains(out, "NVIDIA Corporation AD104GL [L4]"):
		return L4, nil
	case strings.Contains(out, "NVIDIA Corporation Device 2330"), strings.Contains(out, "NVIDIA Corporation GH100[H100"):
		return H100, nil
	default:
		return Others, nil
	}
}

func (c *InstallCommand) checkDriverCompatibility(downloader *cos.GCSDownloader, gpuType GPUType) error {
	driverMajorVersion, err := strconv.Atoi(strings.Split(c.driverVersion, ".")[0])
	if err != nil {
		return errors.Wrap(err, "failed to get driver major version")
	}
	fallback, found := fallbackMap[gpuType]
	if found && (driverMajorVersion > fallback.maxMajorVersion || driverMajorVersion < fallback.minMajorVersion) {
		log.Warningf("\n\nDriver version %s doesn't support %s GPU devices.\n\n", c.driverVersion, gpuType)
		fallbackVersion, err := installer.GetGPUDriverVersion(downloader, fallback.fallbackDriverVersion)
		if err != nil {
			return errors.Wrap(err, "failed to get fallback driver")
		}
		log.Warningf("\n\nUsing driver version %s for %s GPU compatibility.\n\n", fallbackVersion, gpuType)
		c.driverVersion = fallbackVersion
		return nil
	}
	return nil
}
