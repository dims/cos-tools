package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/cos"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

const (
	linkerLocation        = "/bin/ld"
	installDirTemplate    = "/install/%s"
	defaultFilePermission = 0755
	kernelSrcTemplate     = "usr/src/linux-headers-*"
)

func kernelSrcDirectory(dirName string) string {
	files, err := filepath.Glob(filepath.Join(dirName, kernelSrcTemplate))
	if err != nil || len(files) != 1 {
		return ""
	}
	return files[0]
}

func nvidiaInstallerCommand(dirName, runfile string, config gpuconfig.GPUPrecompilationConfig) *exec.Cmd {

	cmd := exec.Command(filepath.Join(dirName, runfile), "--kernel-source-path="+kernelSrcDirectory(dirName), "--add-this-kernel", "--no-install-compat32-libs", "--silent", "--accept-license")
	cmd.Dir = dirName
	return cmd
}

func BuildPrecompiledDriver(ctx context.Context, client *storage.Client, config gpuconfig.GPUPrecompilationConfig) (string, string, error) {
	var err error
	dirName := fmt.Sprintf(installDirTemplate, config.Version)
	if err = os.MkdirAll(dirName, defaultFilePermission); err != nil {
		return "", "", fmt.Errorf("failed to create installation dir: %v", err)
	}
	downloader := gpuconfig.NewGPUArtifactsDownloader(ctx, client, config)
	// download NVIDIA runfile
	var nvidiaInstaller string
	if nvidiaInstaller, err = downloader.DownloadNVIDIARunfile(dirName); err != nil {
		return "", "", fmt.Errorf("failed to download NVIDIA runfile: %v", err)
	}
	// install kernel headers and toolchain
	// sets SYSROOT and PATH env vars
	if err = cos.InstallCrossToolchain(downloader, dirName); err != nil {
		return "", "", fmt.Errorf("failed to install toolchain: %v", err)
	}
	// set CC CXX env vars from toolchain_env
	if err = cos.SetCompilationEnv(downloader); err != nil {
		return "", "", fmt.Errorf("failed to set compilation env vars: %v", err)
	}
	// create symlink to ld - required by NVIDIA driver package
	if err = cos.ForceSymlinkLinker(filepath.Join(dirName, linkerLocation)); err != nil {
		return "", "", fmt.Errorf("failed to create symlink to COS linker: %v", err)
	}
	cc := os.Getenv("CC")
	if cc == "" {
		return "", "", fmt.Errorf("failed to find CC in env")
	} else {
		// create a wrapper removing -Werror=strict-prototypes from the CC command line.
		if err = cos.AddCCWrapperToPath(dirName, dirName, cc); err != nil {
			return "", "", fmt.Errorf("failed to create CC wrapper: %v", err)
		}
	}
	// run NVIDIA driver package
	if err = os.Chmod(filepath.Join(dirName, nvidiaInstaller), defaultFilePermission); err != nil {
		return "", "", err
	}
	cmd := nvidiaInstallerCommand(dirName, nvidiaInstaller, config)
	if err = utils.RunCommandAndLogOutput(cmd, false); err != nil {
		return "", "", fmt.Errorf("error running NVIDIA driver installation package: %v", err)
	}

	outputFileName := strings.Split(nvidiaInstaller, ".run")[0] + "-custom.run"
	return dirName, outputFileName, nil
}
