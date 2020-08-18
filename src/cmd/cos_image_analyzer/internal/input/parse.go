package input

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

var binaryDiffTypes = []string{"Version", "BuildID", "Rootfs", "Kernel-command-line", "Stateful-partition", "Partition-structure", "Sysctl-settings", "OS-config"}

// Custom usage function. See -h flag
func printUsage() {
	usageTemplate := `NAME
	cos_image_analyzer - finds all meaningful differences of two COS Images (binary, package, commit, and release notes differences).
		If only one image is passed in, its binary info, package info, commit log, and release-notes will be returned.

SYNOPSIS
	%s [-local] FILE-1 [FILE-2] (default true)
		FILE - the local file path to the DOS/MBR boot sector file of your image (Ex: disk.raw)
		Ex: %s -local image-cos-77-12371-273-0/disk.raw image-cos-81-12871-119-0/disk.raw

	%s -local -binary=Sysctl-settings,OS-config -release-notes=false cos-77-12371/disk.raw

	%s -gcs GCS-PATH-1 [GCS-PATH-2]
		GCS-PATH - the GCS "bucket/object" path for the COS Image ("object" is type .tar.gz)
		Ex: %s -gcs my-bucket/cos-77-12371-273-0.tar.gz my-bucket/cos-81-12871-119-0.tar.gz


DESCRIPTION
	Input Flags:
	-local (default true, flag is optional)
		input is one or two DOS/MBR disk file on the local filesystem.
	-gcs
		input is one or two objects stored on Google Cloud Storage of type (.tar.gz). This flag temporarily downloads,
		unzips, and loop device mounts the images into this tool's directory.

	Difference Flags:
	-binary (string)
		specify which type of binary difference to show. Types "Version", "BuildID", "Rootfs", "Kernel-command-line",
		"Stateful-partition", "Partition-structure", "Sysctl-settings", and "OS-config" are supported. To list
		multiple types separate by comma. To NOT list any binary difference, set flag to "false". (default all types)
	-package
		specify whether to show package difference. Shows addition/removal of packages and package version updates.
		To NOT list any package difference, set flag to false. (default true)
	-commit
		specify whether to show commit difference. Shows commit changelog between the two images.
		To NOT list any commit difference, set flag to false. (default true)
	-release-notes
		specify whether to show release notes difference. Shows differences in human-written release notes between
		the two images. To NOT list any release notes difference, set flag to false. (default true)

	Attribute Flags
	-verbose
		include flag to increase verbosity of Rootfs, Stateful-partition, and OS-config differences. See the
		"internal/binary/CompressRootfs.txt", "internal/binary/CompressStateful.txt", and "internal/binary/CompressOSConfigs.txt"
		files to edit the default directories whose differences are compressed together.
	-compress-rootfs (string)
		to customize which directories are compressed in a non-verbose Rootfs and OS-config difference output, provide a local 
		file path to a text file. Format of the file must be one root file path per line with an ending back slash and no commas.
		By default the directory(s) that are compressed during a diff are /bin/, /lib/modules/, /lib64/, /usr/libexec/, /usr/bin/,
		/usr/sbin/, /usr/lib64/, /usr/share/zoneinfo/, /usr/share/git/, /usr/lib/, and /sbin/.
	-compress-stateful (string)
		to customize which directories are compressed in a non-verbose Stateful-partition difference output, provide a local
		file path to a text file. Format of file must be one root file path per line with no commas. By default the directory(s)
		that are compressed during a diff are /var_overlay/db/. 

	Output Flags:
	-output (string)
		Specify format of output. Only "terminal" stdout or "json" object is supported. (default "terminal")

OUTPUT
	Based on the "-output" flag. Either "terminal" stdout or machine readable "json" format.
`
	cmd := filepath.Base(os.Args[0])
	usage := fmt.Sprintf(usageTemplate, cmd, cmd, cmd, cmd, cmd)
	fmt.Printf("%s", usage)
}

// FlagErrorChecking validates command-line flags stored in the FlagInfo struct
// Input:
//   (*FlagInfo) flagInfo - A struct that stores all flag input
// Output: nil on success, else error
func FlagErrorChecking(flagInfo *FlagInfo) error {
	// Error Checking
	if (flagInfo.LocalPtr && flagInfo.GcsPtr) || (flagInfo.LocalPtr && flagInfo.CosCloudPtr) || (flagInfo.CosCloudPtr && flagInfo.GcsPtr) {
		return errors.New("Error: Only one input flag is allowed. Multiple appeared")
	}

	if !(flagInfo.GcsPtr) && !(flagInfo.CosCloudPtr) {
		flagInfo.LocalPtr = true
	}

	if flagInfo.BinaryDiffPtr == "" {
		flagInfo.BinaryTypesSelected = binaryDiffTypes
	} else {
		binaryTypesSelected := strings.Split(flagInfo.BinaryDiffPtr, ",")
		for _, elem := range binaryTypesSelected {
			if utilities.InArray(elem, binaryDiffTypes) {
				flagInfo.BinaryTypesSelected = append(flagInfo.BinaryTypesSelected, elem)
			} else {
				return errors.New("Error: Invalid option for \"-binary\" flag")
			}
		}
	}

	if res := utilities.FileExists(flagInfo.CompressRootfsFile, "txt"); res == -1 {
		return errors.New("Error: " + flagInfo.CompressRootfsFile + " file does not exist")
	} else if res == 0 {
		return errors.New("Error: " + flagInfo.CompressRootfsFile + " is not a \".txt\" file")
	}

	if res := utilities.FileExists(flagInfo.CompressStatefulFile, "txt"); res == -1 {
		return errors.New("Error: " + flagInfo.CompressStatefulFile + " file does not exist")
	} else if res == 0 {
		return errors.New("Error: " + flagInfo.CompressStatefulFile + " is not a \".txt\" file")
	}

	if flagInfo.OutputSelected != "terminal" && flagInfo.OutputSelected != "json" {
		return errors.New("Error: \"-output\" flag must be ethier \"terminal\" or \"json\"")
	}

	if len(flag.Args()) < 1 || len(flag.Args()) > 2 {
		return errors.New("Error: Input must be one or two arguments")
	}

	flagInfo.Image1 = flag.Arg(0)
	if len(flag.Args()) == 2 {
		if flag.Arg(0) == flag.Arg(1) {
			return errors.New("Error: Identical image passed in. To analyze single image, pass in one argument")
		}
		flagInfo.Image2 = flag.Arg(1)
	}

	return nil
}

// ParseFlags reads and validates the flags from the command-line
// Input: None (Command-line flags and args)
// Output:
//   (*FlagInfo) flagInfo - A struct that holds input preference from the user
func ParseFlags() (*FlagInfo, error) {
	flagInfo := &FlagInfo{}

	flag.Usage = printUsage
	flag.BoolVar(&flagInfo.LocalPtr, "local", false, "See printUsage for description")
	flag.BoolVar(&flagInfo.GcsPtr, "gcs", false, "")
	flag.BoolVar(&flagInfo.CosCloudPtr, "cos-cloud", false, "")

	flag.StringVar(&flagInfo.ProjectIDPtr, "projectID", "", "")

	flag.StringVar(&flagInfo.BinaryDiffPtr, "binary", "", "")
	flag.BoolVar(&flagInfo.PackageSelected, "package", true, "")
	flag.BoolVar(&flagInfo.CommitSelected, "commit", true, "")
	flag.BoolVar(&flagInfo.ReleaseNotesSelected, "release-notes", true, "")

	flag.BoolVar(&flagInfo.Verbose, "verbose", false, "")
	flag.StringVar(&flagInfo.CompressRootfsFile, "compress-rootfs", "internal/binary/CompressRootfs.txt", "")
	flag.StringVar(&flagInfo.CompressStatefulFile, "compress-stateful", "internal/binary/CompressStateful.txt", "")

	flag.StringVar(&flagInfo.OutputSelected, "output", "terminal", "")
	flag.Parse()

	if err := FlagErrorChecking(flagInfo); err != nil {
		printUsage()
		return &FlagInfo{}, err
	}
	return flagInfo, nil
}

// GetImages reads in all the flags and handles the input based on its type.
// Input:
//   (*FlagInfo) flagInfo - A struct that holds input preference from the user
// Output:
//   (*ImageInfo) image1 - A struct that stores relevent info for image1
//   (*ImageInfo) image2 - A struct that stores relevent info for image2
func GetImages(flagInfo *FlagInfo) (*ImageInfo, *ImageInfo, error) {
	image1, image2 := &ImageInfo{}, &ImageInfo{}

	// Input Selection
	if flagInfo.GcsPtr {
		gcsPath1, gcsPath2 := flagInfo.Image1, flagInfo.Image2

		if err := image1.GetGcsImage(gcsPath1); err != nil {
			return image1, image2, fmt.Errorf("failed to download image stored on GCS for %s: %v", gcsPath1, err)
		}
		if err := image2.GetGcsImage(gcsPath2); err != nil {
			return image1, image2, fmt.Errorf("failed to download image stored on GCS for %s: %v", gcsPath2, err)
		}
		return image1, image2, nil
	} else if flagInfo.CosCloudPtr {
		if flagInfo.ProjectIDPtr == "" {
			return image1, image2, errors.New("Error: COS-cloud input requires the \"projectID\" flag to be set")
		}
		cosCloudPath1, cosCloudPath2 := flagInfo.Image1, flagInfo.Image2

		if err := image1.GetCosImage(cosCloudPath1, flagInfo.ProjectIDPtr); err != nil {
			return image1, image2, fmt.Errorf("failed to get cos image for %s: %v", cosCloudPath1, err)
		}
		if err := image2.GetCosImage(cosCloudPath2, flagInfo.ProjectIDPtr); err != nil {
			return image1, image2, fmt.Errorf("failed to get cos image for %s: %v", cosCloudPath2, err)
		}
		return image1, image2, nil
	} else if flagInfo.LocalPtr {
		localPath1, localPath2 := flagInfo.Image1, flagInfo.Image2

		if err := ValidateLocalImages(localPath1, localPath2); err != nil {
			return image1, image2, fmt.Errorf("failed to validate local images: %v", err)
		}
		if err := image1.GetLocalImage(localPath1); err != nil {
			return image1, image2, fmt.Errorf("failed to get local image for %s: %v", localPath1, err)
		}
		if err := image2.GetLocalImage(localPath2); err != nil {
			return image1, image2, fmt.Errorf("failed to get local image for %s: %v", localPath2, err)
		}
		return image1, image2, nil
	}
	return image1, image2, errors.New("Error: At least one flag needs to be true")
}
