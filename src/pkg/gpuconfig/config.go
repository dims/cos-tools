// Package gpuconfig implements routines for manipulating proto based
// GPU build configuration files.
//
// It also implements the construction of these configs for
// the COS Image and the COS Kernel CI.
package gpuconfig

//go:generate protoc --go_out=:./pb -I. proto/config.proto

import (
	"time"

	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig/pb"
)

type GPUPrecompilationConfig struct {
	ProtoConfig   *pb.COSGPUBuildRequest `json:"-"`
	DriverVersion string                 `json:"driver_version"`
	Milestone     string                 `json:"milestone"`
	Version       string                 `json:"version"`
	VersionType   string                 `json:"version_type"`
}

// stubbing out current time as a function - allows current time to be injected into functions across gpuconfig package and testing
var timeNow = func() time.Time {
	return time.Now()
}
