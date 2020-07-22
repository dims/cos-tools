package binary

import (
	"fmt"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// Global variables
var (
	// Command-line path strings
	// /etc/os-release is the file describing COS versioning
	etcOSRelease = "/etc/os-release"
)

// BinaryDiff is a tool that finds all binary differneces of two COS images
// (COS version, rootfs, kernel command line, stateful parition, ...)
//
// Input:  (string) img1Path - The path to the root directory for COS image1
//		   (string) img2Path - The path to the root directory for COS image2
//
// Output: (stdout) terminal ouput - All differences printed to the terminal
func BinaryDiff(img1Path, img2Path string) error {
	fmt.Println("================== Binary Differences ==================")

	// COS Verison Difference
	fmt.Println("--------- COS Verison Difference ---------")
	verMap1, err := utilities.ReadFileToMap(img1Path+etcOSRelease, "=")
	if err != nil {
		return err
	}
	verMap2, err := utilities.ReadFileToMap(img2Path+etcOSRelease, "=")
	if err != nil {
		return err
	}

	// Compare Version (Major)
	_, err = utilities.CmpMapValues(verMap1, verMap2, "VERSION")
	if err != nil {
		return err
	}
	// Compare BUILD_ID (Minor)
	_, err = utilities.CmpMapValues(verMap1, verMap2, "BUILD_ID")
	if err != nil {
		return err
	}

	return nil
}
