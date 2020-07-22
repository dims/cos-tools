// Package signing provides functionality to manage GPU driver signatures for COS.
package signing

import (
	"os"
	"os/exec"
	"path/filepath"

	"cos.googlesource.com/cos/tools/src/pkg/cos"
	log "github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	gpuDriverPubKeyPem = "gpu-driver-cert.pem"
	gpuDriverPubKeyDer = "gpu-driver-cert.der"
	gpuDriverDummyKey  = "dummy-key"
)

var (
	gpuDriverSigningDir = "/build/sign-gpu-driver"
)

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

	tarballPath := filepath.Join(gpuDriverSigningDir, driverVersion+".signature.tar.gz")
	log.Infof("Decompressing signature %s", tarballPath)
	if err := extractSignatures(tarballPath, gpuDriverSigningDir); err != nil {
		return errors.Wrapf(err, "failed to extract driver signatures for version %s", driverVersion)
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
