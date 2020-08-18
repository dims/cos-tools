package binary

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
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

// Differences is a intermediate Struct used to store all binary differences
// Field names are pre-defined in parse_input.go and will be cross-checked with -binary flag.
type Differences struct {
	Version            []string
	BuildID            []string
	Rootfs             string
	OSConfigs          map[string]string
	Stateful           string
	PartitionStructure string
	KernelConfigs      string
	KernelCommandLine  map[string]string
	SysctlSettings     string
}

// versionDiff calculates the Version difference of two images
func (d *Differences) versionDiff(image1, image2 *input.ImageInfo) {
	if image1.Version != image2.Version {
		d.Version = []string{image1.Version, image2.Version}
	}
}

// buildDiff calculates the BuildID difference of two images
func (d *Differences) buildDiff(image1, image2 *input.ImageInfo) {
	if image1.BuildID != image2.BuildID {
		d.BuildID = []string{image1.BuildID, image2.BuildID}
	}
}

// rootfsDiff calculates the Root FS difference of two images
func (d *Differences) rootfsDiff(image1, image2 *input.ImageInfo, flagInfo *input.FlagInfo) error {
	rootfsDiff, err := directoryDiff(image1.RootfsPartition3, image2.RootfsPartition3, "rootfs", flagInfo.Verbose, flagInfo.CompressRootfsSlice)
	if err != nil {
		return fmt.Errorf("fail to diff Rootfs partitions %v and %v: %v", image1.RootfsPartition3, image2.RootfsPartition3, err)
	}
	d.Rootfs = rootfsDiff
	return nil
}

// osConfigDiff calculates the OsConfig difference of two images
func (d *Differences) osConfigDiff(image1, image2 *input.ImageInfo, flagInfo *input.FlagInfo) error {
	mapOfEtcEntries, err := findOSConfigs(image1, image2) // Get map of /etc entries for both images
	if err != nil {
		return fmt.Errorf("failed to find OS Configs: %v", err)
	}
	output := make(map[string]string)
	for etcEntryName, img := range mapOfEtcEntries {
		etcEntryPath := filepath.Join(etc, etcEntryName) + "/"
		if flagInfo.Verbose || !utilities.InArray(etcEntryPath, flagInfo.CompressRootfsSlice) { // Only diff if Verbose or etcEntry is not in CompressRootfs.txt
			currentImage := img
			if img != "" { // Unique /etc entry in Image 1 or Image2
				output[etcEntryPath] += "Only in " + img + "/rootfs/etc: " + etcEntryName
			} else { // Shared /etc entry in Image 1 and Image 2
				osConfigDiff, err := pureDiff(filepath.Join(image1.RootfsPartition3, etcEntryPath), filepath.Join(image2.RootfsPartition3, etcEntryPath))
				if err != nil {
					return fmt.Errorf("fail to take \"diff -r --no-dereference\" on %v: %v", etcEntryPath, err)
				}
				currentImage = image1.TempDir
				output[etcEntryPath] = osConfigDiff
			}

			fullPath := filepath.Join(currentImage, "/rootfs/", etcEntryPath)
			entryFile, err := os.Stat(fullPath)
			if err != nil {
				return fmt.Errorf("failed to get info on file %v: %v", fullPath, err)
			}
			if output[etcEntryPath] != "" {
				if entryFile.IsDir() {
					output[etcEntryPath] = "Configs for directory " + etcEntryPath + "\n" + output[etcEntryPath]
				} else {
					output[etcEntryPath] = "Configs for file " + etcEntryPath + "\n" + output[etcEntryPath]
				}
			}
		}
	}
	d.OSConfigs = output
	return nil
}

// statefulDiff calculates the stateful partition difference of two images
func (d *Differences) statefulDiff(image1, image2 *input.ImageInfo, flagInfo *input.FlagInfo) error {
	statefulDiff, err := directoryDiff(image1.StatePartition1, image2.StatePartition1, "stateful", flagInfo.Verbose, flagInfo.CompressStatefulSlice)
	if err != nil {
		return fmt.Errorf("failed to diff stateful partitions %v and %v: %v", image1.StatePartition1, image2.StatePartition1, err)
	}
	d.Stateful = statefulDiff
	return nil
}

// partitionStructureDiff calculates the Version difference of two images
func (d *Differences) partitionStructureDiff(image1, image2 *input.ImageInfo) error {
	if image2.TempDir != "" {
		partitionStructureDiff, err := pureDiff(image1.PartitionFile, image2.PartitionFile)
		if err != nil {
			return fmt.Errorf("fail to compare both image's \"partitions.txt\" file: %v", err)
		}
		d.PartitionStructure = partitionStructureDiff
	} else {
		image1Structure, err := ioutil.ReadFile(image1.PartitionFile)
		if err != nil {
			return fmt.Errorf("failed to read partition file of image %v: %v", image1.TempDir, err)
		}
		d.PartitionStructure = string(image1Structure)
	}
	return nil
}

// kernelConfigsDiff calculates the kernel configs difference of two images
func (d *Differences) kernelConfigsDiff(image1, image2 *input.ImageInfo) error {
	if image2.TempDir != "" {
		kernelConfigsDiff, err := pureDiff(image1.KernelConfigsFile, image2.KernelConfigsFile)
		if err != nil {
			return fmt.Errorf("fail to compare the two image's kernel configs files: %v", err)
		}
		d.KernelConfigs = kernelConfigsDiff
	} else {
		image1KernelConfigs, err := ioutil.ReadFile(image1.KernelConfigsFile)
		if err != nil {
			return fmt.Errorf("failed to read kernel configs file of image %v: %v", image1.TempDir, err)
		}
		d.KernelConfigs = string(image1KernelConfigs)
	}
	return nil
}

// kernelCommandLineDiff calculates the kernel commad line difference of two images
func (d *Differences) kernelCommandLineDiff(image1, image2 *input.ImageInfo) error {
	output := make(map[string]string)
	if image2.TempDir != "" {
		mapImage1 := getKclMap(strings.Fields(image1.KernelCommandLine))
		mapImage2 := getKclMap(strings.Fields(image2.KernelCommandLine))

		for key1, value1 := range mapImage1 {
			if value2, ok := mapImage2[key1]; !ok { // Unique KCL parameter in image1
				if value1 != "" {
					output[key1] = "d\n" + "< " + key1 + "=" + value1
				} else {
					output[key1] = "d\n" + "< " + key1
				}
			} else if value2 != value1 { // Image1 and Image2 KCL parameter values differ
				output[key1] = "c\n" + "< " + key1 + "=" + value1 + "\n---\n> " + key1 + "=" + value2
			}
		}
		for key2, value2 := range mapImage2 {
			if _, ok := mapImage1[key2]; !ok { // Unique KCL parameter in image2
				if value2 != "" {
					output[key2] = "a\n" + "> " + key2 + "=" + value2
				} else {
					output[key2] = "a\n" + "> " + key2
				}
			}
		}
	} else {
		output["Image1 KCL"] = image1.KernelCommandLine
	}
	d.KernelCommandLine = output
	return nil
}

// sysctlSettingsDiff calculates the sysctl Settings difference of two images
func (d *Differences) sysctlSettingsDiff(image1, image2 *input.ImageInfo) error {
	if image2.TempDir != "" {
		sysctlSettingsDiff, err := pureDiff(image1.SysctlSettingsFile, image2.SysctlSettingsFile)
		if err != nil {
			return fmt.Errorf("fail to compare the two image's sysctl settings files: %v", err)
		}
		d.SysctlSettings = sysctlSettingsDiff
	} else {
		image1SysctlSettings, err := ioutil.ReadFile(image1.SysctlSettingsFile)
		if err != nil {
			return fmt.Errorf("failed to convert image 1's %v file to string: %v", image1.SysctlSettingsFile, err)
		}
		d.SysctlSettings = string(image1SysctlSettings)
	}
	return nil
}

// FormatVersionDiff returns a formated string of the version difference
func (d *Differences) FormatVersionDiff() string {
	if len(d.Version) == 2 {
		if d.Version[1] != "" {
			return "----------Version----------\n< " + d.Version[0] + "\n> " + d.Version[1] + "\n\n"
		}
		return "----------Version----------\n" + d.Version[0] + "\n\n"
	}
	return ""
}

// FormatBuildIDDiff returns a formated string of the build difference
func (d *Differences) FormatBuildIDDiff() string {
	if len(d.BuildID) == 2 {
		if d.BuildID[1] != "" {
			return "----------BuildID----------\n< " + d.BuildID[0] + "\n> " + d.BuildID[1] + "\n\n"
		}
		return "----------BuildID----------\n" + d.BuildID[0] + "\n\n"
	}
	return ""
}

// FormatRootfsDiff returns a formated string of the rootfs difference
func (d *Differences) FormatRootfsDiff() string {
	if d.Rootfs != "" {
		return "----------RootFS----------\n" + d.Rootfs + "\n\n"
	}
	return ""
}

// FormatStatefulDiff returns a formated string of the stateful partition difference
func (d *Differences) FormatStatefulDiff() string {
	if d.Stateful != "" {
		return "----------Stateful Partition----------\n" + d.Stateful + "\n\n"
	}
	return ""
}

// FormatOSConfigDiff returns a formated string of the OS Config difference
func (d *Differences) FormatOSConfigDiff() string {
	if len(d.OSConfigs) > 0 {
		osConfigDifference := "----------OS Configurations----------\n"
		keys := make([]string, 0)
		for k := range d.OSConfigs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if d.OSConfigs[k] != "" {
				osConfigDifference += d.OSConfigs[k] + "\n\n"
			}
		}
		return osConfigDifference
	}
	return ""
}

// FormatPartitionStructureDiff returns a formated string of the partition structure difference
func (d *Differences) FormatPartitionStructureDiff() string {
	if d.PartitionStructure != "" {
		return "----------Partition Structure----------\n" + d.PartitionStructure + "\n\n"
	}
	return ""
}

// FormatKernelConfigsDiff returns a formated string of the kernel configs difference
func (d *Differences) FormatKernelConfigsDiff() string {
	if d.KernelConfigs != "" {
		return "----------Kernel Configs----------\n" + d.KernelConfigs + "\n\n"
	}
	return ""
}

// FormatKernelCommandLineDiff returns a formated string of the KCL difference
func (d *Differences) FormatKernelCommandLineDiff() string {
	if len(d.KernelCommandLine) > 0 {
		kclDifference := "----------Kernel Command Line----------\n"
		keys := make([]string, 0)
		for k := range d.KernelCommandLine {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if d.KernelCommandLine[k] != "" {
				kclDifference += d.KernelCommandLine[k] + "\n\n"
			}
		}
		return kclDifference
	}
	return ""
}

// FormatSysctlSettingsDiff returns a formated string of the Sysctrl settings difference
func (d *Differences) FormatSysctlSettingsDiff() string {
	if d.SysctlSettings != "" {
		return "----------Sysctl settings----------\n" + d.SysctlSettings + "\n\n"
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
	if utilities.InArray("Kernel-configs", flagInfo.BinaryTypesSelected) {
		if err := BinaryDiff.kernelConfigsDiff(image1, image2); err != nil {
			return BinaryDiff, fmt.Errorf("failed to get Kernel-configs difference: %v", err)
		}
	}
	if utilities.InArray("Kernel-command-line", flagInfo.BinaryTypesSelected) {
		if err := BinaryDiff.kernelCommandLineDiff(image1, image2); err != nil {
			return BinaryDiff, fmt.Errorf("failed to get Kernel-command-line difference: %v", err)
		}
	}
	if utilities.InArray("Sysctl-settings", flagInfo.BinaryTypesSelected) {
		if err := BinaryDiff.sysctlSettingsDiff(image1, image2); err != nil {
			return BinaryDiff, fmt.Errorf("failed to get Sysctl-settings difference: %v", err)
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
