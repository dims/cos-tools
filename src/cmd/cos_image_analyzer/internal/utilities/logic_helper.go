package utilities

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const sectorSize = 512

// InArray determines if a string appears in a string array
func InArray(val string, arr []string) bool {
	for _, elem := range arr {
		if elem == val {
			return true
		}
	}
	return false
}

//EqualArrays determines if two string arrays are equal
func EqualArrays(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}
	for i, elem := range arr1 {
		if arr2[i] != elem {
			return false
		}
	}
	return true
}

// FileExists determines if the path exists, and then if
// the path points to a file of the desired type
// Input:
//   (string) path - Local path to the file
//   (string) desiredType - The type of the file desired
// Output: -1 if file doesn't exist, 0 if exists and is not
// desiredType, and 1 if file exists and is desiredType
func FileExists(path, desiredType string) int {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || info.IsDir() {
		return -1
	}
	fileName := strings.Split(info.Name(), ".")
	fileType := fileName[len(fileName)-1]
	if fileType != desiredType {
		return 0
	}
	return 1
}

// WriteToNewFile creates a file and writes a string into it
func WriteToNewFile(filename string, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		return err
	}
	return file.Sync()
}

// SliceToMapStr initializes a map with keys from input and empty strings as values
func SliceToMapStr(input []string) map[string]string {
	output := make(map[string]string)
	for _, elem := range input {
		output[elem] = ""
	}
	return output
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

// MountDisk finds a free loop device and mounts a DOS/MBR disk file
// Input:
//   (string) diskFile - Name of DOS/MBR file (ex: disk.raw)
//   (string) mountDir - Mount Destination
//   (string) partition - The partition number you are pulling the offset from
// Output:
//   (string) loopDevice - Name of the loop device used to mount
func MountDisk(diskFile, mountDir, partition string) (string, error) {
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

// Unmount umounts a mounted directory and deletes its loop device
func Unmount(mountedDirectory, loopDevice string) error {
	if _, err := exec.Command("sudo", "umount", "-l", mountedDirectory).Output(); err != nil {
		return fmt.Errorf("failed to umount directory %v: %v", mountedDirectory, err)
	}
	if _, err := exec.Command("sudo", "losetup", "-d", loopDevice).Output(); err != nil {
		return fmt.Errorf("failed to delete loop device %v: %v", loopDevice, err)
	}
	return nil
}
