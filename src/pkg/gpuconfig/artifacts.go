package gpuconfig

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"

	"log"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/gcs"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

type GPUArtifactsDownloader struct {
	ctx    context.Context
	client *storage.Client
	config GPUPrecompilationConfig
}

// NewGPUArtifactsDownloader creates a GPUArtifactsDownloader instance.
func NewGPUArtifactsDownloader(ctx context.Context, client *storage.Client, config GPUPrecompilationConfig) *GPUArtifactsDownloader {
	return &GPUArtifactsDownloader{ctx, client, config}
}

// DownloadKernelSrc downloads COS kernel sources to destination directory.
func (d *GPUArtifactsDownloader) DownloadKernelSrc(destDir string) error {
	return d.downloadArtifact(destDir, d.config.ProtoConfig.GetKernelSrcTarballGcs(), "kernel-src.tar.gz")
}

// DownloadToolchainEnv downloads toolchain compilation environment variables to destination directory.
func (d *GPUArtifactsDownloader) DownloadToolchainEnv(destDir string) error {
	return d.downloadArtifact(destDir, d.config.ProtoConfig.GetToolchainEnvGcs(), "toolchain_env")
}

// DownloadToolchain downloads toolchain package to destination directory.
func (d *GPUArtifactsDownloader) DownloadToolchain(destDir string) error {
	return d.downloadArtifact(destDir, d.config.ProtoConfig.GetToolchainTarballGcs(), "toolchain.tar.xz")
}

// DownloadKernelHeaders downloads COS kernel headers to destination directory.
func (d *GPUArtifactsDownloader) DownloadKernelHeaders(destDir string) error {
	return d.downloadArtifact(destDir, d.config.ProtoConfig.GetKernelHeadersTarballGcs(), "kernel-headers.tgz")
}

func (d *GPUArtifactsDownloader) GetArtifact(artifact string) ([]byte, error) {
	return nil, nil
}

func (d *GPUArtifactsDownloader) DownloadNVIDIARunfile(destDir string) (string, error) {
	url, err := url.Parse(d.config.ProtoConfig.GetNvidiaRunfileAddress())
	if err != nil {
		return "", fmt.Errorf("error parsing the artifact path: %v", err)
	}
	nvidiaInstaller := path.Base(url.Path)
	if err := d.downloadArtifact(destDir, url.String(), nvidiaInstaller); err != nil {
		return "", err
	}
	return nvidiaInstaller, nil
}

// DownloadArtifact downloads an artifact from the GCS prefix configured in GPUArtifactsDownloader.
func (d *GPUArtifactsDownloader) DownloadArtifact(destDir, artifactPath string) error {
	return nil
}

func (d *GPUArtifactsDownloader) downloadArtifact(destDir, artifactPath, fileName string) error {
	log.Printf("downloading artifact from:%s\n", artifactPath)
	url, err := url.Parse(artifactPath)
	if err != nil {
		return fmt.Errorf("error parsing the artifact path: %v", err)
	}

	switch url.Scheme {
	case "gs":
		return gcs.DownloadGCSObject(d.ctx, d.client, artifactPath, filepath.Join(destDir, fileName))
	case "https":
		return utils.DownloadContentFromURL(artifactPath, filepath.Join(destDir, fileName), fileName)
	default:
		return fmt.Errorf("only https:// or gs:// urls supported: %s", url)
	}
}
