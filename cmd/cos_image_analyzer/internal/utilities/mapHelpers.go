package utilities

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ReadFileToMap reads a text file line by line into a map. For each line
// key: first word split by separator, value: rest of line after separator.
// Ex: Inputs:  textLine: "NAME=Container-Optimized OS", sep: "="
//	   Outputs:  map: {"NAME":"Container-Optimized OS"}
// Input:
//   (string) filePath - The command-line path to the text file
//   (string) sep - The separator string for the key and value pairs
// Output:
//   (map[string]string) mapOfFile - The map of the read-in text file
func ReadFileToMap(filePath, sep string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return map[string]string{}, err
	}
	defer file.Close()

	mapOfFile := make(map[string]string)
	scanner := bufio.NewScanner(file) // Read file line by line to fill map
	for scanner.Scan() {
		key := strings.Split(string(scanner.Text()[:]), sep)[0]
		mapOfFile[key] = strings.Split(string(scanner.Text()[:]), sep)[1]
	}

	if scanner.Err() != nil {
		return map[string]string{}, err
	}
	return mapOfFile, nil
}

// CmpMapValues is a helper function that compares a value shared by two maps
// Input:
//   (map[string]string) map1 - First map to be compared
//   (map[string]string) map2 - Second map to be compared
//   (string) key - The key of the value be compared in both maps
// Output:
//   (stdout) terminal - If equal, print nothing. Else print difference
//   (int) result - -1 for error, 0 for no difference, 1 for difference
func CmpMapValues(map1, map2 map[string]string, key string) (int, error) {
	value1, ok1 := map1[key]
	value2, ok2 := map2[key]

	if !ok1 || !ok2 { // Error Check: At least one key is not present
		return -1, errors.New("Error:" + key + "key not found in at least one of the maps")
	}

	if value1 != value2 {
		fmt.Println(key, "Difference")
		fmt.Println("< ", value1)
		fmt.Println("> ", value2)
		return 1, nil
	}
	return 0, nil
}
