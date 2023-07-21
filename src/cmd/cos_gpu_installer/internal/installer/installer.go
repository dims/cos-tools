// Package installer provides functionality to install GPU drivers.
package installer

import (
	stderrors "errors"
	"fmt"
	"io/fs"
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
	gpuFirmwareDirContainer       = "/usr/local/nvidia/firmware/nvidia"
	templateGPUDriverFile         = "gpu_%s_version"
	precompiledInstallerURLFormat = "https://storage.googleapis.com/nvidia-drivers-%s-public/nvidia-cos-project/%s/tesla/%s_00/%s/NVIDIA-Linux-x86_64-%s_%s-%s.cos"
	defaultFilePermission         = 0755
	signedURLKey                  = "Expires"
	prebuiltModuleTemplate        = "nvidia-drivers-%s.tgz"
	DefaultVersion                = "default"
	LatestVersion                 = "latest"
	installerURLTemplate          = "https://storage.googleapis.com/nvidia-drivers-%[1]s-public/tesla/%[2]s/NVIDIA-Linux-x86_64-%[2]s.run"
)

var (
	gspFileNames = []string{"gsp.bin", "gsp_tu10x.bin", "gsp_ad10x.bin"}
	// ErrDriverLoad indicates that installed GPU drivers could not be loaded into
	// the kernel.
	ErrDriverLoad = stderrors.New("failed to load GPU drivers")

	errInstallerFailed = stderrors.New("failed to run GPU driver installer")
)

// VerifyDriverInstallation runs some commands to verify the driver installation.
func VerifyDriverInstallation(noVerify bool) error {
	if noVerify {
		log.Infof("Flag --no-verify is set, skip driver installation verification.")
		return nil
	}
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

	// Create symlinks in /dev/char for all possible NVIDIA device nodes
	if err := utils.RunCommandAndLogOutput(exec.Command("nvidia-ctk", "system", "create-dev-char-symlinks", "--create-all"), false); err != nil {
		return errors.Wrap(err, "failed to create symlinks")
	}
	return nil
}

// ConfigureCachedInstalltion updates ldconfig and installs the cached GPU driver kernel modules.
func ConfigureCachedInstalltion(gpuInstallDirHost string, needSigned, test, kernelOpen, noVerify bool, moduleParameters modules.ModuleParameters) error {
	log.V(2).Info("Configuring cached driver installation")

	if err := createHostDirBindMount(gpuInstallDirHost, gpuInstallDirContainer); err != nil {
		return errors.Wrap(err, "failed to create driver installation dir")
	}
	if err := updateContainerLdCache(); err != nil {
		return errors.Wrap(err, "failed to configure cached driver installation")
	}
	if err := loadGPUDrivers(moduleParameters, needSigned, test, kernelOpen, noVerify); err != nil {
		return errors.Wrap(err, "failed to configure cached driver installation")
	}

	return nil
}

// DownloadToInstallDir downloads data from the provided URL to the GPU
// installation directory. It returns the basename of the locally written file.
func DownloadToInstallDir(url, infoStr string) (string, error) {
	outputPath := filepath.Join(gpuInstallDirContainer, strings.Split(path.Base(url), "?"+signedURLKey+"=")[0])
	if err := utils.DownloadContentFromURL(url, outputPath, infoStr); err != nil {
		return "", fmt.Errorf("failed to download file with description %q from %q and install into %q: %v", infoStr, url, gpuInstallDirContainer, err)
	}
	return filepath.Base(outputPath), nil

}

// DownloadDriverInstaller downloads GPU driver installer given driver version and COS version.
func DownloadDriverInstaller(driverVersion, cosMilestone, cosBuildNumber string) (string, error) {
	log.Infof("Downloading GPU driver installer version %s", driverVersion)
	downloadURL, err := getDriverInstallerDownloadURL(driverVersion, cosMilestone, cosBuildNumber)
	if err != nil {
		return "", errors.Wrap(err, "failed to get driver installer download URL")
	}
	return DownloadToInstallDir(downloadURL, "GPU driver installer")
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
	linkerScriptExists, err := utils.CheckFileExists(linkerScript)
	if err != nil {
		return fmt.Errorf("failed to check if %s exists, err: %v", linkerScript, err)
	}
	if !linkerScriptExists {
		// Fallback to module-common.lds, which is used in the 5.4 kernel
		linkerScript = filepath.Join(toolchainDir, "usr", "src", "linux-headers-"+kernelRelease, "scripts", "module-common.lds")
	}
	nvidiaKernelDir := filepath.Join(nvidiaDir, "kernel")
	// Link nvidia.ko
	nvidiaObjs := []string{
		filepath.Join(nvidiaKernelDir, "precompiled", "nv-linux.o"),
		filepath.Join(nvidiaKernelDir, "nvidia", "nv-kernel.o_binary"),
	}
	args := append([]string{"-T", linkerScript, "-r", "-o", filepath.Join(nvidiaKernelDir, "precompiled", "nvidia.ko")}, nvidiaObjs...)
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
	args = append([]string{"-T", linkerScript, "-r", "-o", filepath.Join(nvidiaKernelDir, "precompiled", "nvidia-modeset.ko")}, modesetObjs...)
	cmd = exec.Command(linker, args...)
	log.Infof("Running link command: %v", cmd.Args)
	if err := utils.RunCommandAndLogOutput(cmd, false); err != nil {
		return fmt.Errorf("failed to link nvidia-modeset.ko: %v", err)
	}
	// Move all modules to kernel dir (includes some pre-linked modules, in
	// addition to the above linked ones)
	if err := filepath.WalkDir(filepath.Join(nvidiaKernelDir, "precompiled"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".ko" {
			newPath := filepath.Join(nvidiaKernelDir, filepath.Base(path))
			if err := unix.Rename(path, newPath); err != nil {
				return fmt.Errorf("failed to move %q to %q: %v", path, newPath, err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to copy kernel modules: %v", err)
	}
	log.Info("Done linking drivers")
	return nil
}

func linkDriversLegacy(toolchainDir, nvidiaDir string) error {
	log.Info("Linking drivers using legacy method...")
	// The legacy linking method needs to use "/usr/bin/ld" as the linker to
	// maintain bit-for-bit compatibility with driver signatures. The legacy
	// linking method also finds the linker by searching the PATH for "ld". If
	// bin/ld is present in the toolchain, rename it temporarily so the legacy
	// linking method doesn't use it.
	ld := filepath.Join(toolchainDir, "bin", "ld")
	if _, err := os.Lstat(ld); !os.IsNotExist(err) {
		dst := filepath.Join(toolchainDir, "bin", "ld.orig")
		if err := unix.Rename(ld, dst); err != nil {
			return fmt.Errorf("failed to rename %q to %q: %v", ld, dst, err)
		}
		defer func() {
			if err := unix.Rename(dst, ld); err != nil {
				// At this point, this error is non-fatal. It will become fatal when
				// something tries to use bin/ld in the toolchain. At time of writing,
				// nothing uses bin/ld after this point.
				log.Warningf("Could not restore %q", ld)
			}
		}()
	}
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
	err := utils.RunCommandAndLogOutput(cmd, false)
	log.Info("Done linking drivers")
	if err != nil {
		return fmt.Errorf("%w: %v", errInstallerFailed, err)
	}
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
func RunDriverInstaller(toolchainDir, installerFilename, driverVersion string, needSigned, test, legacyLink, noVerify bool, moduleParameters modules.ModuleParameters) error {
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
	var legacyInstallerFailed bool
	if legacyLink {
		if err := linkDriversLegacy(toolchainDir, extractDir); err != nil {
			if stderrors.Is(err, errInstallerFailed) {
				// This case is expected when module signature enforcement is enabled.
				// Since the installer terminated early, we need to re-run it after
				// signing modules.
				//
				// If we don't sign modules (i.e. needSigned is false), then we'll see
				// an error when we load the modules, and that will be fatal.
				legacyInstallerFailed = true
			} else {
				return fmt.Errorf("failed to link drivers: %v", err)
			}
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
		// in the unsigned case (we expect that legacy linking also does this when
		// the installer fails); we skip this block in the legacy link case to avoid
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
	// The legacy linking method does this when the installer doesn't fail (i.e.
	// module signature verification isn't enforced).
	if (legacyLink && legacyInstallerFailed) || !legacyLink {
		if err := loadGPUDrivers(moduleParameters, needSigned, test, false, noVerify); err != nil {
			return fmt.Errorf("%w: %v", ErrDriverLoad, err)
		}
	}

	// Install libs.
	// The legacy linking method does this when the installer doesn't fail (i.e.
	// module signature verification isn't enforced).
	if (legacyLink && legacyInstallerFailed) || !legacyLink {
		if err := installUserLibs(extractDir); err != nil {
			return fmt.Errorf("failed to install userspace libraries: %v", err)
		}

		// Driver version may be empty if custom nvidia-installer-url is used
		// read from manifest file
		if driverVersion == "" {

			driverVersion = findDriverVersionManifestFile(filepath.Join(extractDir, ".manifest"))
			log.Info("found driver version from nvidia-installer pkg ", driverVersion)
		}

		if err := prepareGSPFirmware(extractDir, driverVersion, needSigned); err != nil {
			return fmt.Errorf("failed to prepare GSP firmware, err: %v", err)
		}
	}

	return nil
}

// GeGGPUDriverVersion gets the supplied GPU driver version.
// Supports "default", "latest", "R470", "R525" aliases
func GetGPUDriverVersion(downloader cos.ArtifactsDownloader, alias string) (string, error) {
	log.Infof("Getting the %s GPU driver version", alias)
	content, err := downloader.GetArtifact(fmt.Sprintf(templateGPUDriverFile, alias))
	if err != nil {
		return "", errors.Wrapf(err, "failed to get %s GPU driver version", alias)
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

func loadGPUDrivers(moduleParams modules.ModuleParameters, needSigned, test, kernelOpen, noVerify bool) error {
	// Don't need to load public key in test mode. Platform key is used.
	if needSigned && !test && !kernelOpen {
		if err := modules.LoadPublicKey("gpu-key", filepath.Join(gpuInstallDirContainer, "pubkey.der"), modules.SecondaryKeyring); err != nil {
			return errors.Wrap(err, "failed to load public key")
		}
		// Load public key to IMA keyring for GSP firmware. For backward compatibility, it's OK if pubkey cannot be loaded.
		if err := modules.LoadPublicKey("gpu-key", filepath.Join(gpuInstallDirContainer, "pubkey.der"), modules.IMAKeyring); err != nil {
			log.Infof("Falied to load public key to IMA keyring, err: %v", err)
		}
	}
	if noVerify {
		log.Infof("Flag --no-verify is set, skip kernel module loading.")
		return nil
	}
	kernelModulePath := filepath.Join(gpuInstallDirContainer, "drivers")
	gpuModules := map[string]string{
		"nvidia":         filepath.Join(kernelModulePath, "nvidia.ko"),
		"nvidia_uvm":     filepath.Join(kernelModulePath, "nvidia-uvm.ko"),
		"nvidia_drm":     filepath.Join(kernelModulePath, "nvidia-drm.ko"),
		"nvidia_modeset": filepath.Join(kernelModulePath, "nvidia-modeset.ko"),
	}
	// Need to load modules in order due to module dependency.
	moduleNames := []string{"nvidia", "nvidia_uvm", "nvidia_drm", "nvidia_modeset"}
	for _, moduleName := range moduleNames {
		modulePath := gpuModules[moduleName]
		if err := modules.LoadModule(moduleName, modulePath, moduleParams); err != nil {
			return errors.Wrapf(err, "failed to load module %s", modulePath)
		}
	}
	return nil
}

func prepareGSPFirmware(extractDir, driverVersion string, needSigned bool) error {
	for _, gspFileName := range gspFileNames {
		signaturePath := signing.GetModuleSignature(gspFileName)
		installerGSPPath := filepath.Join(extractDir, "firmware", gspFileName)
		containerGSPPath := filepath.Join(gpuFirmwareDirContainer, driverVersion, gspFileName)
		haveSignature, err := utils.CheckFileExists(signaturePath)
		if err != nil {
			return fmt.Errorf("failed to check if %s exists, err: %v", signaturePath, err)
		}
		haveFirmware, err := utils.CheckFileExists(installerGSPPath)
		if err != nil {
			return fmt.Errorf("failed to check if %s exists, err: %v", installerGSPPath, err)
		}
		switch {
		case haveSignature && !haveFirmware:
			return fmt.Errorf("firmware doesn't exist but its signature does.")
		case !haveFirmware:
			log.Infof("GSP firmware for %s doesn't exist. Skipping firmware preparation for %s.", gspFileName, gspFileName)
		case !needSigned:
			// No signature needed, copy firmware only.
			if err := copyFirmware(installerGSPPath, containerGSPPath, driverVersion); err != nil {
				return fmt.Errorf("failed to copy firmware, err: %v.", err)
			}
		case !haveSignature:
			log.Infof("GSP firmware signature for %s doesn't exist. Skipping firmware preparation for %s.", gspFileName, gspFileName)
		default:
			// Both firmware and signature exist.
			if err := copyFirmware(installerGSPPath, containerGSPPath, driverVersion); err != nil {
				return fmt.Errorf("failed to copy firmware, err: %v.", err)
			}
			if err := setIMAXattr(signaturePath, containerGSPPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFirmware(installerGSPPath, containerGSPPath, gspFileName string) error {
	if err := os.MkdirAll(filepath.Dir(containerGSPPath), defaultFilePermission); err != nil {
		return fmt.Errorf("Falied to create firmware directory, err: %v", err)
	}
	if err := utils.CopyFile(installerGSPPath, containerGSPPath); err != nil {
		return fmt.Errorf("Falied to copy %s, err: %v", gspFileName, err)
	}
	return nil
}

func setIMAXattr(signaturePath, containerGSPPath string) error {
	signature, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("failed to read signature err: %v", err)
	}
	if err := syscall.Setxattr(containerGSPPath, "security.ima", signature, 0); err != nil {
		return fmt.Errorf("failed to set xattr for security.ima, err: %v", err)
	}
	return nil
}

// tries to read .manifest file to find driverVersion present in the manifest
func findDriverVersionManifestFile(manifestFilePath string) string {
	manifestFileRawBytes, err := os.ReadFile(manifestFilePath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(manifestFileRawBytes), "\n")
	if len(lines) < 2 {
		return ""
	}
	// driver version present in the second line of the file
	driverVersion := strings.TrimSpace(lines[1])
	return driverVersion
}

func RunDriverInstallerPrebuiltModules(downloader *cos.GCSDownloader, installerFilename, driverVersion string, noVerify bool, moduleParameters modules.ModuleParameters) error {
	// fetch the prebuilt modules
	if err := downloader.DownloadArtifact(gpuInstallDirContainer, fmt.Sprintf(prebuiltModuleTemplate, driverVersion)); err != nil {
		return fmt.Errorf("failed to download prebuilt modules: %v", err)
	}

	tarballPath := filepath.Join(gpuInstallDirContainer, fmt.Sprintf(prebuiltModuleTemplate, driverVersion))
	// extract the prebuilt modules and firmware to the installation dirs
	if err := exec.Command("tar", "--overwrite", "--xattrs", "--xattrs-include=*", "-xf", tarballPath, "-C", gpuInstallDirContainer).Run(); err != nil {
		return fmt.Errorf("failed to extract prebuilt modules: %v", err)
	}

	// load the prebuilt kernel modules
	if err := loadGPUDrivers(moduleParameters, false, false, true, noVerify); err != nil {
		return fmt.Errorf("%w: %v", ErrDriverLoad, err)
	}

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
	if err := installUserLibs(extractDir); err != nil {
		return fmt.Errorf("failed to install userspace libraries: %v", err)
	}

	return nil
}

func PrebuiltModulesAvailable(downloader *cos.GCSDownloader, driverVersion string, kernelOpen bool) (bool, error) {
	if !kernelOpen {
		return false, nil
	}

	prebuiltModulesArtifactPath := fmt.Sprintf(prebuiltModuleTemplate, driverVersion)
	return downloader.ArtifactExists(prebuiltModulesArtifactPath)
}

func getGenericDriverInstallerURL(driverVersion string) (string, error) {
	metadataZone, err := utils.GetGCEMetadata("zone")
	if err != nil {
		return "", errors.Wrap(err, "failed to get GCE metadata zone")
	}
	downloadLocation := getInstallerDownloadLocation(metadataZone)

	return fmt.Sprintf(installerURLTemplate, downloadLocation, driverVersion), nil
}

// DownloadGenericDriverInstaller downloads the generic GPU driver installer given driver version.
func DownloadGenericDriverInstaller(driverVersion string) (string, error) {
	log.Infof("Downloading GPU driver installer version %s", driverVersion)
	downloadURL, err := getGenericDriverInstallerURL(driverVersion)
	if err != nil {
		return "", errors.Wrap(err, "failed to get driver installer URL")
	}
	return DownloadToInstallDir(downloadURL, "GPU driver installer")
}
