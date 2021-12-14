package cos

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

const packageInfoDefaultJSONFile = "/etc/cos-package-info.json"

// Package represents a COS package. For example, this schema is used in
// the cos-package-info.json file.
type Package struct {
	Category      string `json:"category"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	EbuildVersion string `json:"ebuild_version"`
}

// PackageInfo contains information about the packages of a COS instance.
// For example, this schema is used in the cos-package-info.json file.
type PackageInfo struct {
	InstalledPackages []Package `json:"installedPackages"`
	BuildTimePackages []Package `json:"buildTimePackages"`
}

// PackageInfoExists returns whether COS package information exists on the
// local OS.
func PackageInfoExists() bool {
	info, err := os.Stat(packageInfoDefaultJSONFile)
	return !os.IsNotExist(err) && !info.IsDir()
}

// GetPackageInfo loads the package information from the local OS and returns
// it.
func GetPackageInfo() (PackageInfo, error) {
	return GetPackageInfoFromFile(packageInfoDefaultJSONFile)
}

// GetPackageInfoFromFile loads the package information from the specified file
// on the local OS and returns it.
func GetPackageInfoFromFile(filename string) (PackageInfo, error) {
	var packageInfo PackageInfo

	f, err := os.Open(filename)
	if err != nil {
		return packageInfo, err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return packageInfo, err
	}

	var pi PackageInfo
	if err = json.Unmarshal(b, &pi); err != nil {
		return packageInfo, err
	}
	return pi, nil
}
