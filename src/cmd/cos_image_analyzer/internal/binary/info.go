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
// Output: nil on success, else error
func GetBinaryInfo(image *input.ImageInfo) error {
	if image.RootfsPartition3 == "" {
		return nil
	}
	osReleaseMap, err := utilities.ReadFileToMap(image.RootfsPartition3+etcOSRelease, "=")
	if err != nil {
		return fmt.Errorf("Failed to read /etc/os-release file in rootfs of image: %v", err)
	}
	var ok bool
	if image.Version, ok = osReleaseMap["VERSION"]; !ok {
		return errors.New("Error: \"Version\" field not found in /etc/os-release file")
	}

	if image.BuildID, ok = osReleaseMap["BUILD_ID"]; !ok {
		return errors.New("Error: \"Build_ID\" field not found in /etc/os-release file")
	}
	return nil
}
