package binary

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// Global variables
var (
	// Command-line path strings
	// /etc is the OS configurations directory
	etc = "/etc/"

	// /etc/os-release is the file describing COS versioning
	etcOSRelease = "/etc/os-release"
)

// Differences is a Intermediate Struct used to store all binary differences
// Field names are pre-defined in parse_input.go and will be cross-checked with -binary flag.
type Differences struct {
	Version            []string
	BuildID            []string
	Rootfs             string
	KernelCommandLine  string
	Stateful           string
	PartitionStructure string
	SysctlSettings     string
	OSConfigs          map[string]string
	KernelConfigs      string
}

// versionDiff calculates the Version difference of two images
func (binaryDiff *Differences) versionDiff(image1, image2 *input.ImageInfo) {
	if image1.Version != image2.Version {
		binaryDiff.Version = []string{image1.Version, image2.Version}
	}
}

// buildDiff calculates the BuildID difference of two images
func (binaryDiff *Differences) buildDiff(image1, image2 *input.ImageInfo) {
	if image1.BuildID != image2.BuildID {
		binaryDiff.BuildID = []string{image1.BuildID, image2.BuildID}
	}
}

// rootfsDiff calculates the Root FS difference of two images
func (binaryDiff *Differences) rootfsDiff(image1, image2 *input.ImageInfo, flagInfo *input.FlagInfo) error {
	compressedRoots, err := ioutil.ReadFile(flagInfo.CompressRootfsFile)
	if err != nil {
		return fmt.Errorf("failed to convert file %v to slice: %v", flagInfo.CompressRootfsFile, err)
	}
	compressedRootsSlice := strings.Split(string(compressedRoots), "\n")
	rootfsDiff, err := directoryDiff(image1.RootfsPartition3, image2.RootfsPartition3, "rootfs", flagInfo.Verbose, compressedRootsSlice)
	if err != nil {
		return fmt.Errorf("fail to find rootfs difference: %v", err)
	}
	binaryDiff.Rootfs = rootfsDiff
	return nil
}

// osConfigDiff calculates the OsConfig difference of two images
func (binaryDiff *Differences) osConfigDiff(image1, image2 *input.ImageInfo, flagInfo *input.FlagInfo) error {
	if err := findOSConfigs(image1, image2, binaryDiff); err != nil {
		return fmt.Errorf("failed to find OS Configs: %v", err)
	}
	for etcEntryName, diff := range binaryDiff.OSConfigs {
		etcEntryPath := filepath.Join(etc, etcEntryName)
		if diff != "" {
			uniqueEntryPath := filepath.Join(diff+"/rootfs/", etcEntryPath)
			info, err := os.Stat(uniqueEntryPath)
			if err != nil {
				return fmt.Errorf("failed to get info on file %v: %v", uniqueEntryPath, err)
			}

			if info.IsDir() {
				binaryDiff.OSConfigs[etcEntryName] = diff + " has unique directory " + etcEntryPath
			} else {
				binaryDiff.OSConfigs[etcEntryName] = diff + " has unique file " + etcEntryPath
			}
		} else {
			compressedRoots, err := ioutil.ReadFile(flagInfo.CompressRootfsFile)
			if err != nil {
				return fmt.Errorf("failed to convert file %v to slice: %v", flagInfo.CompressRootfsFile, err)
			}
			compressedRootsSlice := strings.Split(string(compressedRoots), "\n")

			osConfigDiff, err := directoryDiff(filepath.Join(image1.RootfsPartition3, etcEntryPath), filepath.Join(image2.RootfsPartition3, etcEntryPath), "rootfs", flagInfo.Verbose, compressedRootsSlice)
			if err != nil {
				return fmt.Errorf("fail to find difference on /etc/ entry %v: %v", etcEntryName, err)
			}
			binaryDiff.OSConfigs[etcEntryName] = osConfigDiff
		}
	}
	return nil
}

// statefulDiff calculates the stateful partition difference of two images
func (binaryDiff *Differences) statefulDiff(image1, image2 *input.ImageInfo, flagInfo *input.FlagInfo) error {
	compressedStateful, err := ioutil.ReadFile(flagInfo.CompressStatefulFile)
	if err != nil {
		return fmt.Errorf("failed to convert file %v to slice: %v", flagInfo.CompressStatefulFile, err)
	}
	compressedStatefulSlice := strings.Split(string(compressedStateful), "\n")

	statefulDiff, err := directoryDiff(image1.StatePartition1, image2.StatePartition1, "stateful", flagInfo.Verbose, compressedStatefulSlice)
	if err != nil {
		return fmt.Errorf("failed to diff %v and %v: %v", image1.StatePartition1, image2.StatePartition1, err)
	}
	binaryDiff.Stateful = statefulDiff
	return nil
}

// partitionStructureDiff calculates the Version difference of two images
func (binaryDiff *Differences) partitionStructureDiff(image1, image2 *input.ImageInfo) error {
	if image2.TempDir != "" {
		partitionStructureDiff, err := pureDiff(image1.PartitionFile, image2.PartitionFile)
		if err != nil {
			return fmt.Errorf("fail to find Partition Structure difference: %v", err)
		}
		binaryDiff.PartitionStructure = partitionStructureDiff
	} else {
		image1Structure, err := ioutil.ReadFile(image1.PartitionFile)
		if err != nil {
			return fmt.Errorf("failed to convert file %v to string: %v", image1.PartitionFile, err)
		}
		binaryDiff.PartitionStructure = string(image1Structure)
	}
	return nil
}

// FormatVersionDiff returns a formated string of the version difference
func (binaryDiff *Differences) FormatVersionDiff() string {
	if len(binaryDiff.Version) == 2 {
		if binaryDiff.Version[1] != "" {
			return "-----Version-----\n< " + binaryDiff.Version[0] + "\n> " + binaryDiff.Version[1] + "\n\n"
		}
		return "-----Version-----\n" + binaryDiff.Version[0] + "\n\n"
	}
	return ""
}

// FormatBuildIDDiff returns a formated string of the build difference
func (binaryDiff *Differences) FormatBuildIDDiff() string {
	if len(binaryDiff.BuildID) == 2 {
		if binaryDiff.BuildID[1] != "" {
			return "-----BuildID-----\n< " + binaryDiff.BuildID[0] + "\n> " + binaryDiff.BuildID[1] + "\n\n"
		}
		return "-----BuildID-----\n" + binaryDiff.BuildID[0] + "\n\n"
	}
	return ""
}

// FormatRootfsDiff returns a formated string of the rootfs difference
func (binaryDiff *Differences) FormatRootfsDiff() string {
	if binaryDiff.Rootfs != "" {
		return "-----RootFS-----\n " + binaryDiff.Rootfs + "\n\n"
	}
	return ""
}

// FormatStatefulDiff returns a formated string of the stateful partition difference
func (binaryDiff *Differences) FormatStatefulDiff() string {
	if binaryDiff.Stateful != "" {
		return "-----Stateful Partition-----\n " + binaryDiff.Stateful + "\n\n"
	}
	return ""
}

// FormatOSConfigDiff returns a formated string of the OS Config difference
func (binaryDiff *Differences) FormatOSConfigDiff() string {
	if len(binaryDiff.OSConfigs) > 0 {
		osConfigDifference := "-----OS Configurations-----\n"
		for etcEntryName, diff := range binaryDiff.OSConfigs {
			if diff != "" {
				osConfigDifference += etcEntryName + "\n" + diff + "\n\n"
			}
		}
		return osConfigDifference
	}
	return ""
}

// FormatPartitionStructureDiff returns a formated string of the partition structure difference
func (binaryDiff *Differences) FormatPartitionStructureDiff() string {
	if binaryDiff.PartitionStructure != "" {
		return "-----Partition Structure-----\n " + binaryDiff.PartitionStructure + "\n\n"
	}
	return ""
}

// Diff is a tool that finds all binary differences of two COS images
// (COS version, rootfs, kernel command line, stateful partition, ...)
// Input:
//   (*ImageInfo) image1 - A struct that will store binary info for image1
//   (*ImageInfo) image2 - A struct that will store binary info for image2
//   (*FlagInfo) flagInfo - A struct that holds input preference from the user
// Output:
//   (*Differences) BinaryDiff - A struct that will store the binary differences
func Diff(image1, image2 *input.ImageInfo, flagInfo *input.FlagInfo) (*Differences, error) {
	BinaryDiff := &Differences{}

	if utilities.InArray("Version", flagInfo.BinaryTypesSelected) {
		BinaryDiff.versionDiff(image1, image2)
	}
	if utilities.InArray("BuildID", flagInfo.BinaryTypesSelected) {
		BinaryDiff.buildDiff(image1, image2)
	}
	if utilities.InArray("Partition-structure", flagInfo.BinaryTypesSelected) {
		if err := BinaryDiff.partitionStructureDiff(image1, image2); err != nil {
			return BinaryDiff, fmt.Errorf("Failed to get Partition-structure difference: %v", err)
		}
	}

	if image2.TempDir != "" {
		if utilities.InArray("Rootfs", flagInfo.BinaryTypesSelected) {
			if err := BinaryDiff.rootfsDiff(image1, image2, flagInfo); err != nil {
				return BinaryDiff, fmt.Errorf("Failed to get Roofs difference: %v", err)
			}
		}
		if utilities.InArray("OS-config", flagInfo.BinaryTypesSelected) {
			if err := BinaryDiff.osConfigDiff(image1, image2, flagInfo); err != nil {
				return BinaryDiff, fmt.Errorf("Failed to get OS-config difference: %v", err)
			}
		}
		if utilities.InArray("Stateful-partition", flagInfo.BinaryTypesSelected) {
			if err := BinaryDiff.statefulDiff(image1, image2, flagInfo); err != nil {
				return BinaryDiff, fmt.Errorf("Failed to get Stateful-partition difference: %v", err)
			}
		}
	}
	return BinaryDiff, nil
}
