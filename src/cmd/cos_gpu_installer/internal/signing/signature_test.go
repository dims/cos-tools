package signing

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"pkg/utils"
)

func TestDownloadDriverSignatures(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	origGpuDriverSigningDir := gpuDriverSigningDir
	gpuDriverSigningDir = tmpDir
	defer func(origGpuDriverSigningDir string) { gpuDriverSigningDir = origGpuDriverSigningDir }(origGpuDriverSigningDir)

	downloader := fakeDownloader{}
	if err := DownloadDriverSignatures(&downloader, "418.87.01"); err != nil {
		t.Fatalf("Failed to run DownloadDriverSignatures: %v", err)
	}

	// Verify downloaded signature
	for _, tc := range []struct {
		fn              func() string
		expectedContent string
	}{
		{
			GetPublicKeyPem,
			"pubkey.pem",
		},
		{
			GetPublicKeyDer,
			"pubkey.der",
		},
		{
			GetPrivateKey,
			"",
		},
	} {
		f := tc.fn()
		content, err := ioutil.ReadFile(f)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", f, err)
		}
		if string(content) != tc.expectedContent {
			t.Errorf("Unexpected content of %s: want: %s, got: %s", funcName(tc.fn), tc.expectedContent, string(content))
		}
	}
}

func funcName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

type fakeDownloader struct {
}

func (*fakeDownloader) ListExtensions() ([]string, error)                         { return nil, nil }
func (*fakeDownloader) ListExtensionArtifacts(extension string) ([]string, error) { return nil, nil }
func (*fakeDownloader) GetExtensionArtifact(extension, artifact string) ([]byte, error) {
	return nil, nil
}

func (f *fakeDownloader) DownloadExtensionArtifact(destDir, extension, artifact string) error {
	var archive = map[string][]byte{
		gpuDriverPubKeyPem: []byte("pubkey.pem"),
		gpuDriverPubKeyDer: []byte("pubkey.der"),
	}
	archivePath := filepath.Join(destDir, artifact)
	if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
		return fmt.Errorf("failed to create dir %s", filepath.Dir(archivePath))
	}
	if err := utils.CreateTarFile(archivePath, archive); err != nil {
		return fmt.Errorf("Failed to create tarfile: %v", err)
	}
	return nil
}
