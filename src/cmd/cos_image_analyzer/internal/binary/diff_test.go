package binary

import (
	"testing"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// test ReadFileToMap function
func TestDiff(t *testing.T) {
	for _, tc := range []struct {
		Image1 *input.ImageInfo
		Image2 *input.ImageInfo
		want   *Differences
	}{
		{Image1: &input.ImageInfo{RootfsPartition3: "../testdata/image1", Version: "81", BuildID: "12871.119.0"},
			Image2: &input.ImageInfo{},
			want:   &Differences{Version: []string{"81", ""}, BuildID: []string{"12871.119.0", ""}}},

		{Image1: &input.ImageInfo{RootfsPartition3: "../testdata/image1", Version: "81", BuildID: "12871.119.0"},
			Image2: &input.ImageInfo{RootfsPartition3: "../testdata/image2", Version: "77", BuildID: "12371.273.0"},
			want:   &Differences{Version: []string{"81", "77"}, BuildID: []string{"12871.119.0", "12371.273.0"}}},

		{Image1: &input.ImageInfo{RootfsPartition3: "../testdata/image1", Version: "81", BuildID: "12871.119.0"},
			Image2: &input.ImageInfo{RootfsPartition3: "../testdata/image1", Version: "81", BuildID: "12871.119.1"},
			want:   &Differences{Version: []string{}, BuildID: []string{"12871.119.0", "12871.119.1"}}},

		{Image1: &input.ImageInfo{RootfsPartition3: "../testdata/image1", Version: "81", BuildID: "12871.119.0"},
			Image2: &input.ImageInfo{RootfsPartition3: "../testdata/image1", Version: "81", BuildID: "12871.119.0"},
			want:   &Differences{}},
	} {
		got, _ := Diff(tc.Image1, tc.Image2)

		if !utilities.EqualArrays(tc.want.Version, got.Version) {
			t.Fatalf("Diff expected: %v, got: %v", tc.want.Version, got.Version)
		}
		if !utilities.EqualArrays(tc.want.BuildID, got.BuildID) {
			t.Fatalf("Diff expected: %v, got: %v", tc.want.BuildID, got.BuildID)
		}
	}
}
