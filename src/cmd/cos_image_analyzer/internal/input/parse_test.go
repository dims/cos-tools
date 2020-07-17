package input

import (
	"testing"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// test FlagErrorChecking function
func TestFlagErrorChecking(t *testing.T) {
	for _, tc := range []struct {
		input   *FlagInfo
		want    *FlagInfo
		wantErr bool
	}{
		{input: &FlagInfo{Image1: "../testdata/DOESNOTEXIST.txt", Image2: ""},
			want:    &FlagInfo{},
			wantErr: true},
		{input: &FlagInfo{Image1: "../testdata/false.raw", Image2: ""},
			want:    &FlagInfo{},
			wantErr: true},
		{input: &FlagInfo{Image1: "", Image2: ""},
			want:    &FlagInfo{},
			wantErr: true},
		{input: &FlagInfo{Image1: "arg0", Image2: "", LocalPtr: true, BinaryTypesSelected: []string{"wrongType"}},
			want:    &FlagInfo{},
			wantErr: true},
		{input: &FlagInfo{Image1: "arg0", Image2: "", LocalPtr: true, GcsPtr: true, CosCloudPtr: false, BinaryTypesSelected: []string{"BuildID"}},
			want:    &FlagInfo{},
			wantErr: true},
		{input: &FlagInfo{Image1: "arg0", Image2: "", LocalPtr: true, GcsPtr: false, CosCloudPtr: false, OutputSelected: "notJsonOrTerminal"},
			want:    &FlagInfo{},
			wantErr: true},
		{input: &FlagInfo{Image1: "arg0", Image2: "", LocalPtr: false, GcsPtr: false, CosCloudPtr: false, OutputSelected: "notJsonOrTerminal", BinaryTypesSelected: []string{"BuildID"}},
			want:    &FlagInfo{Image1: "arg0", Image2: "", LocalPtr: true, GcsPtr: false, CosCloudPtr: false, OutputSelected: "notJsonOrTerminal", BinaryTypesSelected: []string{"BuildID"}},
			wantErr: false},
	} {
		gotErr := FlagErrorChecking(tc.input)
		if tc.wantErr && gotErr == nil {
			t.Fatalf("expected error but none returned")
		}
		if gotErr != nil {
			continue
		}

		if tc.input.LocalPtr != tc.want.LocalPtr {
			t.Fatalf("expected: %v, got: %v", tc.want.LocalPtr, tc.input.LocalPtr)
		}
		if tc.input.GcsPtr != tc.want.GcsPtr {
			t.Fatalf("expected: %v, got: %v", tc.want.GcsPtr, tc.input.GcsPtr)
		}
		if tc.input.CosCloudPtr != tc.want.CosCloudPtr {
			t.Fatalf("expected: %v, got: %v", tc.want.CosCloudPtr, tc.input.CosCloudPtr)
		}
		if !utilities.EqualArrays(tc.input.BinaryTypesSelected, tc.want.BinaryTypesSelected) {
			t.Fatalf("expected: %v, got: %v", tc.want.BinaryTypesSelected, tc.input.BinaryTypesSelected)
		}
	}
}
