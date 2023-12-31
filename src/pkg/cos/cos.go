// Package cos provides functionality to read and configure system configs that are specific to COS images.
package cos

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cos.googlesource.com/cos/tools.git/src/pkg/utils"

	log "github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	espPartition      = "/dev/sda12"
	utsFilepath       = "include/generated/utsrelease.h"
	toolchainBinPath  = "bin"
	kernelHeadersPath = "usr/src/linux-headers*"
)

var (
	execCommand = exec.Command
)

// CheckKernelModuleSigning checks whether kernel module signing related options present.
func CheckKernelModuleSigning(kernelCmdline string) bool {
	log.Info("Checking kernel module signing.")

	for _, kernelOption := range []string{
		"loadpin.exclude=kernel-module",
		"modules-load=loadpin_trigger",
		"module.sig_enforce=1",
	} {
		if !strings.Contains(kernelCmdline, kernelOption) {
			return false
		}
	}
	return true
}

// SetCompilationEnv sets compilation environment variables (e.g. CC, CXX) for third-party kernel module compilation.
// TODO(mikewu): pass environment variables to the *exec.Cmd that runs the installer.
func SetCompilationEnv(downloader ArtifactsDownloader) error {
	log.V(2).Info("Downloading compilation environment variables")

	compilationEnvs := make(map[string]string)

	if err := downloader.DownloadToolchainEnv(os.TempDir()); err != nil {
		// Required to support COS builds not having toolchain_env file
		log.V(2).Info("Using default compilation environment variables")
		compilationEnvs["CC"] = "x86_64-cros-linux-gnu-gcc"
		compilationEnvs["CXX"] = "x86_64-cros-linux-gnu-g++"
	} else {
		if compilationEnvs, err = utils.LoadEnvFromFile(os.TempDir(), toolchainEnv); err != nil {
			return errors.Wrap(err, "failed to parse toolchain_env file")
		}
	}

	log.V(2).Info("Setting compilation environment variables")
	for key, value := range compilationEnvs {
		log.V(2).Infof("%s=%s", key, value)
		os.Setenv(key, value)
	}
	return nil
}

// InstallCrossToolchain installs COS toolchain and kernel headers to destination directory.
func InstallCrossToolchain(downloader ArtifactsDownloader, destDir string) error {
	log.Info("Installing the toolchain")

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create dir %s", destDir)
	}
	foundToolchain, foundKernelHeaders := false, false
	if empty, _ := utils.IsDirEmpty(destDir); !empty {
		if _, err := os.Stat(filepath.Join(destDir, toolchainBinPath)); err == nil {
			foundToolchain = true
		}
		if dirs, err := filepath.Glob(filepath.Join(destDir, kernelHeadersPath)); err == nil && len(dirs) > 0 {
			foundKernelHeaders = true
		}
	}

	if foundToolchain {
		log.Info("Found existing toolchain. Skipping download and installation.")
	} else {
		if err := downloader.DownloadToolchain(destDir); err != nil {
			return errors.Wrap(err, "failed to download toolchain")
		}
		log.Info("Unpacking toolchain...")
		// TODO (rnv): exclude rust dirs during unpacking toolchain archive
		if err := exec.Command("tar", "xf", filepath.Join(destDir, toolchainArchive), "-C", destDir).Run(); err != nil {
			return errors.Wrap(err, "failed to extract toolchain archive tarball")
		}
		log.Info("Done unpacking toolchain")
	}

	if foundKernelHeaders {
		log.Info("Found existing kernel headers. Skipping download and installation.")

	} else {
		if err := downloader.DownloadKernelHeaders(destDir); err != nil {
			return fmt.Errorf("failed to download kernel headers: %v", err)
		}
		log.Info("Unpacking kernel headers...")
		if err := exec.Command("tar", "xf", filepath.Join(destDir, kernelHeaders), "-C", destDir).Run(); err != nil {
			return fmt.Errorf("failed to extract kernel headers: %v", err)
		}
		log.Info("Done unpacking kernel headers")
	}

	log.V(2).Info("Configuring environment variables for cross-compilation")
	os.Setenv("PATH", fmt.Sprintf("%s/bin:%s", destDir, os.Getenv("PATH")))
	os.Setenv("SYSROOT", filepath.Join(destDir, "usr/x86_64-cros-linux-gnu"))
	return nil
}

// InstallKernelSrcPkg installs COS kernel source package to destination directory.
func InstallKernelSrcPkg(downloader ArtifactsDownloader, destDir string) error {
	log.Info("Installing the kernel source package")

	if err := downloadKernelSrc(downloader, destDir); err != nil {
		return errors.Wrap(err, "failed to download kernel source")
	}

	if err := configureKernel(destDir); err != nil {
		return errors.Wrap(err, "failed to configure kernel source")
	}

	if err := correctKernelMagicVersionIfNeeded(destDir); err != nil {
		return errors.Wrap(err, "failed to run correctKernelMagicVersionIfNeeded")
	}

	return nil
}

// ConfigureModuleSymvers copys Module.symvers file from kernel header dir to kernel source dir.
func ConfigureModuleSymvers(kernelHeaderDir, kernelSrcDir string) error {
	log.Info("Configuring Module.symvers file")
	if err := utils.CopyFile(filepath.Join(kernelHeaderDir, "Module.symvers"),
		filepath.Join(kernelSrcDir, "Module.symvers")); err != nil {
		return errors.Wrap(err, "failed to copy Module.symvers file")
	}
	return nil
}

func disableKernelOptionFromGrubCfg(kernelOption, grubCfg string) (newGrubCfg string, needReboot bool) {
	newGrubCfg = grubCfg
	needReboot = false
	kernelOptionEnabled := fmt.Sprintf("%v=1", kernelOption)
	kernelOptionDisabled := fmt.Sprintf("%v=0", kernelOption)

	if strings.Contains(grubCfg, kernelOption) {
		if strings.Contains(grubCfg, kernelOptionEnabled) {
			newGrubCfg = strings.ReplaceAll(grubCfg, kernelOptionEnabled, kernelOptionDisabled)
			needReboot = true
		}
	} else {
		newGrubCfg = strings.ReplaceAll(grubCfg, "cros_efi", fmt.Sprintf("cros_efi %v", kernelOptionDisabled))
		needReboot = true
	}
	return newGrubCfg, needReboot
}

func downloadKernelSrc(downloader ArtifactsDownloader, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create dir %s", destDir)
	}

	if empty, _ := utils.IsDirEmpty(destDir); !empty {
		return nil
	}

	log.Info("Kernel sources not found locally, downloading")
	if err := downloader.DownloadKernelSrc(destDir); err != nil {
		return errors.Wrap(err, "failed to download kernel sources")
	}
	if err := exec.Command("tar", "xf", filepath.Join(destDir, kernelSrcArchive), "-C", destDir).Run(); err != nil {
		return errors.Wrap(err, "failed to extract kernel source tarball")
	}

	return nil
}

func configureKernel(kernelSrcDir string) error {
	log.Info("Configuring kernel")
	// TODO(mikewu): consider getting kernel configs from kernel headers.
	kConfig, err := exec.Command("zcat", "/proc/config.gz").Output()
	if err != nil {
		return errors.Wrap(err, "failed to read kernel config")
	}
	if err := ioutil.WriteFile(filepath.Join(kernelSrcDir, ".config"), kConfig, 0644); err != nil {
		return errors.Wrap(err, "failed to write kernel config file")
	}
	cmd := exec.Command("make", "olddefconfig")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = kernelSrcDir
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to run `make olddefconfig`")
	}

	cmd = exec.Command("make", "modules_prepare")
	cmd.Dir = kernelSrcDir
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to run `make modules_prepare`")
	}

	// COS doesn't enable module versioning, disable Module.symvers file check.
	os.Setenv("IGNORE_MISSING_MODULE_SYMVERS", "1")
	return nil
}

func correctKernelMagicVersionIfNeeded(kernelSrcDir string) error {
	// Normally COS kernel release version has a "+" in the end, e.g. "4.19.102+". But
	// the utsrelease file generated here doesn't have it, e.g. "4.19.102". Thus we need
	// to correct the utsrelease file to make it match the real COS kernel release version.
	utsCmd, err := execCommand("uname", "-r").Output()
	if err != nil {
		return errors.Wrap(err, "failed to run `uname -r`")
	}
	kernelVersionCmd := strings.TrimSpace(string(utsCmd))
	utsFile, err := ioutil.ReadFile(filepath.Join(kernelSrcDir, utsFilepath))
	if err != nil {
		return errors.Wrap(err, "failed to read utsrelease file")
	}

	kernelVersionFile := strings.Trim(strings.Fields(string(utsFile))[2], `"`)
	if kernelVersionCmd != kernelVersionFile {
		newUtsFile := strings.ReplaceAll(string(utsFile), kernelVersionFile, kernelVersionCmd)
		log.Info("Modifying kernel release version magic string in source files")
		if err := ioutil.WriteFile(filepath.Join(kernelSrcDir, utsFilepath), []byte(newUtsFile), 0644); err != nil {
			return errors.Wrap(err, "failed to write to utsrelease file")
		}
	}
	return nil
}

func ForceSymlinkLinker(symlinkPath string) error {
	if err := symlinkLinker(symlinkPath); err != nil {
		// remove old symlink and try again
		if _, err := os.Lstat(symlinkPath); err == nil {
			if os.Remove(symlinkPath); err != nil {
				return errors.Wrap(err, "failed to unlink")
			}
		} else if os.IsNotExist(err) {
			return errors.Wrap(err, "failed to check symlink")
		}
		return symlinkLinker(symlinkPath)
	}
	return nil
}

func symlinkLinker(symlinkPath string) error {
	return os.Symlink("x86_64-cros-linux-gnu-ld", symlinkPath)
}

// Creates a CC wrapper and adds it to the env PATH. Newer versions of LLVM have
// different -Wstrict-prototypes behavior that impacts the nvidia installer. The
// kernel always uses -Werror=strict-prototypes by default. This wrapper removes
// -Werror=strict-prototypes from the CC command line.
func AddCCWrapperToPath(toolchainDir, workDir, cc string) error {
	wrapper := `#!/bin/bash
for arg; do
  shift
  if [[ "${arg}" == "-Werror=strict-prototypes" ]]; then continue; fi
  set -- "$@" "${arg}"
done
`
	wrapper += fmt.Sprintf("exec %s/bin/%s $@", toolchainDir, cc)
	if err := os.WriteFile(filepath.Join(workDir, cc), []byte(wrapper), 0755); err != nil {
		return err
	}
	log.V(2).Info("Creating a CC wrapper removing -Werror=strict-prototypes from the CC command line.")
	log.V(2).Info(wrapper)
	os.Setenv("PATH", fmt.Sprintf("%s:%s", workDir, os.Getenv("PATH")))
	return nil
}
