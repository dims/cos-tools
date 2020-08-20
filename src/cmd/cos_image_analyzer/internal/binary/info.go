package binary

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

const cosGCSBucket = "cos-tools"
const kernelHeaderGCSObject = "kernel-headers.tgz"
const pathToKernelConfigs = "usr/src/linux-headers-4.19.112+/.config"

const pathToKernelCommandLine = "efi/boot/grub.cfg" // Located in partition 12 EFI
const kclImageName = "verified image A"
const startOfHashingKCL = "dm="

const pathToSysctlSettings = "/etc/sysctl.d/00-sysctl.conf" // Located in partition 3 Root-A

// getPartitionStructure returns the partition structure of .raw file
func getPartitionStructure(image *input.ImageInfo) error {
	if image.TempDir == "" {
		return nil
	}

	out, err := exec.Command("sudo", "sgdisk", "-p", image.DiskFile).Output()
	if err != nil {
		return fmt.Errorf("failed to call sgdisk -p %v: %v", image.DiskFile, err)
	}

	partitionFile := filepath.Join(image.TempDir, "partitions.txt")
	if err := utilities.WriteToNewFile(partitionFile, string(out[:])); err != nil {
		return fmt.Errorf("failed create file %v and write %v: %v", partitionFile, string(out[:]), err)
	}
	image.PartitionFile = partitionFile
	return nil
}

// getKernelConfigs downloads the kernel configs for a build from GCS and stores
// it into the image's temporary directory
func getKernelConfigs(image *input.ImageInfo) error {
	gcsObject := filepath.Join(image.BuildID, kernelHeaderGCSObject)
	tarFile, err := utilities.GcsDowndload(cosGCSBucket, gcsObject, image.TempDir, kernelHeaderGCSObject, false)
	if err != nil {
		return fmt.Errorf("failed to download GCS object %v from bucket %v: %v", gcsObject, cosGCSBucket, err)
	}

	_, err = exec.Command("tar", "-xf", tarFile, "-C", image.TempDir).Output()
	if err != nil {
		return fmt.Errorf("failed to unzip %v into %v: %v", tarFile, image.TempDir, err)
	}
	image.KernelConfigsFile = filepath.Join(image.TempDir, pathToKernelConfigs)
	return nil
}

// getKernelCommandLine gets the kernel command line from the image's partition 12 EFI
// located in the /efi/boot/grub.cfg file
func getKernelCommandLine(image *input.ImageInfo) error {
	kclPath := filepath.Join(image.EFIPartition12, pathToKernelCommandLine)
	kclFile, err := os.Open(kclPath)
	if err != nil {
		return fmt.Errorf("Failed to open file %v: %v", kclPath, err)
	}
	defer kclFile.Close()

	foundKCL := false
	scanner := bufio.NewScanner(kclFile)
	for scanner.Scan() { // Scan file line by line for "verified Image A"
		kcl := string(scanner.Text()[:])

		if foundKCL {
			if hashStart := strings.Index(kcl, startOfHashingKCL); hashStart >= 0 {
				kcl = kcl[:hashStart] // Remove hash "dm='....'" from kcl
			}
			image.KernelCommandLine = strings.TrimSpace(kcl)
			return nil
		}
		if strings.Contains(kcl, kclImageName) {
			foundKCL = true
		}
	}

	if scanner.Err() != nil {
		return fmt.Errorf("Failed to scan file %v: %v", kclPath, scanner.Err())
	}
	return nil
}

// getSysctlSettings finds an image's Sysctrl settings file under
// the /etc/sysctrl.d/00-sysctl.conf
func getSysctlSettings(image *input.ImageInfo) error {
	sysctlPath := filepath.Join(image.RootfsPartition3, pathToSysctlSettings)
	image.SysctlSettingsFile = sysctlPath
	return nil
}

// GetBinaryInfo finds relevant binary information for the COS image
func GetBinaryInfo(image *input.ImageInfo, flagInfo *input.FlagInfo) error {
	if image.TempDir == "" {
		return nil
	}

	if image.RootfsPartition3 != "" { // Get Version and BuildID
		osReleaseMap, err := utilities.ReadFileToMap(image.RootfsPartition3+etcOSRelease, "=")
		if err != nil {
			return fmt.Errorf("Failed to read /etc/os-release file in rootfs of image %v : %v", image.TempDir, err)
		}
		var ok bool
		if image.Version, ok = osReleaseMap["VERSION"]; !ok {
			return errors.New("Error: \"Version\" field not found in /etc/os-release file")
		}
		if image.BuildID, ok = osReleaseMap["BUILD_ID"]; !ok {
			return errors.New("Error: \"Build_ID\" field not found in /etc/os-release file")
		}
	}

	if utilities.InArray("Partition-structure", flagInfo.BinaryTypesSelected) { // Get partition structure from "sgdisk -p"
		if err := getPartitionStructure(image); err != nil {
			return fmt.Errorf("failed to get partition structure for image %v: %v", image.TempDir, err)
		}
	}

	if utilities.InArray("Kernel-configs", flagInfo.BinaryTypesSelected) { // Get kernel configs from gs://cos-tools/BuildID/kernel-headers.tgz
		if err := getKernelConfigs(image); err != nil {
			return fmt.Errorf("failed to get kernel configs for image %v: %v", image.TempDir, err)
		}
	}

	if utilities.InArray("Kernel-command-line", flagInfo.BinaryTypesSelected) { // Get kernel command line from partition 12 EFI (efi/boot/grub.cfg)
		if err := getKernelCommandLine(image); err != nil {
			return fmt.Errorf("failed to get the kernel command line for image %v: %v", image.TempDir, err)
		}
	}

	if utilities.InArray("Sysctl-settings", flagInfo.BinaryTypesSelected) { // Get Sysctl settings from /etc/sysctl.d/00-sysctl.conf
		if err := getSysctlSettings(image); err != nil {
			return fmt.Errorf("failed to get Sysctl-settings for image %v: %v", image.TempDir, err)
		}
	}
	return nil
}
