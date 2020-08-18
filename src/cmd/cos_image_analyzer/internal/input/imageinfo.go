package input

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

const gcsObjFormat = ".tar.gz"
const makeDirFilemode = 0700
const timeOut = "7200s"
const imageFormat = "vmdk"
const name = "gcr.io/compute-image-tools/gce_vm_image_export:release"
const pathToKernelConfigs = "usr/src/linux-headers-4.19.112+/.config"
const pathToSysctlSettings = "/etc/sysctl.d/00-sysctl.conf" // Located in partition 3 Root-A

// ImageInfo stores all relevant information on a COS image
type ImageInfo struct {
	// Input Overhead
	TempDir          string // Temporary directory holding the mounted image and disk file
	DiskFile         string // Path to the DOS/MBR disk partition file
	StatePartition1  string // Path to mounted directory of partition #1, stateful partition
	RootfsPartition3 string // Path to mounted directory of partition #3, Rootfs-A
	EFIPartition12   string // Path to mounted directory of partition #12, EFI-System
	LoopDevice1      string // Active loop device for mounted image
	LoopDevice3      string // Active loop device for mounted image
	LoopDevice12     string // Active loop device for mounted image

	// Binary info
	Version            string // Major cos version
	BuildID            string // Minor cos version
	PartitionFile      string // Path to the file storing the disk partition structure from "sgdisk"
	SysctlSettingsFile string // Path to the /etc/sysctrl.d/00-sysctl.conf file of an image
	KernelCommandLine  string // The kernel command line boot-time parameters stored in partition 12 efi/boot/grub.cfg
	KernelConfigsFile  string // Path to the ".config" file downloaded from GCS that holds a build's kernel configs
}

// Rename temporary directory and its contents once Version and BuildID are known
func (image *ImageInfo) Rename(flagInfo *FlagInfo) error {
	if image.Version != "" && image.BuildID != "" {
		fullImageName := "cos-" + image.Version + "-" + image.BuildID
		if err := os.Rename(image.TempDir, fullImageName); err != nil {
			return fmt.Errorf("failed to rename directory %v to %v: %v", image.TempDir, fullImageName, err)
		}
		image.TempDir = fullImageName

		if !flagInfo.LocalPtr {
			image.DiskFile = filepath.Join(fullImageName, "disk.raw")
		}
		if image.StatePartition1 != "" {
			image.StatePartition1 = filepath.Join(fullImageName, "stateful")
		}
		if image.RootfsPartition3 != "" {
			image.RootfsPartition3 = filepath.Join(fullImageName, "rootfs")
		}
		if image.EFIPartition12 != "" {
			image.EFIPartition12 = filepath.Join(fullImageName, "efi")
		}
		image.PartitionFile = filepath.Join(fullImageName, "partitions.txt")
		image.KernelConfigsFile = filepath.Join(fullImageName, pathToKernelConfigs)
		image.SysctlSettingsFile = filepath.Join(image.RootfsPartition3, pathToSysctlSettings)
	}
	return nil
}

// MountImage is an ImagInfo method that mounts partitions 1,3 and 12 of
// the image into the temporary directory
// Input:
//   (string) arr - List of binary types selected from the user
// Output: nil on success, else error
func (image *ImageInfo) MountImage(arr []string) error {
	if image.TempDir == "" {
		return nil
	}
	if utilities.InArray("Stateful-partition", arr) {
		stateful := filepath.Join(image.TempDir, "stateful")
		if err := os.Mkdir(stateful, makeDirFilemode); err != nil {
			return fmt.Errorf("failed to create make directory %v: %v", stateful, err)
		}
		image.StatePartition1 = stateful

		loopDevice1, err := utilities.MountDisk(image.DiskFile, image.StatePartition1, "1")
		if err != nil {
			return fmt.Errorf("Failed to mount %v's partition #1 onto %v: %v", image.DiskFile, image.StatePartition1, err)
		}
		image.LoopDevice1 = loopDevice1
	}

	if utilities.InArray("Version", arr) || utilities.InArray("BuildID", arr) || utilities.InArray("Rootfs", arr) || utilities.InArray("Sysctl-settings", arr) || utilities.InArray("OS-config", arr) || utilities.InArray("Kernel-configs", arr) {
		rootfs := filepath.Join(image.TempDir, "rootfs")
		if err := os.Mkdir(rootfs, makeDirFilemode); err != nil {
			return fmt.Errorf("failed to create make directory %v: %v", rootfs, err)
		}
		image.RootfsPartition3 = rootfs

		loopDevice3, err := utilities.MountDisk(image.DiskFile, image.RootfsPartition3, "3")
		if err != nil {
			return fmt.Errorf("Failed to mount %v's partition #3 onto %v: %v", image.DiskFile, image.RootfsPartition3, err)
		}
		image.LoopDevice3 = loopDevice3
	}

	if utilities.InArray("Kernel-command-line", arr) {
		efi := filepath.Join(image.TempDir, "efi")
		if err := os.Mkdir(efi, makeDirFilemode); err != nil {
			return fmt.Errorf("failed to create make directory %v: %v", efi, err)
		}
		image.EFIPartition12 = efi

		loopDevice12, err := utilities.MountDisk(image.DiskFile, image.EFIPartition12, "12")
		if err != nil {
			return fmt.Errorf("Failed to mount %v's partition #12 onto %v: %v", image.DiskFile, image.EFIPartition12, err)
		}
		image.LoopDevice12 = loopDevice12
	}
	return nil
}

// GetGcsImage is an ImagInfo method that calls the GCS client api to
// download a COS image from a GCS bucket, unzips it, and mounts relevant
// partitions. ADC is used for authorization
// Input:
//	 (string) gcsPath - GCS "bucket/object" path for stored COS Image (.tar.gz file)
// Output: nil on success, else error
func (image *ImageInfo) GetGcsImage(gcsPath string) error {
	if gcsPath == "" {
		return nil
	}
	var gcsBucket, gcsObject string
	if startOfBucket := strings.Index(gcsPath, "gs://"); startOfBucket < len(gcsPath)-5 {
		gcsPath = gcsPath[startOfBucket+5:]
	} else {
		printUsage()
		return errors.New("Error: Argument " + gcsPath + " is not a valid gcs path \"gs://<bucket>/<object_path>.tar.gz\"")
	}
	if startOfObject := strings.Index(gcsPath, "/"); startOfObject > 0 && startOfObject < len(gcsPath)-1 {
		gcsBucket = gcsPath[:startOfObject]
		gcsObject = gcsPath[startOfObject+1:]
	} else {
		printUsage()
		return errors.New("Error: Argument " + gcsPath + " is not a valid gcs path \"gs://<bucket>/<object_path>.tar.gz\"")
	}

	tempDir, err := ioutil.TempDir(".", "tempDir") // Removed at end
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	image.TempDir = tempDir

	tarFile, err := utilities.GcsDowndload(gcsBucket, gcsObject, image.TempDir, filepath.Base(gcsObject))
	if err != nil {
		return fmt.Errorf("failed to download GCS object %v from bucket %v: %v", gcsObject, gcsBucket, err)
	}

	_, err = exec.Command("tar", "-xzf", tarFile, "-C", image.TempDir).Output()
	if err != nil {
		return fmt.Errorf("failed to unzip %v into %v: %v", tarFile, image.TempDir, err)
	}
	image.DiskFile = filepath.Join(image.TempDir, "disk.raw")
	return nil
}

// GetLocalImage is an ImageInfo method that creates a temporary directory
// to loop device mount the disk.raw file stored on the local file system
// Input:
//   (string) localPath - Local path to the disk.raw file
// Output: nil on success, else error
func (image *ImageInfo) GetLocalImage(localPath string) error {
	if localPath == "" {
		return nil
	}
	image.DiskFile = localPath

	tempDir, err := ioutil.TempDir(".", "tempDir") // Removed at end
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	image.TempDir = tempDir
	return nil
}

// steps holds GCE payload meta data
type steps struct {
	Args [5]string `json:"args"`
	Name string    `json:"name"`
	Env  [1]string `json:"env"`
}

// gcePayload holds GCE's rest API payload
type gcePayload struct {
	Timeout string    `json:"timeout"`
	Steps   [1]steps  `json:"steps"`
	Tags    [2]string `json:"tags"`
}

// gceExport calls the cloud build REST api that exports a public compute
// image to a specific GCS bucket.
// Input:
//   (string) projectID - project ID of the cloud project holding the image
//   (string) bucket - name of the GCS bucket holding the COS Image
//   (string) image - name of the source image to be exported
// Output: nil on success, else error
func gceExport(projectID, bucket, image string) error {
	// API Variables
	gceURL := "https://cloudbuild.googleapis.com/v1/projects/" + projectID + "/builds"
	destURI := "gs://" + bucket + "/" + image + "." + imageFormat
	args := [5]string{"-timeout=" + timeOut, "-source_image=" + image, "-client_id=api", "-format=" + imageFormat, "-destination_uri=" + destURI}
	env := [1]string{"BUILD_ID=$BUILD_ID"}
	tags := [2]string{"gce-daisy", "gce-daisy-image-export"}

	// Build API bodies
	steps := [1]steps{{Args: args, Name: name, Env: env}}
	payload := gcePayload{
		Timeout: timeOut,
		Steps:   steps,
		Tags:    tags}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to json marshal GCE payload: %v", err)
	}
	log.Println(string(requestBody))

	resp, err := http.Post(gceURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to make POST request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read returned POST request: %v", err)
	}

	log.Println(string(body))
	return nil
}

// GetCosImage calls the cloud build api to export a public COS image to a
// a GCS bucket and then calls GetGcsImage() to download that image from GCS.
// ADC is used for authorization.
// Input:
//   (*ImageInfo) image - A struct that holds the relevent
//	 CosCloudPath "bucket/image" and projectID for the stored COS Image
// Output: nil on success, else error
func (image *ImageInfo) GetCosImage(cosCloudPath, projectID string) error {
	if cosCloudPath == "" {
		return nil
	}
	cosArray := strings.Split(cosCloudPath, "/")
	if len(cosArray) != 2 {
		printUsage()
		return errors.New("Error: Argument " + cosCloudPath + " is not a valid cos-cloud path (\"/\" separators)")
	}
	gcsBucket := cosArray[0]
	publicCosImage := cosArray[1]
	if err := gceExport(projectID, gcsBucket, publicCosImage); err != nil {
		return fmt.Errorf("failed to export %v cos image to GCS bucket %v: %v", publicCosImage, gcsBucket, err)
	}

	gcsPath := filepath.Join(gcsBucket, publicCosImage, gcsObjFormat)
	if err := image.GetGcsImage(gcsPath); err != nil {
		return fmt.Errorf("failed to download image stored on GCS for %v: %v", gcsPath, err)
	}
	return nil
}

// Cleanup is a ImageInfo method that removes a mounted directory & loop device
// Input:
//   (*ImageInfo) image - A struct that holds the relevent info to clean up
// Output: nil on success, else error
func (image *ImageInfo) Cleanup() error {
	if image.TempDir == "" {
		return nil
	}
	if image.LoopDevice1 != "" {
		if err := utilities.Unmount(image.StatePartition1, image.LoopDevice1); err != nil {
			return fmt.Errorf("failed to unmount mount directory %v and/or loop device %v: %v", image.StatePartition1, image.LoopDevice1, err)
		}
	}
	if image.LoopDevice3 != "" {
		if err := utilities.Unmount(image.RootfsPartition3, image.LoopDevice3); err != nil {
			return fmt.Errorf("failed to unmount mount directory %v and/or loop device %v: %v", image.RootfsPartition3, image.LoopDevice3, err)
		}
	}
	if image.LoopDevice12 != "" {
		if err := utilities.Unmount(image.EFIPartition12, image.LoopDevice12); err != nil {
			return fmt.Errorf("failed to unmount mount directory %v and/or loop device %v: %v", image.EFIPartition12, image.LoopDevice12, err)
		}
	}

	if err := os.RemoveAll(image.TempDir); err != nil {
		return fmt.Errorf("failed to delete directory %v: %v", image.TempDir, err)
	}
	return nil
}
