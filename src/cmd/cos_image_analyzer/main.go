// cos_Image_Analyzer finds all the meaningful differences of two COS Images
// (binary, package, commit, and release notes differences)
//
// Input:  (string) rootImg1 - The path for COS image1
//		   (string) rootImg2 - The path for COS image2
//		   (int) inputFlag - 0-Local filesystem path to root directory,
//		   1-COS cloud names, 2-GCS object names
//
// Output: (stdout) terminal ouput - All differences printed to the terminal
package main

import (
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/binary"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"fmt"
	"os"
	"runtime"
)

func cosImageAnalyzer(img1Path, img2Path string) error {
	err := binary.BinaryDiff(img1Path, img2Path)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	if runtime.GOOS != "linux" {
		fmt.Printf("Error: This is a Linux tool, can not run on %s", runtime.GOOS)
		os.Exit(1)
	}
	rootImg1, rootImg2, err := input.ParseInput()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err1 := cosImageAnalyzer(rootImg1, rootImg2)
	if err1 != nil {
		fmt.Println(err1)
		os.Exit(1)
	}
	// Cleanup(rootImg1, loop1) Debating on a struct that holds this info
	// Cleanup(rootImg2, loop2)

}
