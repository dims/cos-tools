package binary

import (
	"testing"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools.git/src/cmd/cos_image_analyzer/internal/utilities"
)

// test Diff function
func TestDiff(t *testing.T) {
	testCompressRootfsSlice := []string{"/proc/", "/usr/lib/", "/util/", "/etc/os-release/", "/etc/sysctl.d/"}
	testCompressStatefulSlice := []string{"/var_overlay/"}

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
		"/etc/docker/": `Configs for directory /etc/docker/
diff -r --no-dereference ../testdata/image1/rootfs/etc/docker/credentials.txt ../testdata/image2/rootfs/etc/docker/credentials.txt
1,2c1,2
< Name: docker.10.2.4
< job: makes micro kernels
\ No newline at end of file
---
> Name: docker.10.2.1
> job: makes macro kernels
\ No newline at end of file
Only in ../testdata/image1/rootfs/etc/docker/util: docker.txt
Only in ../testdata/image1/rootfs/etc/docker/util: lib32
Only in ../testdata/image2/rootfs/etc/docker/util: lib64`,
		"/etc/os-release/": `Configs for file /etc/os-release/
1c1
< BUILD_ID=12871.119.0
---
> BUILD_ID=12371.273.0
3c3
< KERNEL_COMMIT_ID=fa84f12c6d738af9486e69a006a57df923f9476a
---
> KERNEL_COMMIT_ID=5d4ffd91281840f7a118143d77fbefb02e87943c
5c5
< VERSION_ID=81
---
> VERSION_ID=77
8c8
< VERSION=81
---
> VERSION=77`,
		"/etc/sysctl.d/": `Configs for directory /etc/sysctl.d/
diff -r --no-dereference ../testdata/image1/rootfs/etc/sysctl.d/00-sysctl.conf ../testdata/image2/rootfs/etc/sysctl.d/00-sysctl.conf
8c8
< net.ipv4.conf.all.rp_filter = 1
---
> net.ipv4.conf.all.rp_filter = 2
11c11,14
< net.ipv4.tcp_slow_start_after_idle = 0
\ No newline at end of file
---
> net.ipv4.tcp_slow_start_after_idle = 1
> 
> # dumby variable
> net.ipv4.conf = 2
\ No newline at end of file`}

	testBriefOSConfig := map[string]string{
		"/etc/docker/": `Configs for directory /etc/docker/
diff -r --no-dereference ../testdata/image1/rootfs/etc/docker/credentials.txt ../testdata/image2/rootfs/etc/docker/credentials.txt
1,2c1,2
< Name: docker.10.2.4
< job: makes micro kernels
\ No newline at end of file
---
> Name: docker.10.2.1
> job: makes macro kernels
\ No newline at end of file
Only in ../testdata/image1/rootfs/etc/docker/util: docker.txt
Only in ../testdata/image1/rootfs/etc/docker/util: lib32
Only in ../testdata/image2/rootfs/etc/docker/util: lib64`}

	// Stateful test data
	testVerboseStatefulDiff := `Only in ../testdata/image1/stateful/dev_image: image1_dev.txt
Only in ../testdata/image2/stateful/dev_image: image2_dev.txt
Only in ../testdata/image1/stateful/var_overlay/db: image1_data.txt
Only in ../testdata/image2/stateful/var_overlay/db: image2_data.txt`
	testBriefStatefulDiff := `Only in ../testdata/image1/stateful/dev_image: image1_dev.txt
Only in ../testdata/image2/stateful/dev_image: image2_dev.txt
Unique files in ../testdata/image1/stateful/var_overlay
Unique files in ../testdata/image2/stateful/var_overlay`

	// Partition Structure data
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

	// Kernel configs data
	testKernelConfigsImage1 := `#
# Compiler: Chromium OS 10.0_pre377782_p20200113-r10 clang version 10.0.0 (/var/cache/chromeos-cache/distfiles/host/egit-src/llvm-project 4e8231b5cf0f5f62c7a51a857e29f5be5cb55734)
#
CONFIG_GCC_VERSION=0
CONFIG_CC_IS_CLANG=y
CONFIG_CLANG_VERSION=100000

#
# General setup
#
CONFIG_INIT_ENV_ARG_LIMIT=32
CONFIG_LOCALVERSION=""
CONFIG_BUILD_SALT=""
CONFIG_HAVE_KERNEL_GZIP=y`
	testKernelConfigsDiff := `2c2
< # Compiler: Chromium OS 10.0_pre377782_p20200113-r10 clang version 10.0.0 (/var/cache/chromeos-cache/distfiles/host/egit-src/llvm-project 4e8231b5cf0f5f62c7a51a857e29f5be5cb55734)
---
> # Compiler: Chromium OS 9.0_pre361749_p20190714-r4 clang version 9.0.0 (/var/cache/chromeos-cache/distfiles/host/egit-src/llvm-project c11de5eada2decd0a495ea02676b6f4838cd54fb) (based on LLVM 9.0.0svn)
6c6
< CONFIG_CLANG_VERSION=100000
---
> CONFIG_CLANG_VERSION=90000
11a12
> # CONFIG_COMPILE_TEST is not set
12a14
> # CONFIG_LOCALVERSION_AUTO is not set
14c16
< CONFIG_HAVE_KERNEL_GZIP=y
\ No newline at end of file
---
> CONFIG_HAVE_KERNEL_GZIP=y`

	// Kernel Command Line data
	kclImage1 := `linux /syslinux/vmlinuz.A init=/usr/lib/systemd/systemd boot=local rootwait ro dm_verity.dev_wait=50`
	kclImage2 := `linux /syslinux/vmlinuz.A init=/usr/lib32/systemd/ ro dm_verity.dev_wait=1      i915.modeset=1 cros_efi`
	testKCLImage1 := map[string]string{"Image1 KCL": kclImage1}
	testKCLDiff := map[string]string{
		"boot": `d
< boot=local`,
		"cros_efi": `a
> cros_efi`,
		"dm_verity.dev_wait": `c
< dm_verity.dev_wait=50
---
> dm_verity.dev_wait=1`,
		"i915.modeset": `a
> i915.modeset=1`,
		"init": `c
< init=/usr/lib/systemd/systemd
---
> init=/usr/lib32/systemd/`,
		"rootwait": `d
< rootwait`}

	// Sysctl settings
	testSysctlSettingsImage1 := `# /etc/sysctl.conf
# Look in /proc/sys/ for all the things you can setup.
#

# Enables source route verification
net.ipv4.conf.default.rp_filter = 1
# Enable reverse path
net.ipv4.conf.all.rp_filter = 1

# Disable shrinking the cwnd when connection is idle
net.ipv4.tcp_slow_start_after_idle = 0`
	testSysctlSettingsDiff := `8c8
< net.ipv4.conf.all.rp_filter = 1
---
> net.ipv4.conf.all.rp_filter = 2
11c11,14
< net.ipv4.tcp_slow_start_after_idle = 0
\ No newline at end of file
---
> net.ipv4.tcp_slow_start_after_idle = 1
> 
> # dumby variable
> net.ipv4.conf = 2
\ No newline at end of file`
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
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Rootfs"}, Verbose: true, CompressRootfsSlice: testCompressRootfsSlice},
			want:     &Differences{Rootfs: testVerboseRootfsDiff}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", RootfsPartition3: "../testdata/image2/rootfs/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Rootfs"}, Verbose: false, CompressRootfsSlice: testCompressRootfsSlice},
			want:     &Differences{Rootfs: testBriefRootfsDiff}},

		// OS Config difference test
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"OS-config"}},
			want:     &Differences{}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", RootfsPartition3: "../testdata/image2/rootfs/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"OS-config"}, Verbose: true, CompressRootfsSlice: testCompressRootfsSlice},
			want:     &Differences{OSConfigs: testVerboseOSConfig}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", RootfsPartition3: "../testdata/image1/rootfs/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", RootfsPartition3: "../testdata/image2/rootfs/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"OS-config"}, Verbose: false, CompressRootfsSlice: testCompressRootfsSlice},
			want:     &Differences{OSConfigs: testBriefOSConfig}},

		// Stateful difference test
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", StatePartition1: "../testdata/image1/stateful/"},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Stateful-partition"}},
			want:     &Differences{}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", StatePartition1: "../testdata/image1/stateful/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", StatePartition1: "../testdata/image2/stateful/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Stateful-partition"}, Verbose: true, CompressStatefulSlice: testCompressStatefulSlice},
			want:     &Differences{Stateful: testVerboseStatefulDiff}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", StatePartition1: "../testdata/image1/stateful/"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", StatePartition1: "../testdata/image2/stateful/"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Stateful-partition"}, Verbose: false, CompressStatefulSlice: testCompressStatefulSlice},
			want:     &Differences{Stateful: testBriefStatefulDiff}},

		// Partition Structure
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", PartitionFile: "../testdata/image1/partitions.txt"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", PartitionFile: "../testdata/image2/partitions.txt"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Partition-structure"}},
			want:     &Differences{PartitionStructure: testPartitionStructure}},

		// Kernel Configs
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", KernelConfigsFile: "../testdata/image1/usr/src/linux-headers-4.19.112+/.config"},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Kernel-configs"}},
			want:     &Differences{KernelConfigs: testKernelConfigsImage1}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", KernelConfigsFile: "../testdata/image1/usr/src/linux-headers-4.19.112+/.config"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", KernelConfigsFile: "../testdata/image2/usr/src/linux-headers-4.19.112+/.config"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Kernel-configs"}},
			want:     &Differences{KernelConfigs: testKernelConfigsDiff}},

		// Kernel command line
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", KernelCommandLine: kclImage1},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Kernel-command-line"}},
			want:     &Differences{KernelCommandLine: testKCLImage1}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", KernelCommandLine: kclImage1},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", KernelCommandLine: kclImage2},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Kernel-command-line"}},
			want:     &Differences{KernelCommandLine: testKCLDiff}},

		// Sysctl settings
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", SysctlSettingsFile: "../testdata/image1/rootfs/etc/sysctl.d/00-sysctl.conf"},
			Image2:   &input.ImageInfo{},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Sysctl-settings"}},
			want:     &Differences{SysctlSettings: testSysctlSettingsImage1}},
		{Image1: &input.ImageInfo{TempDir: "../testdata/image1", SysctlSettingsFile: "../testdata/image1/rootfs/etc/sysctl.d/00-sysctl.conf"},
			Image2:   &input.ImageInfo{TempDir: "../testdata/image2", SysctlSettingsFile: "../testdata/image2/rootfs/etc/sysctl.d/00-sysctl.conf"},
			FlagInfo: &input.FlagInfo{BinaryTypesSelected: []string{"Sysctl-settings"}},
			want:     &Differences{SysctlSettings: testSysctlSettingsDiff}},
	} {
		got, _ := Diff(tc.Image1, tc.Image2, tc.FlagInfo)

		if !utilities.EqualArrays(tc.want.Version, got.Version) {
			t.Fatalf("Diff expected version %v, got: %v", tc.want.Version, got.Version)
		}
		if !utilities.EqualArrays(tc.want.BuildID, got.BuildID) {
			t.Fatalf("Diff expected BuildID %v, got: %v", tc.want.BuildID, got.BuildID)
		}
		if tc.want.Rootfs != got.Rootfs {
			t.Fatalf("Diff expected Rootfs diff \n%v\ngot:\n%v", tc.want.Rootfs, got.Rootfs)
		}
		for etcEntry := range tc.want.OSConfigs {
			if res, _ := utilities.CmpMapValues(tc.want.OSConfigs, got.OSConfigs, etcEntry); res != 0 {
				t.Fatalf("Diff expected OSConfigs \n%v\ngot:\n%v", tc.want.OSConfigs, got.OSConfigs)
			}
		}
		if tc.want.Stateful != got.Stateful {
			t.Fatalf("Diff expected stateful diff \n%v\ngot:\n%v", tc.want.Stateful, got.Stateful)
		}
		if tc.want.PartitionStructure != got.PartitionStructure {
			t.Fatalf("Diff expected partition structure diff \n$%v$\ngot:\n$%v$", tc.want.PartitionStructure, got.PartitionStructure)
		}
		for kclParameter, diff := range tc.want.KernelCommandLine {
			if diff != got.KernelCommandLine[kclParameter] {
				t.Fatalf("Diff expected kernel command line \n$%v$\ngot:\n$%v$", tc.want.KernelCommandLine, got.KernelCommandLine)
			}
		}
		if tc.want.KernelConfigs != got.KernelConfigs {
			t.Fatalf("Diff expected kernel configs diff \n$%v$\ngot:\n$%v$", tc.want.KernelConfigs, got.KernelConfigs)
		}
		if tc.want.SysctlSettings != got.SysctlSettings {
			t.Fatalf("Diff expected sysctl settings \n$%v$\ngot:\n$%v$", tc.want.SysctlSettings, got.SysctlSettings)
		}
	}
}
