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
	// Output
	OutputSelected string
}
