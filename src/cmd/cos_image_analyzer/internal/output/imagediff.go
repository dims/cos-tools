package output

import (
	"encoding/json"
	"fmt"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/binary"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/packagediff"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// ImageDiff stores all of the differences between the two images
type ImageDiff struct {
	BinaryDiff  *binary.Differences
	PackageDiff *packagediff.Differences
}

// Formater is a ImageDiff function that outputs the image differences based on the "-output" flag.
// Either to the terminal (default) or to a stored json object
// Input:
//   (string) image1 - Temp directory name of image1
//   (string) image2 - Temp directory name of image2
//   (*FlagInfo) flagInfo - A struct that holds input preference from the user
// Output:
//   ([]string) diffstrings/jsonObjectStr - Based on "-output" flag, either formated string
//   for the terminal or a string json object
func (imageDiff *ImageDiff) Formater(image1, image2 string, flagInfo *input.FlagInfo) (string, error) {
	if flagInfo.OutputSelected == "terminal" {
		binaryStrings := ""
		binaryFunctions := map[string]func() string{
			"Version":             imageDiff.BinaryDiff.FormatVersionDiff,
			"BuildID":             imageDiff.BinaryDiff.FormatBuildIDDiff,
			"Rootfs":              imageDiff.BinaryDiff.FormatRootfsDiff,
			"Stateful-partition":  imageDiff.BinaryDiff.FormatStatefulDiff,
			"OS-config":           imageDiff.BinaryDiff.FormatOSConfigDiff,
			"Partition-structure": imageDiff.BinaryDiff.FormatPartitionStructureDiff,
			"Kernel-configs":      imageDiff.BinaryDiff.FormatKernelConfigsDiff,
			"Kernel-command-line": imageDiff.BinaryDiff.FormatKernelCommandLineDiff,
			"Sysctl-settings":     imageDiff.BinaryDiff.FormatSysctlSettingsDiff,
		}
		for _, diff := range input.BinaryDiffTypes {
			if utilities.InArray(diff, flagInfo.BinaryTypesSelected) {
				binaryStrings += binaryFunctions[diff]()
			}
		}

		if len(binaryStrings) > 0 {
			if flagInfo.Image2 == "" {
				binaryStrings = "================= Binary Info =================\nImage: " + image1 + "\n" + binaryStrings
			} else {
				binaryStrings = "================= Binary Differences =================\nImages: " + image1 + " and " + image2 + "\n" + binaryStrings
			}
		}

		packageStrings := imageDiff.PackageDiff.FormatPackageListDiff(image1, image2)
		if len(packageStrings) > 0 {
			if flagInfo.Image2 == "" {
				packageStrings = "================= Package List =================\nImage: " + image1 + "\n" + packageStrings
			} else {
				packageStrings = "================= Package Differences =================\nImages: " + image1 + " and " + image2 + "\n" + packageStrings
			}
		}

		diffStrings := binaryStrings + packageStrings
		return diffStrings, nil
	}
	jsonObjectBytes, err := json.Marshal(imageDiff)
	if err != nil {
		return "", fmt.Errorf("failed to json marshal the image difference struct: %v", err)
	}
	jsonObjectStr := string(jsonObjectBytes[:])
	return jsonObjectStr, nil
}

// Print is a ImageDiff method that prints out all image differences
func (imageDiff *ImageDiff) Print(differences string) {
	fmt.Print(differences)
}
