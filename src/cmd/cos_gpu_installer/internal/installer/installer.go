// Package installer provides functionality to install GPU drivers.
package installer

import (
	stderrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/internal/signing"
	"cos.googlesource.com/cos/tools.git/src/pkg/cos"
	"cos.googlesource.com/cos/tools.git/src/pkg/modules"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"

	log "github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

const (
	gpuInstallDirContainer        = "/usr/local/nvidia"
	defaultGPUDriverFile          = "gpu_default_version"
	latestGPUDriverFile           = "gpu_latest_version"
	precompiledInstallerURLFormat = "https://storage.googleapis.com/nvidia-drivers-%s-public/nvidia-cos-project/%s/tesla/%s_00/%s/NVIDIA-Linux-x86_64-%s_%s-%s.cos"
	defaultFilePermission         = 0755
)

var (
	// ErrDriverLoad indicates that installed GPU drivers could not be loaded into
	// the kernel.
	ErrDriverLoad = stderrors.New("failed to load GPU drivers")
)

// VerifyDriverInstallation runs some commands to verify the driver installation.
func VerifyDriverInstallation() error {
	log.Info("Verifying GPU driver installation")

	newPathEnv := fmt.Sprintf("%s/bin:%s", gpuInstallDirContainer, os.Getenv("PATH"))
	os.Setenv("PATH", newPathEnv)
	// Run nvidia-smi to check whether nvidia GPU driver is installed.
	if err := utils.RunCommandAndLogOutput(exec.Command("nvidia-smi"), false); err != nil {
		return errors.Wrap(err, "failed to verify GPU driver installation")
	}

	// Create unified memory device file.
	if err := utils.RunCommandAndLogOutput(exec.Command("nvidia-modprobe", "-c0", "-u", "-m"), false); err != nil {
		return errors.Wrap(err, "failed to create unified memory device file")
	}

	return nil
}

// ConfigureCachedInstalltion updates ldconfig and installs the cached GPU driver kernel modules.
func ConfigureCachedInstalltion(gpuInstallDirHost string, needSigned bool) error {
	log.V(2).Info("Configuring cached driver installation")

	if err := createHostDirBindMount(gpuInstallDirHost, gpuInstallDirContainer); err != nil {
		return errors.Wrap(err, "failed to create driver installation dir")
	}
	if err := updateContainerLdCache(); err != nil {
		return errors.Wrap(err, "failed to configure cached driver installation")
	}
	if err := loadGPUDrivers(needSigned); err != nil {
		return errors.Wrap(err, "failed to configure cached driver installation")
	}

	return nil
}

// DownloadDriverInstaller downloads GPU driver installer given driver version and COS version.
func DownloadDriverInstaller(driverVersion, cosMilestone, cosBuildNumber string) (string, error) {
	log.Infof("Downloading GPU driver installer version %s", driverVersion)
	downloadURL, err := getDriverInstallerDownloadURL(driverVersion, cosMilestone, cosBuildNumber)
	if err != nil {
		return "", errors.Wrap(err, "failed to get driver installer download URL")
	}
	outputPath := filepath.Join(gpuInstallDirContainer, path.Base(downloadURL))
	if err := utils.DownloadContentFromURL(downloadURL, outputPath, "GPU driver installer"); err != nil {
		return "", errors.Wrapf(err, "failed to download GPU driver installer version %s", driverVersion)
	}
	return filepath.Base(outputPath), nil
}

// ConfigureDriverInstallationDirs configures GPU driver installation directories by creating mounts.
func ConfigureDriverInstallationDirs(gpuInstallDirHost string, kernelRelease string) (chan<- int, error) {
	log.Info("Configuring driver installation directories")

	if err := createHostDirBindMount(gpuInstallDirHost, gpuInstallDirContainer); err != nil {
		return nil, errors.Wrap(err, "failed to create dirver installation dir")
	}

	if err := createOverlayFS(
		"/usr/bin", gpuInstallDirContainer+"/bin", gpuInstallDirContainer+"/bin-workdir"); err != nil {
		return nil, errors.Wrap(err, "failed to create bin overlay")
	}
	if err := createOverlayFS(
		"/usr/lib/x86_64-linux-gnu", gpuInstallDirContainer+"/lib64", gpuInstallDirContainer+"/lib64-workdir"); err != nil {
		return nil, errors.Wrap(err, "failed to create lib64 overlay")
	}
	modulePath := filepath.Join("/lib/modules", kernelRelease, "video")
	if err := createOverlayFS(
		modulePath, gpuInstallDirContainer+"/drivers", gpuInstallDirContainer+"/drivers-workdir"); err != nil {
		return nil, errors.Wrap(err, "failed to create drivers overlay")
	}

	if err := updateContainerLdCache(); err != nil {
		return nil, errors.Wrap(err, "failed to update container ld cache")
	}

	ch := make(chan int, 1)
	go func() {
		// cleans up mounts created above.
		<-ch
		syscall.Unmount("/usr/bin", 0)
		syscall.Unmount("/usr/lib/x86_64-linux-gnu", 0)
		syscall.Unmount(modulePath, 0)
		syscall.Unmount(gpuInstallDirContainer, 0)
	}()
	return ch, nil
}

func extractPrecompiled(nvidiaDir string) error {
	log.Info("Extracting precompiled artifacts...")
	precompiledDir := filepath.Join(nvidiaDir, "kernel", "precompiled")
	files, err := os.ReadDir(precompiledDir)
	if err != nil {
		return fmt.Errorf("failed to read %q: %v", precompiledDir, err)
	}
	var precompiledArchive string
	if len(files) == 0 {
		return stderrors.New("failed to find precompiled artifacts in this nvidia installer")
	}
	if len(files) == 1 {
		precompiledArchive = filepath.Join(precompiledDir, files[0].Name())
	}
	if len(files) > 1 {
		var fileNames []string
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
		sort.Strings(fileNames)
		log.Warningf("Found multiple precompiled archives in this nvidia installer: %q", strings.Join(fileNames, ","))
		log.Warningf("Using precompiled archive named %q", fileNames[len(fileNames)-1])
		precompiledArchive = filepath.Join(precompiledDir, fileNames[len(fileNames)-1])
	}
	cmd := exec.Command(filepath.Join(nvidiaDir, "mkprecompiled"), "--unpack", precompiledArchive, "-o", precompiledDir)
	if err := utils.RunCommandAndLogOutput(cmd, false); err != nil {
		return fmt.Errorf("failed to unpack precompiled artifacts: %v", err)
	}
	log.Info("Done extracting precompiled artifacts")
	return nil
}

func linkDrivers(toolchainDir, nvidiaDir string) error {
	log.Info("Linking drivers...")
	var kernelInfo unix.Utsname
	if err := unix.Uname(&kernelInfo); err != nil {
		return fmt.Errorf("failed to find kernel release info using uname: %v", err)
	}
	kernelRelease := strings.Trim(string(kernelInfo.Release[:]), "\x00")
	// COS 85+ kernels use lld as their linker
	linker := filepath.Join(toolchainDir, "bin", "ld.lld")
	linkerScript := filepath.Join(toolchainDir, "usr", "src", "linux-headers-"+kernelRelease, "scripts", "module.lds")
	if _, err := os.Stat(linkerScript); os.IsNotExist(err) {
		// Fallback to module-common.lds, which is used in the 5.4 kernel
		linkerScript = filepath.Join(toolchainDir, "usr", "src", "linux-headers-"+kernelRelease, "scripts", "module-common.lds")
	}
	nvidiaKernelDir := filepath.Join(nvidiaDir, "kernel")
	// Link nvidia.ko
	nvidiaObjs := []string{
		filepath.Join(nvidiaKernelDir, "precompiled", "nv-linux.o"),
		filepath.Join(nvidiaKernelDir, "nvidia", "nv-kernel.o_binary"),
	}
	args := append([]string{"-T", linkerScript, "-r", "-o", filepath.Join(nvidiaKernelDir, "nvidia.ko")}, nvidiaObjs...)
	cmd := exec.Command(linker, args...)
	log.Infof("Running link command: %v", cmd.Args)
	if err := utils.RunCommandAndLogOutput(cmd, false); err != nil {
		return fmt.Errorf("failed to link nvidia.ko: %v", err)
	}
	// Link nvidia-modeset.ko
	modesetObjs := []string{
		filepath.Join(nvidiaKernelDir, "precompiled", "nv-modeset-linux.o"),
		filepath.Join(nvidiaKernelDir, "nvidia-modeset", "nv-modeset-kernel.o_binary"),
	}
	args = append([]string{"-T", linkerScript, "-r", "-o", filepath.Join(nvidiaKernelDir, "nvidia-modeset.ko")}, modesetObjs...)
	cmd = exec.Command(linker, args...)
	log.Infof("Running link command: %v", cmd.Args)
	if err := utils.RunCommandAndLogOutput(cmd, false); err != nil {
		return fmt.Errorf("failed to link nvidia-modeset.ko: %v", err)
	}
	// nvidia-uvm.ko is pre-linked; move to kernel dir
	oldPath := filepath.Join(nvidiaKernelDir, "precompiled", "nvidia-uvm.ko")
	newPath := filepath.Join(nvidiaKernelDir, "nvidia-uvm.ko")
	if err := unix.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to move %q to %q: %v", oldPath, newPath, err)
	}
	// nvidia-drm.ko is pre-linked; move to kernel dir
	oldPath = filepath.Join(nvidiaKernelDir, "precompiled", "nvidia-drm.ko")
	newPath = filepath.Join(nvidiaKernelDir, "nvidia-drm.ko")
	if err := unix.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to move %q to %q: %v", oldPath, newPath, err)
	}
	log.Info("Done linking drivers")
	return nil
}

func linkDriversLegacy(nvidiaDir string, needSigned bool) error {
	log.Info("Linking drivers using legacy method...")
	cmd := exec.Command(filepath.Join(nvidiaDir, "nvidia-installer"),
		"--utility-prefix="+gpuInstallDirContainer,
		"--opengl-prefix="+gpuInstallDirContainer,
		"--x-prefix="+gpuInstallDirContainer,
		"--install-libglvnd",
		"--no-install-compat32-libs",
		"--log-file-name="+filepath.Join(gpuInstallDirContainer, "nvidia-installer.log"),
		"--silent",
		"--accept-license",
	)
	log.Infof("Installer arguments:\n%v", cmd.Args)
	if err := utils.RunCommandAndLogOutput(cmd, needSigned); err != nil {
		return fmt.Errorf("failed to run GPU driver installer: %v", err)
	}
	log.Info("Done linking drivers")
	return nil
}

func installUserLibs(nvidiaDir string) error {
	log.Info("Installing userspace libraries...")
	cmd := exec.Command(filepath.Join(nvidiaDir, "nvidia-installer"),
		"--utility-prefix="+gpuInstallDirContainer,
		"--opengl-prefix="+gpuInstallDirContainer,
		"--x-prefix="+gpuInstallDirContainer,
		"--install-libglvnd",
		"--no-install-compat32-libs",
		"--log-file-name="+filepath.Join(gpuInstallDirContainer, "nvidia-installer.log"),
		"--silent",
		"--accept-license",
		"--no-kernel-module",
	)
	log.Infof("Installer arguments:\n%v", cmd.Args)
	if err := utils.RunCommandAndLogOutput(cmd, false); err != nil {
		return fmt.Errorf("failed to run GPU driver installer: %v", err)
	}
	log.Info("Done installing userspace libraries")
	return nil
}

// RunDriverInstaller runs GPU driver installer. Only works if the provided
// installer includes precompiled drivers.
func RunDriverInstaller(toolchainDir, installerFilename string, needSigned, legacyLink bool) error {
	log.Info("Running GPU driver installer")

	// Extract files to a fixed path first to make sure md5sum of generated gpu drivers are consistent.
	extractDir := "/tmp/extract"
	if err := os.RemoveAll(extractDir); err != nil {
		return fmt.Errorf("failed to clean %q: %v", extractDir, err)
	}
	cmd := exec.Command("sh", installerFilename, "-x", "--target", extractDir)
	cmd.Dir = gpuInstallDirContainer
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to extract installer files")
	}

	// Extract precompiled artifacts.
	if err := extractPrecompiled(extractDir); err != nil {
		return fmt.Errorf("failed to extract precompiled artifacts: %v", err)
	}

	// Link drivers.
	if legacyLink {
		if err := linkDriversLegacy(extractDir, needSigned); err != nil {
			return fmt.Errorf("failed to link drivers: %v", err)
		}
	} else {
		if err := linkDrivers(toolchainDir, extractDir); err != nil {
			return fmt.Errorf("failed to link drivers: %v", err)
		}
	}

	kernelFiles, err := ioutil.ReadDir(filepath.Join(extractDir, "kernel"))
	if err != nil {
		return errors.Wrapf(err, "failed to list files in directory %s", filepath.Join(extractDir, "kernel"))
	}
	if needSigned {
		// sign GPU drivers.
		for _, kernelFile := range kernelFiles {
			if strings.HasSuffix(kernelFile.Name(), ".ko") {
				module := kernelFile.Name()
				signaturePath := signing.GetModuleSignature(module)
				modulePath := filepath.Join(extractDir, "kernel", module)
				signedModulePath := filepath.Join(gpuInstallDirContainer, "drivers", module)
				if err := modules.AppendSignature(signedModulePath, modulePath, signaturePath); err != nil {
					return errors.Wrapf(err, "failed to sign kernel module %s", module)
				}
			}
		}
		// Copy public key.
		if err := utils.CopyFile(signing.GetPublicKeyDer(), filepath.Join(gpuInstallDirContainer, "pubkey.der")); err != nil {
			return errors.Wrapf(err, "failed to copy file %s", signing.GetPublicKeyDer())
		}
	} else if !legacyLink {
		// Copy drivers to the desired end directory. This is done as part of
		// `modules.AppendSignature` in the above signing block, but we need to do
		// it for unsigned modules as well. Legacy linking already does this copy
		// in the unsigned case; we skip this block in the legacy link case to avoid
		// redundancy.
		for _, kernelFile := range kernelFiles {
			if strings.HasSuffix(kernelFile.Name(), ".ko") {
				module := kernelFile.Name()
				src := filepath.Join(extractDir, "kernel", module)
				dst := filepath.Join(gpuInstallDirContainer, "drivers", module)
				if err := utils.CopyFile(src, dst); err != nil {
					return fmt.Errorf("failed to copy kernel module %q: %v", module, err)
				}
			}
		}
	}

	// Load GPU drivers.
	// The legacy linking method already does this in the unsigned case.
	if needSigned || !legacyLink {
		if err := loadGPUDrivers(needSigned); err != nil {
			return fmt.Errorf("%w: %v", ErrDriverLoad, err)
		}
	}

	// Install libs.
	// The legacy linking method already installs these libs in the unsigned
	// case. This step is redundant in that case.
	if needSigned || !legacyLink {
		if err := installUserLibs(extractDir); err != nil {
			return fmt.Errorf("failed to install userspace libraries: %v", err)
		}
	}

	return nil
}

// GetDefaultGPUDriverVersion gets the default GPU driver version.
func GetDefaultGPUDriverVersion(downloader cos.ArtifactsDownloader) (string, error) {
	log.Info("Getting the default GPU driver version")
	content, err := downloader.GetArtifact(defaultGPUDriverFile)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get default GPU driver version")
	}
	return strings.Trim(string(content), "\n "), nil
}

// GetLatestGPUDriverVersion gets the latest GPU driver version.
func GetLatestGPUDriverVersion(downloader cos.ArtifactsDownloader) (string, error) {
	log.Info("Getting the latest GPU driver version")
	content, err := downloader.GetArtifact(latestGPUDriverFile)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get latest GPU driver version")
	}
	return strings.Trim(string(content), "\n "), nil
}

func updateContainerLdCache() error {
	log.V(2).Info("Updating container's ld cache")

	f, err := os.Create("/etc/ld.so.conf.d/nvidia.conf")
	if err != nil {
		f.Close()
		return errors.Wrap(err, "failed to update ld cache")
	}
	f.WriteString(gpuInstallDirContainer + "/lib64")
	f.Close()

	err = exec.Command("ldconfig").Run()
	if err != nil {
		return errors.Wrap(err, "failed to update ld cache")
	}
	return nil
}

func getDriverInstallerDownloadURL(driverVersion, cosMilestone, cosBuildNumber string) (string, error) {
	metadataZone, err := utils.GetGCEMetadata("zone")
	if err != nil {
		return "", errors.Wrap(err, "failed to get GCE metadata zone")
	}
	downloadLocation := getInstallerDownloadLocation(metadataZone)

	return getPrecompiledInstallerURL(driverVersion, cosMilestone, cosBuildNumber, downloadLocation), nil
}

func getInstallerDownloadLocation(metadataZone string) string {
	fields := strings.Split(metadataZone, "/")
	zone := fields[len(fields)-1]
	locationMapping := map[string]string{
		"us":     "us",
		"asia":   "asia",
		"europe": "eu",
	}
	location, ok := locationMapping[strings.Split(zone, "-")[0]]
	if !ok {
		location = "us"
	}
	return location
}

func getPrecompiledInstallerURL(driverVersion, cosMilestone, cosBuildNumber, downloadLocation string) string {
	// 418.67 -> 418
	majorVersion := strings.Split(driverVersion, ".")[0]
	// 12371.284.0 -> 12371-284-0
	cosBuildNumber = strings.Replace(cosBuildNumber, ".", "-", -1)
	return fmt.Sprintf(
		precompiledInstallerURLFormat,
		downloadLocation, cosMilestone, majorVersion, driverVersion, driverVersion, cosMilestone, cosBuildNumber)
}

func createHostDirBindMount(hostDir, bindMountPath string) error {
	if err := os.MkdirAll(hostDir, defaultFilePermission); err != nil {
		return errors.Wrapf(err, "failed to create dir %s", hostDir)
	}
	if err := os.MkdirAll(bindMountPath, defaultFilePermission); err != nil {
		return errors.Wrapf(err, "failed to create dir %s", bindMountPath)
	}
	if err := syscall.Mount(hostDir, bindMountPath, "", syscall.MS_BIND, ""); err != nil {
		return errors.Wrapf(err, "failed to create bind mount %s", bindMountPath)
	}
	// Remount to clear noexec flag.
	if err := syscall.Mount("", bindMountPath, "",
		syscall.MS_REMOUNT|syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_RELATIME, ""); err != nil {
		return errors.Wrapf(err, "failed to remount %s", bindMountPath)
	}
	return nil
}

func createOverlayFS(lowerDir, upperDir, workDir string) error {
	if err := os.MkdirAll(lowerDir, defaultFilePermission); err != nil {
		return errors.Wrapf(err, "failed to create dir %s", lowerDir)
	}
	if err := os.MkdirAll(upperDir, defaultFilePermission); err != nil {
		return errors.Wrapf(err, "failed to create dir %s", upperDir)
	}
	if err := os.MkdirAll(workDir, defaultFilePermission); err != nil {
		return errors.Wrapf(err, "failed to create dir %s", workDir)
	}

	if err := syscall.Mount("none", lowerDir, "overlay", 0,
		fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)); err != nil {
		return errors.Wrapf(err, "failed to create overlayfs (lowerdir=%s, upperdir=%s)", lowerDir, upperDir)
	}
	return nil
}

func loadGPUDrivers(needSigned bool) error {
	if needSigned {
		if err := modules.LoadPublicKey("gpu-key", filepath.Join(gpuInstallDirContainer, "pubkey.der")); err != nil {
			return errors.Wrap(err, "failed to load public key")
		}
	}
	gpuModules := map[string]string{
		"nvidia":         filepath.Join(gpuInstallDirContainer, "drivers", "nvidia.ko"),
		"nvidia_uvm":     filepath.Join(gpuInstallDirContainer, "drivers", "nvidia-uvm.ko"),
		"nvidia_drm":     filepath.Join(gpuInstallDirContainer, "drivers", "nvidia-drm.ko"),
		"nvidia_modeset": filepath.Join(gpuInstallDirContainer, "drivers", "nvidia-modeset.ko"),
	}
	// Need to load modules in order due to module dependency.
	moduleNames := []string{"nvidia", "nvidia_uvm", "nvidia_drm", "nvidia_modeset"}
	for _, moduleName := range moduleNames {
		modulePath := gpuModules[moduleName]
		if err := modules.LoadModule(moduleName, modulePath); err != nil {
			return errors.Wrapf(err, "failed to load module %s", modulePath)
		}
	}
	return nil
}
