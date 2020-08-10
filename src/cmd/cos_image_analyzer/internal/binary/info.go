package binary

import (
	"errors"
	"fmt"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// GetBinaryInfo finds relevant binary information for the COS image
// Input:
//   (*ImageInfo) image - A struct that will store binary info for the image
//   (localInput) bool - Flag to determine whether to rename disk.raw file
//   (*FlagInfo) flagInfo - A struct that holds input preference from the user
// Output: nil on success, else error
func GetBinaryInfo(image *input.ImageInfo, flagInfo *input.FlagInfo) error {
	if image.TempDir == "" {
		return nil
	}

	if image.RootfsPartition3 != "" {
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

	if utilities.InArray("Partition-structure", flagInfo.BinaryTypesSelected) {
		if err := image.GetPartitionStructure(); err != nil {
			return fmt.Errorf("failed to get partition structure for image %v: %v", image.TempDir, err)
		}
	}

	return nil
}
