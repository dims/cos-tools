package binary

import (
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
)

// Global variables
var (
	// Command-line path strings
	// /etc/os-release is the file describing COS versioning
	etcOSRelease = "/etc/os-release"
)

// Differences is a Intermediate Struct used to store all binary differences
// Field names are pre-defined in parse_input.go and will be cross-checked with -binary flag.
type Differences struct {
	Version []string
	BuildID []string
}

// VersionDiff calculates the Version difference of two images
func (binaryDiff *Differences) VersionDiff(image1 *input.ImageInfo, image2 *input.ImageInfo) {
	if image1.Version != image2.Version {
		binaryDiff.Version = []string{image1.Version, image2.Version}
	}
}

// BuildDiff calculates the BuildID difference of two images
func (binaryDiff *Differences) BuildDiff(image1 *input.ImageInfo, image2 *input.ImageInfo) {
	if image1.BuildID != image2.BuildID {
		binaryDiff.BuildID = []string{image1.BuildID, image2.BuildID}
	}
}

// FormatVersionDiff returns a formated string of the version difference
func (binaryDiff *Differences) FormatVersionDiff(flagInfo *input.FlagInfo) string {
	if len(binaryDiff.Version) == 2 {
		return "Version\n< " + binaryDiff.Version[0] + "\n> " + binaryDiff.Version[1] + "\n"
	}
	return ""
}

// FormatBuildIDDiff returns a formated string of the build difference
func (binaryDiff *Differences) FormatBuildIDDiff(flagInfo *input.FlagInfo) string {
	if len(binaryDiff.BuildID) == 2 {
		return "BuildID\n< " + binaryDiff.BuildID[0] + "\n> " + binaryDiff.BuildID[1] + "\n"
	}
	return ""
}

// Diff is a tool that finds all binary differences of two COS images
// (COS version, rootfs, kernel command line, stateful partition, ...)
// Input:
//   (*ImageInfo) image1 - A struct that will store binary info for image1
//   (*ImageInfo) image2 - A struct that will store binary info for image2
// Output:
//   (*Differences) BinaryDiff - A struct that will store the binary differences
func Diff(image1, image2 *input.ImageInfo) (*Differences, error) {
	BinaryDiff := &Differences{}
	BinaryDiff.VersionDiff(image1, image2)
	BinaryDiff.BuildDiff(image1, image2)

	return BinaryDiff, nil
}
