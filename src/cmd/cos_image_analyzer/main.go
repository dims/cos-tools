// cos_Image_Analyzer finds all the meaningful differences of two COS Images
// (binary, package, commit, and release notes differences)
// Input:
//   (string) img1Path - The path for COS image1
//   (string) img2Path - The path for COS image2
//   (int) inputFlag - 0-Local filesystem path to root directory,
//   1-COS cloud names, 2-GCS object names
// Output:
//   (stdout) terminal ouput - All differences printed to the terminal
package main

import (
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/binary"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
	"flag"
	"log"
	"os"
	"runtime"
)

func cosImageAnalyzer(img1Path, img2Path string, inputFlag int) error {
	err := binary.BinaryDiff(img1Path, img2Path)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	if runtime.GOOS != "linux" {
		log.Fatalf("Error: This is a Linux tool, can not run on %s", runtime.GOOS)
	}
	// Flag Declartions
	flag.Usage = utilities.printUsage
	cloudPtr := flag.Bool("cloud", false, "input arguments are two cos-cloud images")
	gcsPtr := flag.Bool("gcs", false, "input arguments are two gcs objects")
	flag.Parse()
	if flag.NFlag() > 1 || len(flag.Args()) != 2 {
		log.Fatalf("Error: %s requires at most one flag and two arguments. Use -h flag for usage", os.Args[0])
	}

	inputFlag := 0
	if *cloudPtr {
		inputFlag = 1
	} else if *gcsPtr {
		inputFlag = 2
	}
	cosImageAnalyzer(flag.Args()[0], flag.Args()[1], inputFlag)
}
