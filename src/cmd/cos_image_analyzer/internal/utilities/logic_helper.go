package utilities

import (
	"errors"
	"io"
	"os"
	"path/filepath"
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

// CopyFile copies a file over to a new destinaton
// Input:
//   (string) path - Local path to the file
//   (string) dest - Destination to copy the file
// Output:
//   (string) copiedFile - path to the newly copied file
func CopyFile(path, dest string) (string, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || info.IsDir() {
		return "", errors.New("Error: " + path + " is not a file")
	}

	sourceFile, err := os.Open(path)
	if err != nil {
		return "", errors.New("Error: failed to open file " + path)
	}
	defer sourceFile.Close()

	// Create new file
	copiedFile := filepath.Join(dest, info.Name())
	newFile, err := os.Create(copiedFile)
	if err != nil {
		return "", errors.New("Error: failed to create file " + copiedFile)
	}
	defer newFile.Close()

	if _, err := io.Copy(newFile, sourceFile); err != nil {
		return "", errors.New("Error: failed to copy " + path + " into " + copiedFile)
	}
	return copiedFile, nil
}
