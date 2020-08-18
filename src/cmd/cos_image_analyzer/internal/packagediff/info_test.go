package packagediff

import (
	"testing"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
)

// test GetPackageInfo function
func TestGetPackageInfo(t *testing.T) {
	testPackageList1 := []Package{
		{Category: "sys-boot", Name: "shim", Version: "14.0.20180308", Revision: "4"},
		{Category: "chromeos-launch", Name: "cloud-network-boot", Version: "1.0.0", Revision: "4"},
		{Category: "sys-kernel", Name: "lakitu-kernel-4_19", Version: "4.20.127", Revision: "533"},
		{Category: "sys-apps", Name: "findutils", Version: "4.9.10", Revision: "1"},
		{Category: "app-emulation", Name: "runc", Version: "1.0.0_rc10", Revision: "1"}}
	testPackageList2 := []Package{}
	testPackageList3 := []Package{
		{Category: "sys-boot", Name: "shim", Version: "14.0.20180308", Revision: "4"},
		{Category: "chromeos-base", Name: "cloud-network-init", Version: "1.0.0", Revision: "4"},
		{Category: "sys-kernel", Name: "lakitu-kernel-4_19", Version: "4.19.127", Revision: "535"},
		{Category: "app-shells", Name: "dash", Version: "0.5.9.1", Revision: "7"}}
	for _, tc := range []struct {
		image    *input.ImageInfo
		flagInfo *input.FlagInfo
		want     []Package
	}{ // Test image1
		{image: &input.ImageInfo{TempDir: "../testdata/image1/", RootfsPartition3: "../testdata/image1/rootfs/"},
			flagInfo: &input.FlagInfo{PackageSelected: true},
			want:     testPackageList1},
		// Test empty image
		{image: &input.ImageInfo{},
			flagInfo: &input.FlagInfo{PackageSelected: true},
			want:     testPackageList2},
		// Test image2
		{image: &input.ImageInfo{TempDir: "../testdata/image2/", RootfsPartition3: "../testdata/image2/rootfs/"},
			flagInfo: &input.FlagInfo{PackageSelected: true},
			want:     testPackageList3},
	} {
		got, _ := GetPackageInfo(tc.image, tc.flagInfo)

		for _, pw := range tc.want {
			pg, ok := searchPackageList(pw.Name, got)
			if !ok {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}
			if pw.Name != pg.Name {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}
			if pw.Category != pg.Category {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}
			if pw.Version != pg.Version {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}
			if pw.Revision != pg.Revision {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}

		}
	}
}
