package utilities

import (
	"testing"
)

// test ReadFileToMap function
func TestReadFileToMap(t *testing.T) {
	// test normal file
	testFile, sep := "../testData/os-release-77", "="
	expectedMap := map[string]string{"BUILD_ID": "12371.273.0", "ID": "cos"}
	resultMap := ReadFileToMap(testFile, sep)

	// Compare result with expected
	if resultMap["BUILD_ID"] != expectedMap["BUILD_ID"] && resultMap["ID"] != expectedMap["ID"] {
		t.Errorf("ReadFileToMap failed, expected %v, got %v", expectedMap, resultMap)
	}
}

// test ReadFileToMap function
func TestCmpMapValues(t *testing.T) {
	// test data
	testMap1 := map[string]string{"BUILD_ID": "12371.273.0", "VERSION": "77", "ID": "cos"}
	testMap2 := map[string]string{"BUILD_ID": "12871.119.0", "VERSION": "81", "ID": "cos"}
	testKey1, testKey2 := "ID", "VERSION"

	// test similar keys
	if result1 := CmpMapValues(testMap1, testMap2, testKey1); result1 != 0 { // Expect 0 for same values
		t.Errorf("CmpMapValues failed, expected %v, got %v", 0, result1)
	}

	// test different keys
	if result2 := CmpMapValues(testMap1, testMap2, testKey2); result2 != 1 { // Expect 1 for different values
		t.Errorf("CmpMapValues failed, expected %v, got %v", 1, result2)
	}
}
