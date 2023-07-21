// Package signing provides functionality to manage GPU driver signatures for COS.
package signing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"cos.googlesource.com/cos/tools.git/src/pkg/cos"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
	log "github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	gpuDriverPubKeyPem = "gpu-driver-cert.pem"
	gpuDriverPubKeyDer = "gpu-driver-cert.der"
	gpuDriverDummyKey  = "dummy-key"
	signatureTemplate  = "nvidia-drivers-%s-signature.tar.gz"
)

var (
	gpuDriverSigningDir = "/build/sign-gpu-driver"
)

// DownloadDriverSignaturesV2 downloads GPU driver signatures from COS build artifacts.
func DownloadDriverSignaturesV2(downloader *cos.GCSDownloader, driverVersion string) error {
	if err := os.MkdirAll(gpuDriverSigningDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create signing dir %s", gpuDriverSigningDir)
	}
	log.Infof("Downloading driver signature for version %s", driverVersion)
	signatureName := fmt.Sprintf(signatureTemplate, driverVersion)
	if err := downloader.DownloadArtifact(gpuDriverSigningDir, signatureName); err != nil {
		return errors.Wrapf(err, "failed to download driver signature for version %s", driverVersion)
	}

	if err := decompressSignature(signatureName); err != nil {
		return errors.Wrapf(err, "failed to decompress driver signature for version %s.", driverVersion)
	}

	return nil
}

// DownloadDriverSignatures downloads GPU driver signatures.
func DownloadDriverSignatures(downloader cos.ExtensionsDownloader, driverVersion string) error {
	if err := os.MkdirAll(gpuDriverSigningDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create signing dir %s", gpuDriverSigningDir)
	}

	log.Infof("Downloading driver signature for version %s", driverVersion)
	if err := downloader.DownloadExtensionArtifact(
		gpuDriverSigningDir, cos.GPUExtension, driverVersion+".signature.tar.gz"); err != nil {
		return errors.Wrapf(err, "failed to download driver signature for version %s", driverVersion)
	}

	if err := decompressSignature(driverVersion + ".signature.tar.gz"); err != nil {
		return errors.Wrapf(err, "failed to decompress driver signature for version %s.", driverVersion)
	}

	return nil
}

// DownloadDriverSignaturesFromURL downloads GPU driver signatures from a provided URL.
func DownloadDriverSignaturesFromURL(signatureURL string) error {
	if err := os.MkdirAll(gpuDriverSigningDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create signing dir %s", gpuDriverSigningDir)
	}

	log.Infof("Downloading driver signature from URL: %s", signatureURL)
	signatureName := filepath.Base(signatureURL)
	outputPath := filepath.Join(gpuDriverSigningDir, signatureName)
	if err := utils.DownloadContentFromURL(signatureURL, outputPath, "driver signature"); err != nil {
		return errors.Wrapf(err, "failed to download driver signature from URL %s.", signatureURL)
	}

	if err := decompressSignature(signatureName); err != nil {
		return errors.Wrapf(err, "failed to decompress driver signature: %s.", signatureName)
	}

	return nil
}

func decompressSignature(signatureName string) error {
	tarballPath := filepath.Join(gpuDriverSigningDir, signatureName)
	log.Infof("Decompressing signature %s", tarballPath)
	if err := extractSignatures(tarballPath, gpuDriverSigningDir); err != nil {
		return errors.Wrapf(err, "failed to extract driver signatures %s", signatureName)
	}

	// Create a dummy private key. We don't need private key to sign the driver
	// because we already have the signature.
	f, err := os.Create(filepath.Join(gpuDriverSigningDir, gpuDriverDummyKey))
	if err != nil {
		return errors.Wrapf(err, "failed to create dummy key file in %s", filepath.Join(gpuDriverSigningDir, gpuDriverDummyKey))
	}
	if err := f.Close(); err != nil {
		return errors.Wrapf(err, "failed to close dummy key file in %s", filepath.Join(gpuDriverSigningDir, gpuDriverDummyKey))
	}
	return nil
}

// GetPrivateKey returns the filepath of the private key of a given GPU driver.
// This is a dummy key as the driver has been signed in advance.
func GetPrivateKey() string {
	return filepath.Join(gpuDriverSigningDir, gpuDriverDummyKey)
}

// GetPublicKeyPem returns the filepath of the public key in pem format.
func GetPublicKeyPem() string {
	return filepath.Join(gpuDriverSigningDir, gpuDriverPubKeyPem)
}

// GetPublicKeyDer returns the filepath of the public key in der format.
func GetPublicKeyDer() string {
	return filepath.Join(gpuDriverSigningDir, gpuDriverPubKeyDer)
}

// GetModuleSignature returns siganture path given kernel module name.
func GetModuleSignature(moduleName string) string {
	return filepath.Join(gpuDriverSigningDir, moduleName+".sig")
}

func extractSignatures(tarballPath, destPath string) error {
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return errors.Wrapf(err, "failed to create dir %s", destPath)
	}
	if err := exec.Command("tar", "xf", tarballPath, "-C", destPath).Run(); err != nil {
		return errors.Wrap(err, "failed to extract driver signatures")
	}
	return nil
}
