package binary

import (
	"testing"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
)

// test GetBinaryInf function
func TestGetBinaryInfo(t *testing.T) {
	kclImage1 := `linux /syslinux/vmlinuz.A init=/usr/lib/systemd/systemd boot=local rootwait ro noresume noswap loglevel=7 noinitrd console=ttyS0 security=apparmor virtio_net.napi_tx=1 systemd.unified_cgroup_hierarchy=false systemd.legacy_systemd_cgroup_controller=false csm.disabled=1  dm_verity.error_behavior=3 dm_verity.max_bios=-1 dm_verity.dev_wait=1       i915.modeset=1 cros_efi root=/dev/dm-0`
	for _, tc := range []struct {
		image    *input.ImageInfo
		flagInfo *input.FlagInfo
		want     *input.ImageInfo
	}{ // Version and BuildID
		{image: &input.ImageInfo{TempDir: "../testdata/image1/", RootfsPartition3: "../testdata/image1/rootfs/"},
			flagInfo: &input.FlagInfo{LocalPtr: true},
			want:     &input.ImageInfo{RootfsPartition3: "../testdata/image1", Version: "81", BuildID: "12871.119.0"}},
		{image: &input.ImageInfo{},
			flagInfo: &input.FlagInfo{LocalPtr: true},
			want:     &input.ImageInfo{}},
		{image: &input.ImageInfo{TempDir: "../testdata/image2/", RootfsPartition3: "../testdata/image2/rootfs/"},
			flagInfo: &input.FlagInfo{LocalPtr: true},
			want:     &input.ImageInfo{RootfsPartition3: "../testdata/image2/rootfs", Version: "77", BuildID: "12371.273.0"}},

		// Kernel Command Line
		{image: &input.ImageInfo{TempDir: "../testdata/image1", EFIPartition12: "../testdata/image1/efi/"},
			flagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Kernel-command-line"}},
			want:     &input.ImageInfo{TempDir: "../testdata/image1", EFIPartition12: "../testdata/image1/efi/", KernelCommandLine: kclImage1}},
	} {
		GetBinaryInfo(tc.image, tc.flagInfo)

		if tc.want.Version != tc.image.Version {
			t.Fatalf("GetBinaryInfo expected: %v, got: %v", tc.want.Version, tc.image.Version)
		}
		if tc.want.BuildID != tc.image.BuildID {
			t.Fatalf("GetBinaryInfo expected: %v, got: %v", tc.want.BuildID, tc.image.BuildID)
		}
		if tc.image.KernelCommandLine != tc.want.KernelCommandLine {
			t.Fatalf("Diff kernel command line expected:$%v$, got:$%v$", tc.want.KernelCommandLine, tc.image.KernelCommandLine)
		}
	}
}
