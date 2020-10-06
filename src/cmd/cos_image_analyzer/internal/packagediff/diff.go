package packagediff

import (
	"fmt"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_image_analyzer/internal/input"
)

// PkgDiff is used to hold package difference between the two images
type PkgDiff struct {
	category   []string
	name       []string
	version    []string
	revision   []string
	typeOFDiff string // "image1" or "image2" if package is unique to image1 or image2, "shared" if package is shared in both images
}

// Differences is an intermediate struct used to store package lists and differences
type Differences struct {
	PackageDiff []PkgDiff // If two images are passed in, this is a slice of all package differences
	PackageList []Package // If only one image is passed in, return full package list
}

// searchPackageList determines whether a package name appears in a package list
func searchPackageList(packageName string, packageList []Package) (Package, bool) {
	for _, p := range packageList {
		if p.Name == packageName {
			return p, true
		}
	}
	return Package{}, false
}

// packageListDiff calculates the package list difference the two images
// Input:
//   ([]Package) packagesImage1 - Image1's package list
//   ([]Package) packagesImage2 - Image2's package list
//   (*FlagInfo) flagInfo - A struct that holds input preference from the user
// Output: nil on success, else err
func (d *Differences) packageListDiff(packagesImage1, packagesImage2 []Package, flagInfo *input.FlagInfo) error {
	if flagInfo.Image2 != "" {
		for _, p1 := range packagesImage1 {
			pkdDiff := PkgDiff{}
			p2, ok := searchPackageList(p1.Name, packagesImage2)
			if !ok { // Unique package to image 1
				pkdDiff.typeOFDiff = "image1"
				pkdDiff.name = []string{p1.Name}
				pkdDiff.category = []string{p1.Category}
				pkdDiff.version = []string{p1.Version}
				pkdDiff.revision = []string{p1.Revision}
				d.PackageDiff = append(d.PackageDiff, pkdDiff)
			} else { // Shared package to image1 and image2
				if p1.Category != p2.Category {
					pkdDiff.category = []string{p1.Category, p2.Category}
				}
				if p1.Version != p2.Version {
					pkdDiff.version = []string{p1.Version, p2.Version}
				}
				if p1.Revision != p2.Revision {
					pkdDiff.revision = []string{p1.Revision, p2.Revision}
				}
				if len(pkdDiff.category) == 2 || len(pkdDiff.version) == 2 || len(pkdDiff.revision) == 2 {
					pkdDiff.typeOFDiff = "shared"
					pkdDiff.name = []string{p1.Name, p2.Name}
					d.PackageDiff = append(d.PackageDiff, pkdDiff)
				}
			}
		}

		for _, p2 := range packagesImage2 {
			pkdDiff := PkgDiff{}
			if _, ok := searchPackageList(p2.Name, packagesImage1); !ok { // Unique package to image2
				pkdDiff.typeOFDiff = "image2"
				pkdDiff.category = []string{p2.Category}
				pkdDiff.name = []string{p2.Name}
				pkdDiff.version = []string{p2.Version}
				pkdDiff.revision = []string{p2.Revision}
				d.PackageDiff = append(d.PackageDiff, pkdDiff)
			}
		}
	} else {
		d.PackageList = packagesImage1
	}
	return nil
}

// FormatPackageListDiff returns a formated string of the package list difference
//   (string) image1 - Temp directory name of image1
//   (string) image2 - Temp directory name of image2
func (d *Differences) FormatPackageListDiff(image1, image2 string) string {
	if len(d.PackageList) > 0 { // One image is passed in, return full package list
		pkgList := ""
		for _, p := range d.PackageList {
			pkgStr := "Package " + p.Name + "\n" + "category: " + p.Category + "\n" + "version: " + p.Version + "\n" + "revision: " + p.Revision + "\n\n"
			pkgList += pkgStr
		}
		return pkgList
	} else if len(d.PackageDiff) > 0 { // Two images are passed in, compare based on Differences
		pkgDiff := ""
		for _, pd := range d.PackageDiff {
			pkgStr := ""
			if pd.typeOFDiff == "shared" { // Compare shared packages
				if len(pd.category) == 2 {
					pkgStr += "category:\n" + "< " + pd.category[0] + "\n" + "> " + pd.category[1] + "\n"
				}
				if len(pd.version) == 2 {
					pkgStr += "version:\n" + "< " + pd.version[0] + "\n" + "> " + pd.version[1] + "\n"
				}
				if len(pd.revision) == 2 {
					pkgStr += "revision:\n" + "< " + pd.revision[0] + "\n" + "> " + pd.revision[1] + "\n"
				}
				if pkgStr != "" && len(pd.name) == 2 {
					pkgStr = "Package " + pd.name[0] + " in " + image1 + " and " + image2 + " differ\n" + pkgStr + "\n"
					pkgDiff += pkgStr
				}
			} else { // Unique package, return all info
				if len(pd.name) == 1 && len(pd.category) == 1 && len(pd.version) == 1 && len(pd.revision) == 1 {
					if pd.typeOFDiff == "image1" {
						pkgStr += "Only in " + image1
					} else if pd.typeOFDiff == "image2" {
						pkgStr += "Only in " + image2
					}
					pkgStr += ": " + pd.name[0] + "\ncategory: " + pd.category[0] + "\nversion: " + pd.version[0] + "\nrevision: " + pd.revision[0] + "\n\n"
					pkgDiff += pkgStr
				}
			}
		}
		return pkgDiff
	}
	return ""
}

// Diff is a tool that finds all package differences of two COS images
// (Category, Name, Version, Revision)
// Input:
//   (*ImageInfo) image1 - A struct that will store package info for image1
//   (*ImageInfo) image2 - A struct that will store package info for image2
//   (*FlagInfo) flagInfo - A struct that holds input preference from the user
// Output:
//   (*Differences) packageDiff - A struct that will store the package differences
func Diff(packagesImage1, packagesImage2 []Package, flagInfo *input.FlagInfo) (*Differences, error) {
	packageDiff := &Differences{}

	if flagInfo.PackageSelected {
		if err := packageDiff.packageListDiff(packagesImage1, packagesImage2, flagInfo); err != nil {
			return packageDiff, fmt.Errorf("failed to take package list difference: %v", err)
		}
	}
	return packageDiff, nil
}
