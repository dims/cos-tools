package binary

import (
	"testing"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// test Diff function
func TestDiff(t *testing.T) {
	// Rootfs test data
	testVerboseRootfsDiff := `Files ../testdata/image1/rootfs/lib64/python.txt and ../testdata/image2/rootfs/lib64/python.txt differ
Files ../testdata/image1/rootfs/proc/security/access.conf and ../testdata/image2/rootfs/proc/security/access.conf differ
Files ../testdata/image1/rootfs/proc/security/configs and ../testdata/image2/rootfs/proc/security/configs differ
Only in ../testdata/image1/rootfs/usr/lib: usr-lib-image1
Only in ../testdata/image2/rootfs/usr/lib: usr-lib-image2`
	testBriefRootfsDiff := `Files ../testdata/image1/rootfs/lib64/python.txt and ../testdata/image2/rootfs/lib64/python.txt differ
Files in ../testdata/image1/rootfs/proc and ../testdata/image2/rootfs/proc differ
Unique files in ../testdata/image1/rootfs/usr/lib
Unique files in ../testdata/image2/rootfs/usr/lib`

	// OS Config test data
	testVerboseOSConfig := map[string]string{
		"docker": `Files ../testdata/image1/rootfs/etc/docker/credentials.txt and ../testdata/image2/rootfs/etc/docker/credentials.txt differ
Only in ../testdata/image1/rootfs/etc/docker/util: docker.txt
Only in ../testdata/image1/rootfs/etc/docker/util: lib32
Only in ../testdata/image2/rootfs/etc/docker/util: lib64`,
		"os-release": "Files ../testdata/image1/rootfs/etc/os-release and ../testdata/image2/rootfs/etc/os-release differ"}
	testBriefOSConfig := map[string]string{
		"docker": `Files ../testdata/image1/rootfs/etc/docker/credentials.txt and ../testdata/image2/rootfs/etc/docker/credentials.txt differ
Only in ../testdata/image1/rootfs/etc/docker/util: docker.txt
Only in ../testdata/image1/rootfs/etc/docker/util: lib32
Only in ../testdata/image2/rootfs/etc/docker/util: lib64`,
		"os-release": "Files ../testdata/image1/rootfs/etc/os-release and ../testdata/image2/rootfs/etc/os-release differ"}

	// Stateful test data
	testVerboseStatefulDiff := `Only in ../testdata/image1/stateful/dev_image: image1_dev.txt
Only in ../testdata/image2/stateful/dev_image: image2_dev.txt
Only in ../testdata/image1/stateful/var_overlay/db: image1_data.txt
Only in ../testdata/image2/stateful/var_overlay/db: image2_data.txt`
	testBriefStatefulDiff := `Only in ../testdata/image1/stateful/dev_image: image1_dev.txt
Only in ../testdata/image2/stateful/dev_image: image2_dev.txt
Unique files in ../testdata/image1/stateful/var_overlay
Unique files in ../testdata/image2/stateful/var_overlay`

	// Partition Structure test data
	testPartitionStructure := `1c1
< Disk /img_disks/cos_81_12871_119_disk/disk.raw: 20971520 sectors, 10.0 GiB
---
> Disk /img_disks/cos_77_12371_273_disk/disk.raw: 20971520 sectors, 10.0 GiB
3c3
< Disk identifier (GUID): 0274E604-5DE3-5E4E-A4FD-F4D00FBBD7AA
---
> Disk identifier (GUID): AB9719F2-3174-4F46-8079-1CF470D2D9BC
11c11
<    1         8704000        18874476   4.8 GiB     8300  STATE
---
>    1         8704000        18874476   4.8 GiB     0700  STATE
18c18
<    8           86016          118783   16.0 MiB    8300  OEM
---
>    8           86016          118783   16.0 MiB    0700  OEM`

	for _, tc := range []struct {
		Image1   *input.ImageInfo
		Image2   *input.ImageInfo
		FlagInfo *input.FlagInfo
		want     *Differences
	}{ // Version and BuildID difference tests
		{Image1: &input.ImageInfo{RootfsPartition3: "../testdata/image1/rootfs/", Version: "81", BuildID: "12871.119.0"},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Version", "BuildID"}},
			want:     &Differences{Version: []string{"81", ""}, BuildID: []string{"12871.119.0", ""}}},

		{Image1: &input.ImageInfo{RootfsPartition3: "../testdata/image1/rootfs/", Version: "81", BuildID: "12871.119.0"},
			Image2:   &input.ImageInfo{RootfsPartition3: "../testdata/image2/rootfs/", Version: "77", BuildID: "12371.273.0"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Version", "BuildID"}},
			want:     &Differences{Version: []string{"81", "77"}, BuildID: []string{"12871.119.0", "12371.273.0"}}},

		{Image1: &input.ImageInfo{RootfsPartition3: "../testdata/image1/rootfs/", Version: "81", BuildID: "12871.119.0"},
			Image2:   &input.ImageInfo{RootfsPartition3: "../testdata/image1/rootfs/", Version: "81", BuildID: "12871.119.1"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Version", "BuildID"}},
			want:     &Differences{Version: []string{}, BuildID: []string{"12871.119.0", "12871.119.1"}}},

		{Image1: &input.ImageInfo{RootfsPartition3: "../testdata/image1/rootfs/", Version: "81", BuildID: "12871.119.0"},
			Image2:   &input.ImageInfo{RootfsPartition3: "../testdata/image1/rootfs/", Version: "81", BuildID: "12871.119.0"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Version", "BuildID"}},
			want:     &Differences{}},

		// Rootfs difference test
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Rootfs"}},
			want:     &Differences{}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", RootfsPartition3: "../testdata/image2/rootfs/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Rootfs"}, Verbose: true, CompressRootfsFile: "../testdata/CompressRootfsFile.txt"},
			want:     &Differences{Rootfs: testVerboseRootfsDiff}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", RootfsPartition3: "../testdata/image2/rootfs/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Rootfs"}, Verbose: false, CompressRootfsFile: "../testdata/CompressRootfsFile.txt"},
			want:     &Differences{Rootfs: testBriefRootfsDiff}},

		// OS Config difference test
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"OS-config"}},
			want:     &Differences{}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", RootfsPartition3: "../testdata/image2/rootfs/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"OS-config"}, Verbose: true, CompressRootfsFile: "../testdata/CompressRootfsFile.txt"},
			want:     &Differences{OSConfigs: testVerboseOSConfig}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", RootfsPartition3: "../testdata/image2/rootfs/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"OS-config"}, Verbose: false, CompressRootfsFile: "../testdata/CompressRootfsFile.txt"},
			want:     &Differences{OSConfigs: testBriefOSConfig}},

		// Stateful difference test
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Stateful-partition"}},
			want:     &Differences{}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", StatePartition1: "../testdata/image1/stateful/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", StatePartition1: "../testdata/image2/stateful/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Stateful-partition"}, Verbose: true, CompressStatefulFile: "../testdata/CompressStatefulFile.txt"},
			want:     &Differences{Stateful: testVerboseStatefulDiff}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", StatePartition1: "../testdata/image1/stateful/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", StatePartition1: "../testdata/image2/stateful/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Stateful-partition"}, Verbose: false, CompressStatefulFile: "../testdata/CompressStatefulFile.txt"},
			want:     &Differences{Stateful: testBriefStatefulDiff}},

		// Partition Structure
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Partition-structure"}},
			want:     &Differences{}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", PartitionFile: "../testdata/image1/partitions.txt"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", PartitionFile: "../testdata/image2/partitions.txt"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Partition-structure"}},
			want:     &Differences{PartitionStructure: testPartitionStructure}},
	} {
		got, _ := Diff(tc.Image1, tc.Image2, tc.FlagInfo)

		if !utilities.EqualArrays(tc.want.Version, got.Version) {
			t.Fatalf("Diff expected version %v, got: %v", tc.want.Version, got.Version)
		}
		if !utilities.EqualArrays(tc.want.BuildID, got.BuildID) {
			t.Fatalf("Diff expected BuildID %v, got: %v", tc.want.BuildID, got.BuildID)
		}
		if tc.want.Rootfs != got.Rootfs {
			t.Fatalf("Diff expected Rootfs diff \n$%v$\ngot:\n$%v$", tc.want.Rootfs, got.Rootfs)
		}
		for etcEntry := range got.OSConfigs {
			if res, _ := utilities.CmpMapValues(tc.want.OSConfigs, got.OSConfigs, etcEntry); res != 0 {
				t.Fatalf("Diff expected OSConfigs \n$%v$\ngot:\n$%v$", tc.want.OSConfigs, got.OSConfigs)
			}
		}
		if tc.want.Stateful != got.Stateful {
			t.Fatalf("Diff expected stateful diff \n$%v$\ngot:\n$%v$", tc.want.Stateful, got.Stateful)
		}
		if tc.want.PartitionStructure != got.PartitionStructure {
			t.Fatalf("Diff expected partition structure diff \n$%v$\ngot:\n$%v$", tc.want.PartitionStructure, got.PartitionStructure)
		}
	}
}
