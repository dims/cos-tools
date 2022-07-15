// Package gpuconfig implements routines for manipulating proto based
// GPU build configuration files.
//
// It also implements the construction of these configs for
// the COS Image and the COS Kernel CI.
package gpuconfig

//go:generate protoc --go_out=:./pb -I. proto/config.proto
