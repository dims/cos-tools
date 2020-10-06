package cos

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/golang/glog"
	"github.com/pkg/errors"

	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

const (
	// TODO(mikewu): consider making GCS buckets as flags.
	cosToolsGCS      = "cos-tools"
	internalGCS      = "container-vm-image-staging"
	chromiumOSSDKGCS = "chromiumos-sdk"
	kernelInfo       = "kernel_info"
	kernelSrcArchive = "kernel-src.tar.gz"
	kernelHeaders    = "kernel-headers.tgz"
	toolchainURL     = "toolchain_url"
	toolchainArchive = "toolchain.tar.xz"
	toolchainEnv     = "toolchain_env"
	crosKernelRepo   = "https://chromium.googlesource.com/chromiumos/third_party/kernel"
)

// ArtifactsDownloader defines the interface to download COS artifacts.
type ArtifactsDownloader interface {
	DownloadKernelSrc(destDir string) error
	DownloadToolchainEnv(destDir string) error
	DownloadToolchain(destDir string) error
	DownloadKernelHeaders(destDir string) error
	DownloadArtifact(destDir, artifact string) error
	GetArtifact(artifact string) ([]byte, error)
}

// GCSDownloader is the struct downloading COS artifacts from GCS bucket.
type GCSDownloader struct {
	envReader *EnvReader
	Internal  bool
}

// NewGCSDownloader creates a GCSDownloader instance.
func NewGCSDownloader(e *EnvReader, i bool) *GCSDownloader {
	return &GCSDownloader{e, i}
}

// DownloadKernelSrc downloads COS kernel sources to destination directory.
func (d *GCSDownloader) DownloadKernelSrc(destDir string) error {
	return d.DownloadArtifact(destDir, kernelSrcArchive)
}

// DownloadToolchainEnv downloads toolchain compilation environment variables to destination directory.
func (d *GCSDownloader) DownloadToolchainEnv(destDir string) error {
	return d.DownloadArtifact(destDir, toolchainEnv)
}

// DownloadToolchain downloads toolchain package to destination directory.
func (d *GCSDownloader) DownloadToolchain(destDir string) error {
	downloadURL, err := d.getToolchainURL()
	if err != nil {
		return errors.Wrap(err, "failed to download toolchain")
	}
	outputPath := filepath.Join(destDir, toolchainArchive)
	if err := utils.DownloadContentFromURL(downloadURL, outputPath, toolchainArchive); err != nil {
		return errors.Wrap(err, "failed to download toolchain")
	}
	return nil
}

// DownloadKernelHeaders downloads COS kernel headers to destination directory.
func (d *GCSDownloader) DownloadKernelHeaders(destDir string) error {
	return d.DownloadArtifact(destDir, kernelHeaders)
}

// GetArtifact gets an artifact from GCS buckets and returns its content.
func (d *GCSDownloader) GetArtifact(artifactPath string) ([]byte, error) {
	tmpDir, err := ioutil.TempDir("", "tmp")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tmpDir)

	if err = d.DownloadArtifact(tmpDir, artifactPath); err != nil {
		return nil, errors.Wrapf(err, "failed to download artifact %s", artifactPath)
	}

	content, err := ioutil.ReadFile(filepath.Join(tmpDir, filepath.Base(artifactPath)))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", filepath.Join(tmpDir, artifactPath))
	}

	return content, nil
}

// DownloadArtifact downloads an artifact from GCS buckets, including public bucket and internal bucket.
// TODO(mikewu): consider allow users to pass in GCS directories in arguments.
func (d *GCSDownloader) DownloadArtifact(destDir, artifactPath string) error {
	var err error

	if err = utils.DownloadFromGCS(destDir, cosToolsGCS, d.artifactPublicPath(artifactPath)); err == nil {
		return nil
	}
	log.Errorf("Failed to download %s from public GCS: %v", artifactPath, err)

	if d.Internal {
		if err = utils.DownloadFromGCS(destDir, internalGCS, d.artifactInternalPath(artifactPath)); err == nil {
			return nil
		}
		log.Errorf("Failed to download %s from internal GCS: %v", artifactPath, err)
	}

	return errors.Errorf("failed to download %s", artifactPath)
}

func (d *GCSDownloader) artifactPublicPath(artifactPath string) string {
	return fmt.Sprintf("%s/%s", d.envReader.BuildNumber(), artifactPath)
}

func (d *GCSDownloader) artifactInternalPath(artifactPath string) string {
	return fmt.Sprintf("lakitu-release/R%s-%s/%s", d.envReader.Milestone(), d.envReader.BuildNumber(), artifactPath)
}

func (d *GCSDownloader) getToolchainURL() (string, error) {
	// First, check if the toolchain path is available locally
	tcPath := d.envReader.ToolchainPath()
	if tcPath != "" {
		log.V(2).Info("Found toolchain path file locally")
		return fmt.Sprintf("https://storage.googleapis.com/%s/%s", chromiumOSSDKGCS, tcPath), nil
	}

	// Next, check if the toolchain path is available in GCS.
	tmpDir, err := ioutil.TempDir("", "temp")
	if err != nil {
		return "", errors.Wrap(err, "failed to create tmp dir")
	}
	defer os.RemoveAll(tmpDir)
	if err := d.DownloadArtifact(tmpDir, toolchainURL); err != nil {
		return "", err
	}
	toolchainURLContent, err := ioutil.ReadFile(filepath.Join(tmpDir, toolchainURL))
	if err != nil {
		return "", errors.Wrap(err, "failed to read toolchain URL file")
	}
	return string(toolchainURLContent), nil
}
