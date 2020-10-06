package packagediff

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_image_analyzer/internal/input"
)

const pathToPackageList = "/etc/package_list"

// Package is used to store individual package data parsed from the package list json file
type Package struct {
	Category string
	Name     string
	Version  string
	Revision string
}

// InstalledPackages is used to store an image’s full package list parsed from the package list json file
type InstalledPackages struct {
	InstalledPackages []Package
}

// ****** NOTE ******
// This function is a temporary implementation. Switch this out with the awaited cos-tools library function.
// getInstalledPackages returns the package list for an image by parsing its /etc/package_list json file
func getInstalledPackages(rootfs string) ([]Package, error) {
	fullPath := filepath.Join(rootfs, pathToPackageList)
	packageListBytes, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return []Package{}, fmt.Errorf("failed to read package list file %v: %v", fullPath, err)
	}
	var IP InstalledPackages
	if err := json.Unmarshal(packageListBytes, &IP); err != nil {
		return []Package{}, fmt.Errorf("failed to parse json for package list file %v: %v", fullPath, err)
	}
	return IP.InstalledPackages, nil
}

// ******************

// GetPackageInfo finds relevant package list information for the COS image
func GetPackageInfo(image *input.ImageInfo, flagInfo *input.FlagInfo) ([]Package, error) {
	if image.TempDir == "" {
		return []Package{}, nil
	}

	if flagInfo.PackageSelected { // Get package list from /etc/package_list
		packageList, err := getInstalledPackages(image.RootfsPartition3)
		if err != nil {
			return []Package{}, fmt.Errorf("failed to get package list from image %v: %v", image.TempDir, err)
		}
		return packageList, nil
	}
	return []Package{}, nil
}
