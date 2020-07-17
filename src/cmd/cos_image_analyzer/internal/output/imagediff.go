package output

import (
	"encoding/json"
	"fmt"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/binary"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// ImageDiff stores all of the differences between the two images
type ImageDiff struct {
	BinaryDiff *binary.Differences
}

// Formater is a ImageDiff function that outputs the image differences based on the "-output" flag.
// Either to the terminal (default) or to a stored json object
// Input:
//   (*FlagInfo) flagInfo - A struct that holds input preference from the user
// Output:
//   ([]string) diffstrings/jsonObjectStr - Based on "-output" flag, either formated string
//   for the terminal or a string json object
func (imageDiff *ImageDiff) Formater(flagInfo *input.FlagInfo) (string, error) {
	if flagInfo.OutputSelected == "terminal" {
		binaryStrings := ""
		binaryFunctions := map[string]func(*input.FlagInfo) string{
			"Version": imageDiff.BinaryDiff.FormatVersionDiff,
			"BuildID": imageDiff.BinaryDiff.FormatBuildIDDiff,
		}
		for diff := range binaryFunctions {
			if utilities.InArray(diff, flagInfo.BinaryTypesSelected) {
				binaryStrings += binaryFunctions[diff](flagInfo)
			}
		}

		if len(binaryStrings) > 0 {
			if flagInfo.Image2 == "" {
				binaryStrings = "================= Binary Info =================\n" + binaryStrings
			} else {
				binaryStrings = "================= Binary Differences =================\n" + binaryStrings
			}
		}

		diffStrings := binaryStrings
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
