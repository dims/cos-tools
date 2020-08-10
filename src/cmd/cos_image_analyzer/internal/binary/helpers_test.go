package binary

import (
	"testing"
)

// test DirectoryDiff function
func TestDirectoryDiff(t *testing.T) {
	testVerboseOutput := `Files ../testdata/image1/rootfs/lib64/python.txt and ../testdata/image2/rootfs/lib64/python.txt differ
Files ../testdata/image1/rootfs/proc/security/access.conf and ../testdata/image2/rootfs/proc/security/access.conf differ
Files ../testdata/image1/rootfs/proc/security/configs and ../testdata/image2/rootfs/proc/security/configs differ
Only in ../testdata/image1/rootfs/usr/lib: usr-lib-image1
Only in ../testdata/image2/rootfs/usr/lib: usr-lib-image2`
	testBriefOutput := `Files ../testdata/image1/rootfs/lib64/python.txt and ../testdata/image2/rootfs/lib64/python.txt differ
Files in ../testdata/image1/rootfs/proc and ../testdata/image2/rootfs/proc differ
Unique files in ../testdata/image1/rootfs/usr/lib
Unique files in ../testdata/image2/rootfs/usr/lib`

	for _, tc := range []struct {
		dir1           string
		dir2           string
		root           string
		verbose        bool
		compressedDirs []string
		want           string
	}{
		{dir1: "../testdata/image1/rootfs/", dir2: "../testdata/image2/rootfs/", root: "rootfs", verbose: true, compressedDirs: []string{"/proc/", "/usr/lib/"}, want: testVerboseOutput},
		{dir1: "../testdata/image1/rootfs/", dir2: "../testdata/image2/rootfs/", root: "rootfs", verbose: false, compressedDirs: []string{"/proc/", "/usr/lib/"}, want: testBriefOutput},
	} {
		got, _ := directoryDiff(tc.dir1, tc.dir2, tc.root, tc.verbose, tc.compressedDirs)
		if got != tc.want {
			t.Fatalf("directoryDiff expected:\n$%v$\ngot:\n$%v$", tc.want, got)
		}
	}
}

// test PureDiff function
func TestPureDiff(t *testing.T) {
	testOutput1 := `1c1
< testing 123 can you hear me?
---
> testing 456 can you hear me?`
	testOutput2 := `1c1
< These are not the configs you are looking for
---
> These are the configs you are looking for`
	for _, tc := range []struct {
		input1 string
		input2 string
		want   string
	}{
		{input1: "../testdata/image1/rootfs/proc/security/access.conf", input2: "../testdata/image2/rootfs/proc/security/access.conf", want: testOutput1},
		{input1: "../testdata/image1/rootfs/proc/security/configs", input2: "../testdata/image2/rootfs/proc/security/configs", want: testOutput2},
		{input1: "../testdata/image1/rootfs/proc/security/lib-image1", input2: "../testdata/image2/rootfs/proc/security/lib-image2", want: ""},
	} {
		got, _ := pureDiff(tc.input1, tc.input2)
		if got != tc.want {
			t.Fatalf("PureDiff expected:\n$%v$\ngot:\n$%v$", tc.want, got)
		}
	}
}
