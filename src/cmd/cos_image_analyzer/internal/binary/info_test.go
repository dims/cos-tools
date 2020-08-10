package binary

import (
	"testing"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
)

// test GetBinaryInf function
func TestGetBinaryInfo(t *testing.T) {
	for _, tc := range []struct {
		image    *input.ImageInfo
		flagInfo *input.FlagInfo
		want     *input.ImageInfo
	}{
		{image: &input.ImageInfo{TempDir: "../testdata/image1/", RootfsPartition3: "../testdata/image1/rootfs/"},
			flagInfo: &input.FlagInfo{LocalPtr: true},
			want:     &input.ImageInfo{RootfsPartition3: "../testdata/image1", Version: "81", BuildID: "12871.119.0"}},

		{image: &input.ImageInfo{},
			flagInfo: &input.FlagInfo{LocalPtr: true},
			want:     &input.ImageInfo{}},

		{image: &input.ImageInfo{TempDir: "../testdata/image2/", RootfsPartition3: "../testdata/image2/rootfs/"},
			flagInfo: &input.FlagInfo{LocalPtr: true},
			want:     &input.ImageInfo{RootfsPartition3: "../testdata/image2/rootfs", Version: "77", BuildID: "12371.273.0"}},
	} {
		GetBinaryInfo(tc.image, tc.flagInfo)

		if tc.want.Version != tc.image.Version {
			t.Fatalf("GetBinaryInfo expected: %v, got: %v", tc.want.Version, tc.image.Version)
		}
		if tc.want.BuildID != tc.image.BuildID {
			t.Fatalf("GetBinaryInfo expected: %v, got: %v", tc.want.BuildID, tc.image.BuildID)
		}
	}
}
