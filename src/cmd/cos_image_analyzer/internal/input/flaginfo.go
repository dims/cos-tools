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

	//Verbosity of output
	Verbose bool

	// File used to compress output from Rootfs and OS-Config difference
	// (either user provided or default CompressRootfs.txt)
	CompressRootfsFile string

	// File used to compress output from Stateful-partition difference
	// (either user provided or default CompressStateful.txt)
	CompressStatefulFile string

	// Output
	OutputSelected string
}
