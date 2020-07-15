package input

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

// Custom usage function. See -h flag
func printUsage() {
	usageTemplate := `NAME
cos_image_analyzer - finds all meaningful differences of two COS Images
(binary, package, commit, and release notes differences)

SYNOPSIS 
%s [-local] DIRECTORY-1 DIRECTORY-2 (default true)
	DIRECTORY 1/2 - the local directory path to the root of the COS Image

%s [-gcs] GCS-PATH-1 GCS-PATH-2 
	GCS-PATH 1/2 - GCS "bucket/object" path for the COS Image (.tar.gz file) 
	Ex: %s -gcs my-bucket/cos-77-12371-273-0.tar.gz my-bucket/cos-81-12871-119-0.tar.gz

%s [-cos-cloud]  COS-CLOUD-PATH-1 COS-CLOUD-PATH-2 
	COS-CLOUD-PATH 1/2 - The "projectID/gcs-bucket/image" path of the source image to be exported
	Ex: %s -cos-cloud my-project/my-bucket/my-exported-image1 my-project/my-bucket/my-exported-image2

DESCRIPTION
`
	usage := fmt.Sprintf(usageTemplate, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
	fmt.Printf("%s", usage)
	flag.PrintDefaults()
	fmt.Println("\nOUTPUT\n(stdout) terminal output - All differences printed to the terminal")
}

// ParseInput handles the input based on its type and returns the root
// directory path of both images to the start of the CosImageAnalyzer
//
// Input:  None (reads command-line args)
//
// Output: (string) rootImg1 - The local filesystem path for COS image1
//		   (string) rootImg2 - The local filesystem path for COS image2
func ParseInput() (string, string, error) {
	// Flag Declaration
	flag.Usage = printUsage
	localPtr := flag.Bool("local", true, "input is two mounted images on local filesystem")
	gcsPtr := flag.Bool("gcs", false, "input is two objects stored on Google Cloud Storage")
	cosCloudPtr := flag.Bool("cos-cloud", false, "input is two public COS-cloud images")
	flag.Parse()

	if flag.NFlag() > 1 {
		printUsage()
		return "", "", errors.New("Error: Only one flag allowed")
	}

	// Input Selection
	if *gcsPtr {
		if len(flag.Args()) != 2 {
			printUsage()
			return "", "", errors.New("Error: GCS input requires two agruments")
		}
		rootImg1, err := GetGcsImage(flag.Args()[0], 1)
		if err != nil {
			return "", "", err
		}
		rootImg2, err := GetGcsImage(flag.Args()[1], 2)
		if err != nil {
			return "", "", err
		}
		return rootImg1, rootImg2, nil
	} else if *cosCloudPtr {
		if len(flag.Args()) != 2 {
			printUsage()
			return "", "", errors.New("Error: COS-cloud input requires two agruments")
		}
		rootImg1, err := GetCosImage(flag.Args()[0])
		if err != nil {
			return "", "", err
		}
		rootImg2, err := GetCosImage(flag.Args()[1])
		if err != nil {
			return "", "", err
		}
		return rootImg1, rootImg2, nil
	} else if *localPtr {
		if len(flag.Args()) != 2 {
			printUsage()
			return "", "", errors.New("Error: Local input requires two arguments")
		}
		return flag.Args()[0], flag.Args()[1], nil
	}
	printUsage()
	return "", "", errors.New("Error: At least one flag needs to be true")
}
