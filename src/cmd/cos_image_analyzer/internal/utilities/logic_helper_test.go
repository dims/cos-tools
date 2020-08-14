package utilities

import (
	"testing"
)

// test TestInArray function
func TestInArray(t *testing.T) {
	type test struct {
		testString string
		testSlice  []string
		want       bool
	}

	tests := []test{
		{testString: "77", testSlice: []string{"77", "81"}, want: true},
		{testString: "86", testSlice: []string{"77", "81"}, want: false},
		{testString: "77", testSlice: []string{""}, want: false},
	}

	for _, tc := range tests {
		got := InArray(tc.testString, tc.testSlice)
		if tc.want != got {
			t.Fatalf("InArray(%v, %v) call expected: %v, got: %v", tc.testString, tc.testSlice, tc.want, got)
		}
	}

}

// test TestEqualArrays function
func TestEqualArrays(t *testing.T) {
	type test struct {
		testSlice1 []string
		testSlice2 []string
		want       bool
	}

	tests := []test{
		{testSlice1: []string{"77", "81"}, testSlice2: []string{"77", "81"}, want: true},
		{testSlice1: []string{"77"}, testSlice2: []string{"77", "81"}, want: false},
		{testSlice1: []string{}, testSlice2: []string{""}, want: false},
		{testSlice1: []string{}, testSlice2: []string{}, want: true},
	}

	for _, tc := range tests {
		got := EqualArrays(tc.testSlice1, tc.testSlice2)
		if tc.want != got {
			t.Fatalf("EqualArray(%v, %v) call expected: %v, got: %v", tc.testSlice1, tc.testSlice2, tc.want, got)
		}
	}
}

// test FileExists function
func TestFileExists(t *testing.T) {
	type test struct {
		testFile        string
		testDesiredType string
		want            int
	}

	tests := []test{
		{testFile: "../testdata/DOESNOTEXIST.txt", testDesiredType: "txt", want: -1},
		{testFile: "../testdata/blank.txt", testDesiredType: "raw", want: 0},
		{testFile: "../testdata/blank.txt", testDesiredType: "txt", want: 1},
	}

	for _, tc := range tests {
		got := FileExists(tc.testFile, tc.testDesiredType)
		if tc.want != got {
			t.Fatalf("FileExits(%v, %v) call expected: %v, got: %v", tc.testFile, tc.testDesiredType, tc.want, got)
		}
	}
}

// test SliceToMapStr function
func TestSliceToMapStr(t *testing.T) {
	type test struct {
		input []string
		want  map[string]string
	}

	tests := []test{
		{input: []string{"a", "b", "c", "d"}, want: map[string]string{"a": "", "b": "", "c": "", "d": ""}},
		{input: []string{}, want: map[string]string{}},
	}

	for _, tc := range tests {
		got := SliceToMapStr(tc.input)
		for k, v := range tc.want {
			if v != got[k] {
				t.Fatalf("SliceToMapStr call expected: %v, got: %v", tc.want, got)
			}
		}
	}
}
