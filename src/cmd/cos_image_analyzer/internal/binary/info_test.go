package binary

import (
	"testing"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
)

// test ReadFileToMap function
func TestGetBinaryInfo(t *testing.T) {
	// test normal file
	testImage1 := &input.ImageInfo{RootfsPartition3: "../testdata/image1"}
	testImage2 := &input.ImageInfo{}
	testImage3 := &input.ImageInfo{RootfsPartition3: "../testdata/image2"}

	expectedImage1 := &input.ImageInfo{RootfsPartition3: "../testdata/image1", Version: "81", BuildID: "12871.119.0"}
	expectedImage2 := &input.ImageInfo{}
	expectedImage3 := &input.ImageInfo{RootfsPartition3: "../testdata/image2", Version: "77", BuildID: "12371.273.0"}

	type test struct {
		Image1 *input.ImageInfo
		want   *input.ImageInfo
	}

	tests := []test{
		{Image1: testImage1, want: expectedImage1},
		{Image1: testImage2, want: expectedImage2},
		{Image1: testImage3, want: expectedImage3},
	}

	for _, tc := range tests {
		GetBinaryInfo(tc.Image1)

		if tc.want.Version != tc.Image1.Version {
			t.Fatalf("Diff expected: %v, got: %v", tc.want.Version, tc.Image1.Version)
		}
		if tc.want.BuildID != tc.Image1.BuildID {
			t.Fatalf("Diff expected: %v, got: %v", tc.want.BuildID, tc.Image1.BuildID)
		}
	}

}
