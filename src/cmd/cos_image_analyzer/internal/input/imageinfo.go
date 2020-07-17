package input

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

const sectorSize = 512
const gcsObjFormat = ".tar.gz"
const filemode = 0700
const timeOut = "7200s"
const imageFormat = "vmdk"
const name = "gcr.io/compute-image-tools/gce_vm_image_export:release"

// ImageInfo stores all relevant information on a COS image
type ImageInfo struct {
	// Input Overhead
	TempDir          string // Temporary directory holding the mounted image and disk file
	DiskFile         string // Path to the DOS/MBR disk partition file
	RootfsPartition3 string // Path to mounted directory of partition #3, Rootfs-A
	LoopDevice3      string // Active loop device for mounted image

	// Binary info
	Version string
	BuildID string

	// Package info
	// Commit info
	// Release notes info
}

// getPartitionStart finds the start partition offset of the disk
// Input:
//   (string) diskFile - Name of DOS/MBR file (ex: disk.raw)
//   (string) partition - The partition number you are pulling the offset from
// Output:
//   (int) start - The start of the partition on the disk
func getPartitionStart(partition, diskRaw string) (int, error) {
	//create command
	cmd1 := exec.Command("fdisk", "-l", diskRaw)
	cmd2 := exec.Command("grep", diskRaw+partition)

	reader, writer := io.Pipe()
	var buf bytes.Buffer

	cmd1.Stdout = writer
	cmd2.Stdin = reader
	cmd2.Stdout = &buf

	cmd1.Start()
	cmd2.Start()
	cmd1.Wait()
	writer.Close()
	cmd2.Wait()
	reader.Close()

	words := strings.Fields(buf.String())
	if len(words) < 2 {
		return -1, errors.New("Error: " + diskRaw + " is not a valid DOS/MBR boot sector file")
	}
	start, err := strconv.Atoi(words[1])
	if err != nil {
		return -1, fmt.Errorf("failed to convert Ascii %v to string: %v", words[1], err)
	}

	return start, nil
}

// mountDisk finds a free loop device and mounts a DOS/MBR disk file
// Input:
//   (string) diskFile - Name of DOS/MBR file (ex: disk.raw)
//   (string) mountDir - Mount Destination
//   (string) partition - The partition number you are pulling the offset from
// Output:
//   (string) loopDevice - Name of the loop device used to mount
func mountDisk(diskFile, mountDir, partition string) (string, error) {
	startOfPartition, err := getPartitionStart(partition, diskFile)
	if err != nil {
		return "", fmt.Errorf("failed to get start of partition #%v: %v", partition, err)
	}
	offset := strconv.Itoa(sectorSize * startOfPartition)

	out, err := exec.Command("sudo", "losetup", "--show", "-fP", diskFile).Output()
	if err != nil {
		return "", fmt.Errorf("failed to create new loop device for %v: %v", diskFile, err)
	}

	loopDevice := string(out[:len(out)-1])
	_, err = exec.Command("sudo", "mount", "-o", "ro,loop,offset="+offset, loopDevice, mountDir).Output()
	if err != nil {
		return "", fmt.Errorf("failed to mount loop device %v at %v: %v", loopDevice, mountDir, err)
	}
	return loopDevice, nil
}

// MountImage is an ImagInfo method that mounts partitions 1,3 and 12 of
// the image into the temporary directory
// Input: None
// Output: nil on success, else error
func (image *ImageInfo) MountImage() error {
	if image.TempDir == "" {
		return nil
	}

	rootfs := filepath.Join(image.TempDir, "rootFS")
	if err := os.Mkdir(rootfs, filemode); err != nil {
		return fmt.Errorf("failed to create make directory %v: %v", rootfs, err)
	}
	image.RootfsPartition3 = rootfs

	loopDevice3, err := mountDisk(image.DiskFile, image.RootfsPartition3, "3")
	if err != nil {
		return fmt.Errorf("Failed to mount %v's partition #3 onto %v: %v", image.DiskFile, image.RootfsPartition3, err)
	}
	image.LoopDevice3 = loopDevice3

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
	gcsArray := strings.Split(gcsPath, "/")
	if len(gcsArray) != 2 {
		printUsage()
		return errors.New("Error: Argument " + gcsPath + " is not a valid gcs path (\"/\" separators)")
	}
	gcsBucket := gcsArray[0]
	gcsObject := gcsArray[1]

	tempDir, err := ioutil.TempDir(".", "tempDir") // Removed at end
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	image.TempDir = tempDir

	tarFile, err := utilities.GcsDowndload(gcsBucket, gcsObject, image.TempDir)
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

// ValidateLocalImages ensures the two images are one or two unique boot files
// Input:
//   (string) localPath1 - Local path to the first disk.raw file
//   (string) localPath2 - Local path to the second disk.raw file
// Output: nil on success, else error
func ValidateLocalImages(localPath1, localPath2 string) error {
	if localPath2 == "" {
		if res := utilities.FileExists(localPath1, "raw"); res == -1 {
			return errors.New("Error: " + localPath1 + " file does not exist")
		} else if res == 0 {
			return errors.New("Error: " + localPath1 + " is not a \".raw\" file")
		}
		return nil
	}

	if res := utilities.FileExists(localPath2, "raw"); res == -1 {
		return errors.New("Error: " + localPath2 + " file does not exist")
	} else if res == 0 {
		return errors.New("Error: " + localPath2 + " is not a \".raw\" file")
	}

	info1, _ := os.Stat(localPath1)
	info2, _ := os.Stat(localPath2)
	if os.SameFile(info1, info2) {
		return errors.New("Error: Identical image passed in. To analyze single image, pass in one argument")
	}
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

	gcsPath := filepath.Join(gcsBucket, publicCosImage+gcsObjFormat)
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

	if image.LoopDevice3 != "" {
		if _, err := exec.Command("sudo", "umount", image.RootfsPartition3).Output(); err != nil {
			return fmt.Errorf("failed to umount directory %v: %v", image.RootfsPartition3, err)
		}
		if _, err := exec.Command("sudo", "losetup", "-d", image.LoopDevice3).Output(); err != nil {
			return fmt.Errorf("failed to delete loop device %v: %v", image.LoopDevice3, err)
		}
	}

	if err := os.RemoveAll(image.TempDir); err != nil {
		return fmt.Errorf("failed to delete directory %v: %v", image.TempDir, err)
	}
	return nil
}
