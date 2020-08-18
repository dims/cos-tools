package input

// FlagInfo holds input preference from the user
type FlagInfo struct {
	// Args
	Image1 string
	Image2 string

	// Input Types
	LocalPtr    bool
	GcsPtr      bool
	CosCloudPtr bool

	// Authentication
	ProjectIDPtr string

	// Binary
	BinaryDiffPtr       string
	BinaryTypesSelected []string
	// Package
	PackageSelected bool
	// Commit
	CommitSelected bool
	// Release Notes
	ReleaseNotesSelected bool

	// Verbosity of output
	// If true, full Rootfs, Os-Config, and Stateful Partition output is shown.
	// Else false (default), Rootfs and Stateful Partition directories listed on files
	// 	pointed to by CompressRootfsFile and CompressStatefulFile respectively are compressed.
	// 	For OS-configs difference, all /etc entries that are listed in CompressRootfsFile are ignored.
	Verbose bool

	// File used to compress directories in the output from Rootfs difference and
	// for ignore entries under /etc for OS-Config difference
	// (either user provided or default CompressRootfs.txt)
	CompressRootfsFile string
	// Slice of CompressRootfsFile
	CompressRootfsSlice []string

	// File used to compress directories in the output from Stateful-partition difference
	// (either user provided or default CompressStateful.txt)
	CompressStatefulFile string
	// Slice of CompressRootfsFile
	CompressStatefulSlice []string

	// Output
	OutputSelected string
}
