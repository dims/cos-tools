package input

import (
	"bytes"
	"cloud.google.com/go/storage"
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const contextTimeOut = time.Second * 50

// gcsDowndload calls the GCS client api to download a specifed object from
// a GCS bucket. ADC is used for authorization
// Input:
//   (io.Writier) w - Output destination for download info
//   (string) bucket - Name of the GCS bucket
//   (string) object - Name of the GCS object
//   (string) destDir - Destination for downloaded GCS object
// Output:
//   (string) downloadedFile - Path to downloaded GCS object
func gcsDowndload(w io.Writer, bucket, object, destDir string) (string, error) {
	// Call API to download GCS object into tempDir
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, contextTimeOut)
	defer cancel()

	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return "", err
	}

	log.Print(log.New(w, "Blob "+object+" downloaded.\n", log.Ldate|log.Ltime|log.Lshortfile))

	downloadedFile := filepath.Join(destDir, object)
	if err := ioutil.WriteFile(downloadedFile, data, 0666); err != nil {
		return "", err
	}
	return downloadedFile, nil
}

// getPartitionStart finds the start partition offset of the disk
// Input:
//   (string) diskFile - Name of DOS/MBR file (ex: disk.raw)
//   (string) parition - The parition number you are pulling the offset from
// Output:
//   (int) start - The start of the partition on the disk
func getPartitionStart(partition, diskRaw string) (int, error) {
	//create command
	cmd1 := exec.Command("fdisk", "-l", diskRaw)
	cmd2 := exec.Command("grep", "disk.raw"+partition)

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
	start, err := strconv.Atoi(words[1])
	if err != nil {
		return -1, err
	}

	return start, nil
}

// mountDisk finds a free loop device and mounts a DOS/MBR disk file
// Input:
//   (string) diskFile - Name of DOS/MBR file (ex: disk.raw)
//   (string) mountDir - Mount Destiination
// Output: nil on success, else error
func mountDisk(diskFile, mountDir string, flag int) error {
	sectorSize := 512
	startOfPartition, err := getPartitionStart("3", diskFile)
	if err != nil {
		return err
	}
	offset := strconv.Itoa(sectorSize * startOfPartition)
	out, err := exec.Command("sudo", "losetup", "--show", "-fP", diskFile).Output()
	if err != nil {
		return err
	}
	_, err1 := exec.Command("sudo", "mount", "-o", "ro,loop,offset="+offset, string(out[:len(out)-1]), mountDir).Output()
	if err1 != nil {
		return err1
	}

	return nil
}

// GetGcsImage calls the GCS client api that downloads a specifed object from
// a GCS bucket and unzips its contents. ADC is used for authorization
// Input:
//   (string) gcsPath - GCS "bucket/object" path for COS Image (.tar.gz file)
// Output:
//   (string) imageDir - Path to the mounted directory of the  COS Image
func GetGcsImage(gcsPath string, flag int) (string, error) {
	bucket := strings.Split(gcsPath, "/")[0]
	object := strings.Split(gcsPath, "/")[1]

	tempDir, err := ioutil.TempDir(".", "tempDir-"+object) // Removed at end
	if err != nil {
		return "", err
	}

	tarFile, err := gcsDowndload(os.Stdout, bucket, object, tempDir)
	if err != nil {
		return "", err
	}

	imageDir := filepath.Join(tempDir, "Image-"+object)
	if err = os.Mkdir(imageDir, 0700); err != nil {
		return "", err
	}

	_, err1 := exec.Command("tar", "-xzf", tarFile, "-C", imageDir).Output()
	if err1 != nil {
		return "", err1
	}

	diskRaw := filepath.Join(imageDir, "disk.raw")
	if err = mountDisk(diskRaw, imageDir, flag); err != nil {
		return "", err
	}

	return imageDir, nil
}
