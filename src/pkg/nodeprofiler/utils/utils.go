// Package utils defines common structs used by the COS Node Profiler Agent
package utils

import "time"

// USEMetrics contain the USE metrics (utilization, saturation, errors)
// for a particular component of the system.
type USEMetrics struct {
	// Timestamp refers to the point in time when the USE metrics for this
	// component was collected.
	Timestamp time.Time
	// Interval refers to the time interval for which the USE metrics for this
	// component was collected.
	Interval time.Duration
	// Utilization is the percent over a time interval for which the resource
	// was busy servicing work.
	Utilization float64
	// Saturation is the degree to which the resource has extra work which it
	// can’t service. The value for Saturation has different meanings
	// depending on the component being analyzed. But for simplicity sake
	// Saturation here is just a bool which tells us whether this specific
	// component is saturated or not.
	Saturation bool
	// Errors is the number of errors seen in the component over a given
	// time interval.
	Errors int64
}

// USEReport contains the USE Report from a single run of the node profiler.
// The USE Report contains helpful information to help diagnose performance
// issues seen by customers on their k8s clusters.
type USEReport struct {
	// Components contains the USE Metrics for each component of the system.
	// Such components include CPU, memory, network, storage, etc.
	Components []USEMetrics
	// Analysis provides insights into the USE metrics collected, including
	// a guess as to which component may be causing performance issues.
	Analysis string
}

// ProfilerReport contains debugging information provided by the profiler
// tool. Currently, it will only provide USEMetrics (Utilization,
// Saturation, Errors), kernel trace outputs, and the outputs of
// arbitrary shell commands provided by the user.
// In future, we can add following different types of dynamic reports:
//
// type PerfReport - Captures perf command output
// type STraceReport - Captures strace output
// type BPFTraceReport - Allows users to add eBPF hooks and capture its
//                       output
type ProfilerReport struct {
	// Static reports
	// USEMetrics provides Utilization, Saturation and Errors for different
	// components on the system
	USEInfo USEReport
	// RawCommands captures the output of arbitrary shell commands provided
	// by the user. Example usage: count # of systemd units; count # of
	// cgroups
	RawCommands map[string][]byte
	// Dynamic tracing reports
	// KernelTraces captures the output of the ftrace command. The key is the
	// kernel trace point and the value is the output
	KernelTraces map[string][]byte
}

// ProfilerConfig tells the profiler which dynamic reports it should
// generate and capture.
type ProfilerConfig struct {
	// KernelTracePoints are the trace points we should insert and capture.
	KernelTracePoints []string
	// RawCommands are the shell commands that we should run and capture
	// output.
	RawCommands []string
}
