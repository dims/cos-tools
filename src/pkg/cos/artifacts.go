package cos

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cloud.google.com/go/compute/metadata"
	log "github.com/golang/glog"
	"github.com/pkg/errors"

	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

const (
	cosToolsGCS      = "cos-tools"
	cosToolsGCSAsia  = "cos-tools-asia"
	cosToolsGCSEU    = "cos-tools-eu"
	kernelInfo       = "kernel_info"
	kernelSrcArchive = "kernel-src.tar.gz"
	kernelHeaders    = "kernel-headers.tgz"
	toolchainURL     = "toolchain_url"
	toolchainArchive = "toolchain.tar.xz"
	toolchainEnv     = "toolchain_env"
	crosKernelRepo   = "https://chromium.googlesource.com/chromiumos/third_party/kernel"
)

// Map VM zone prefix to specific cos-tools bucket for geo-redundancy.
var cosToolsPrefixMap = map[string]string{
	"us":           cosToolsGCS,
	"northamerica": cosToolsGCS,
	"southamerica": cosToolsGCS,
	"europe":       cosToolsGCSEU,
	"asia":         cosToolsGCSAsia,
	"australia":    cosToolsGCSAsia,
}

// ArtifactsDownloader defines the interface to download COS artifacts.
type ArtifactsDownloader interface {
	DownloadKernelSrc(destDir string) error
	DownloadToolchainEnv(destDir string) error
	DownloadToolchain(destDir string) error
	DownloadKernelHeaders(destDir string) error
	DownloadArtifact(destDir, artifact string) error
	GetArtifact(artifact string) ([]byte, error)
	ArtifactExists(artifact string) (bool, error)
}

// GCSDownloader is the struct downloading COS artifacts from GCS bucket.
type GCSDownloader struct {
	envReader         *EnvReader
	gcsDownloadBucket string
	gcsDownloadPrefix string
}

// NewGCSDownloader creates a GCSDownloader instance.
func NewGCSDownloader(e *EnvReader, bucket, prefix string) *GCSDownloader {
	// If bucket is not set, use cos-tools, cos-tools-asia or cos-tools-eu
	// according to the zone the VM is running in for geo-redundancy.
	// If cannot fetch zone from metadata or get an unknown zone prefix,
	// use cos-tools as the default GCS bucket.
	if bucket == "" {
		zone, err := metadata.Zone()
		if err != nil {
			log.Warningf("failed to get zone from metadata, will use 'gs://cos-tools' as artifact bucket, err: %v", err)
			bucket = cosToolsGCS
		} else {
			zonePrefix := strings.Split(zone, "-")[0]
			if geoBucket, found := cosToolsPrefixMap[zonePrefix]; found {
				bucket = geoBucket
			} else {
				bucket = cosToolsGCS
			}
		}
	}
	// Use build number as the default GCS download prefix.
	if prefix == "" {
		prefix = e.BuildNumber()
	}
	return &GCSDownloader{e, bucket, prefix}
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
	return d.DownloadArtifact(destDir, toolchainArchive)
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

// DownloadArtifact downloads an artifact from the GCS prefix configured in GCSDownloader.
func (d *GCSDownloader) DownloadArtifact(destDir, artifactPath string) error {
	gcsPath := path.Join(d.gcsDownloadPrefix, artifactPath)
	if err := utils.DownloadFromGCS(destDir, d.gcsDownloadBucket, gcsPath); err != nil {
		return errors.Errorf("failed to download %s from gs://%s/%s : %s", artifactPath, d.gcsDownloadBucket, gcsPath, err)
	}
	return nil
}

func (d *GCSDownloader) ArtifactExists(artifactPath string) (bool, error) {
	var objects []string
	var err error
	if objects, err = utils.ListGCSBucket(d.gcsDownloadBucket, filepath.Join(d.gcsDownloadPrefix, artifactPath)); err != nil {
		return false, errors.Wrap(err, "failed to find artifact")
	}
	for _, object := range objects {
		if object == filepath.Join(d.gcsDownloadPrefix, artifactPath) {
			return true, nil
		}
	}
	return false, nil
}
