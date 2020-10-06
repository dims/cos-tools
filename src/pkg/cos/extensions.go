package cos

import (
	"fmt"
	"path/filepath"
	"regexp"

	"cos.googlesource.com/cos/tools.git/src/pkg/utils"

	log "github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	// GPUExtension is the name of GPU extension.
	GPUExtension = "gpu"
)

// ExtensionsDownloader is the struct downloading COS extensions from GCS bucket.
type ExtensionsDownloader interface {
	ListExtensions() ([]string, error)
	ListExtensionArtifacts(extension string) ([]string, error)
	DownloadExtensionArtifact(destDir, extension, artifact string) error
	GetExtensionArtifact(extension, artifact string) ([]byte, error)
}

// ListExtensions lists all supported extensions.
func (d *GCSDownloader) ListExtensions() ([]string, error) {
	var objects []string
	var err error
	if objects, err = utils.ListGCSBucket(cosToolsGCS, d.artifactPublicPath("extensions")); err != nil || len(objects) == 0 {
		log.Errorf("Failed to list extensions from public GCS: %v", err)
		if d.Internal {
			if objects, err = utils.ListGCSBucket(internalGCS, d.artifactInternalPath("extensions")); err != nil {
				log.Errorf("Failed to list extensions from internal GCS: %v", err)
			}
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to list extensions")
	}

	var extensions []string
	re := regexp.MustCompile(`extensions/(\w+)$`)
	for _, object := range objects {
		if match := re.FindStringSubmatch(object); match != nil {
			extensions = append(extensions, match[1])
		}
	}
	return extensions, nil
}

// ListExtensionArtifacts lists all artifacts of a given extension.
// TODO(mikewu): make this extension specific.
func (d *GCSDownloader) ListExtensionArtifacts(extension string) ([]string, error) {
	var objects []string
	var err error
	extensionPath := filepath.Join("extensions", extension)
	if objects, err = utils.ListGCSBucket(cosToolsGCS, d.artifactPublicPath(extensionPath)); err != nil || len(objects) == 0 {
		log.Errorf("Failed to list extension artifacts from public GCS: %v", err)
		// TODO(mikewu): use flags to specify GCS directories.
		if d.Internal {
			if objects, err = utils.ListGCSBucket(internalGCS, d.artifactInternalPath(extensionPath)); err != nil {
				log.Errorf("Failed to list extension artifacts from internal GCS: %v", err)
			}
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to list extensions")
	}

	var artifacts []string
	re := regexp.MustCompile(fmt.Sprintf(`extensions/%s/(.+)$`, extension))
	for _, object := range objects {
		if match := re.FindStringSubmatch(object); match != nil {
			artifacts = append(artifacts, match[1])
		}
	}
	return artifacts, nil
}

// DownloadExtensionArtifact downloads an artifact of the given extension.
func (d *GCSDownloader) DownloadExtensionArtifact(destDir, extension, artifact string) error {
	artifactPath := filepath.Join("extensions", extension, artifact)
	return d.DownloadArtifact(destDir, artifactPath)
}

// GetExtensionArtifact reads the content of an artifact of the given extension.
func (d *GCSDownloader) GetExtensionArtifact(extension, artifact string) ([]byte, error) {
	artifactPath := filepath.Join("extensions", extension, artifact)
	return d.GetArtifact(artifactPath)
}
