package utilities

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

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

// Unmount umounts a mounted directory and deletes its loop device
func Unmount(mountedDirectory, loopDevice string) error {
	if _, err := exec.Command("sudo", "umount", mountedDirectory).Output(); err != nil {
		return fmt.Errorf("failed to umount directory %v: %v", mountedDirectory, err)
	}
	if _, err := exec.Command("sudo", "losetup", "-d", loopDevice).Output(); err != nil {
		return fmt.Errorf("failed to delete loop device %v: %v", loopDevice, err)
	}
	return nil
}
