package profiler

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/utils"

	log "github.com/sirupsen/logrus"
)

// Component interface defines functions that can be implemented by the
// system components to be used when collecting USE Metrics.
type Component interface {
	// CollectUtilization calculates the utilization score of a component.
	// It takes in a map of commands and uses it to get the parsed output
	// for the commands it will specify.
	CollectUtilization(cmdOutputs map[string]utils.ParsedOutput) error
	// CollectSaturation calculates the saturation value of a component.
	// It takes in a map of commands and specifies the commands it
	// needs to calculate saturation.
	CollectSaturation(cmdOutputs map[string]utils.ParsedOutput) error
	// CollectErrors finds the errors in a component.
	// It takes in a map of commands to their parsed output and uses that
	// to specify which commands (and therefore output) it needs.
	CollectErrors(cmdOutputs map[string]utils.ParsedOutput) error
	// CollectUSEMetrics collects USEMetrics for the component.
	CollectUSEMetrics(cmdOutputs map[string]utils.ParsedOutput) error
	// USEMetrics returns the USEMetrics of the component.
	USEMetrics() *USEMetrics
	// Name retuns the name of the component.
	Name() string
}

// CPU holds information about the CPU component:
// name and USE Metrics collected.
type CPU struct {
	name    string
	metrics *USEMetrics
}

// CollectUtilization calculates the utilization score for the CPU Component.
// It does this by summing the time spent running non-kernel code (user time),
// time spent running kernel code (system time), and time stolen from a vitual
// virtual machine (steal) to get the total CPU time spent servicing work.
// These values can be found on vmstat's 'us' (user), 'sy' (system), and 'st'
// (steal) columns.
func (c *CPU) CollectUtilization(outputs map[string]utils.ParsedOutput) error {
	cmd := "vmstat"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return fmt.Errorf("missing output for vmstat")
	}
	us, usPresent := parsedOutput["us"]
	if !usPresent {
		return fmt.Errorf("missing vmstat column 'us'")
	}
	sy, syPresent := parsedOutput["sy"]
	if !syPresent {
		return fmt.Errorf("missing vmstat column 'sy'")
	}
	st, stPresent := parsedOutput["st"]
	if !stPresent {
		return fmt.Errorf("missing vmstat column 'st'")
	}
	columns := [][]string{us, sy, st}
	var total int
	// loop over us, sy, st columns and sum their values
	for _, column := range columns {
		sum, err := utils.SumAtoi(column)
		if err != nil {
			return err
		}
		total += sum
	}
	count := len(us)
	c.metrics.Utilization = math.Round((float64(total)/float64(count))*100) / 100
	return nil
}

// calculateCPUCount gets the number of processors in the system.
// It does this by getting the value lscpu's "CPU(s)" row.
func (c *CPU) calculateCPUCount(outputs map[string]utils.ParsedOutput) (int, error) {
	cmd := "lscpu"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return 0, fmt.Errorf("missing output for lscpu")
	}
	val, ok := parsedOutput["CPU(s)"]
	if !ok {
		return 0, fmt.Errorf("missing lscpu row 'CPU(s)'")
	}
	count, err := strconv.Atoi(val[0])
	if err != nil {
		return 0, fmt.Errorf("could not convert %s to an int: %v", val[0], err)
	}
	return count, nil
}

// CollectSaturation calculates the saturation value for the CPU component.
// It does this by comparing the number of runnable processes with the number
// of CPUs in the system. If the number of processes (running or waiting) is
// greater than the CPU count, the CPU component is saturated. The value of
// runnable processes is found on vmstat's 'r' column and CPU count from
// lscpu's "CPU(s)" row.
func (c *CPU) CollectSaturation(outputs map[string]utils.ParsedOutput) error {
	cmd := "vmstat"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return fmt.Errorf("missing output for vmstat")
	}
	running, present := parsedOutput["r"]
	if !present {
		return fmt.Errorf("missing vmstat column 'r'")
	}
	// loop over the "r" column and sum the values
	sum, err := utils.SumAtoi(running)
	if err != nil {
		return err
	}

	num := len(running)
	runningProcs := sum / num
	count, err := c.calculateCPUCount(outputs)
	if err != nil {
		return err
	}
	c.metrics.Saturation = runningProcs > count
	return nil
}

// CollectErrors collects errors for the CPU component.
func (c *CPU) CollectErrors(outputs map[string]utils.ParsedOutput) error {
	// Not yet implemented.
	return nil
}

// USEMetrics returns the USE Metrics for the CPU Component.
func (c *CPU) USEMetrics() *USEMetrics {
	return c.metrics
}

// Name returns the name of the CPU component.
func (c *CPU) Name() string {
	return c.name
}

// CollectUSEMetrics collects USE Metrics for the CPU component.
func (c *CPU) CollectUSEMetrics(outputs map[string]utils.ParsedOutput) error {
	metrics := c.metrics
	metrics.Timestamp = time.Now()
	start := metrics.Timestamp

	var gotErr bool
	if err := c.CollectUtilization(outputs); err != nil {
		gotErr = true
		log.Errorf("failed to collect utilization for CPU: %v", err)
	}
	if err := c.CollectSaturation(outputs); err != nil {
		gotErr = true
		log.Errorf("failed to collect saturation for CPU: %v", err)
	}
	end := time.Now()
	metrics.Interval = end.Sub(start)

	if gotErr {
		err := "failed to collect all USE Metrics for CPU. " +
			"Please check the logs for more information"
		return fmt.Errorf(err)
	}
	return nil
}

// MemCap holds information about the Memory capacity component:
// name and USE Metrics collected.
type MemCap struct {
	name    string
	metrics *USEMetrics
}

// CollectUtilization calculates the utilization score for Memory Capacity.
// It does this by getting the quotient of used memory (main and virtual)
// and total memory (main and virtual). The values for main memory can be
// found on free's "Mem" row while virtual memory stats can be found on the
// "Swap" row. To get the used and total values for each row, free's "used"
// and "total" columns are used.
func (m *MemCap) CollectUtilization(outputs map[string]utils.ParsedOutput) error {
	cmd := "free"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return fmt.Errorf("missing output for free")
	}
	memUsed, muPresent := parsedOutput["Mem:used"]
	if !muPresent {
		return fmt.Errorf("missing free's Mem row and used column")
	}
	swapUsed, suPresent := parsedOutput["Swap:used"]
	if !suPresent {
		return fmt.Errorf("missing free's Swap row and used column")
	}

	memory := [][]string{memUsed, swapUsed}
	var used int
	for _, mem := range memory {
		sum, err := utils.SumAtoi(mem)
		if err != nil {
			return err
		}
		used += sum
	}

	total, err := m.calculateTotalMemory(outputs)
	if err != nil {
		return err
	}
	m.metrics.Utilization = math.Round((float64(used)/float64(total))*1000) / 1000
	return nil
}

// calculateTotalMemory calculates the total memory on the system.
// It does this by summing the total Main and total Swap memory which
// can be found on free's "Mem" row + "total" column, and "Swap" row +
// "total" column.
func (m *MemCap) calculateTotalMemory(outputs map[string]utils.ParsedOutput) (int, error) {
	cmd := "free"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return 0, fmt.Errorf("missing output for free")
	}
	memTotal, mtPresent := parsedOutput["Mem:total"]
	if !mtPresent {
		return 0, fmt.Errorf("missing free's Mem row and total column")
	}
	swapTotal, stPresent := parsedOutput["Swap:total"]
	if !stPresent {
		return 0, fmt.Errorf("missing free's Swap row and total column")
	}
	memory := [][]string{memTotal, swapTotal}
	var total int
	for _, mem := range memory {
		sum, err := utils.SumAtoi(mem)
		if err != nil {
			return 0, err
		}
		total += sum
	}
	return total, nil
}

// CollectSaturation calculates the saturation value for Memory Capacity.
// It does this by checking whether the amount of memory being swapped in
// and out of the disks is significant. This indicates that the system is
// low on memory and the kernel is relying heavily on pages from the swap
// space on the disk. Here we define "significant" as the amount of swapped
// memory amounting to roughly 10% of the total memory." The values for
// memory swapped in and out of disks can be found on vmstat's 'si'
// (swapped in) and 'so' (swapped to) columns.
func (m *MemCap) CollectSaturation(outputs map[string]utils.ParsedOutput) error {
	cmd := "vmstat"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return fmt.Errorf("missing output for vmstat")
	}
	si, siPresent := parsedOutput["si"]
	if !siPresent {
		return fmt.Errorf("missing vmstat column 'si'")
	}
	so, soPresent := parsedOutput["so"]
	if !soPresent {
		return fmt.Errorf("missing vmstat column 'so'")
	}
	memory := [][]string{si, so}
	var swaps int
	for _, swap := range memory {
		sum, err := utils.SumAtoi(swap)
		if err != nil {
			return err
		}
		swaps += sum
	}
	average := swaps / len(si)
	total, err := m.calculateTotalMemory(outputs)
	if err != nil {
		return err
	}
	// ten percent of total memory
	threshold := 0.1 * float64(total)
	m.metrics.Saturation = float64(average) > threshold
	return nil
}

// CollectErrors collects errors for the MemCap component.
func (m *MemCap) CollectErrors(outputs map[string]utils.ParsedOutput) error {
	// Not yet implemented.
	return nil
}

// USEMetrics returns the USE Metrics for the Memory Capacity Component.
func (m *MemCap) USEMetrics() *USEMetrics {
	return m.metrics
}

// Name returns the name of the Memory Capacity component.
func (m *MemCap) Name() string {
	return m.name
}

// CollectUSEMetrics collects USE Metrics for the MemCap component.
func (m *MemCap) CollectUSEMetrics(outputs map[string]utils.ParsedOutput) error {
	metrics := m.metrics
	metrics.Timestamp = time.Now()
	start := metrics.Timestamp

	var gotErr bool
	if err := m.CollectUtilization(outputs); err != nil {
		gotErr = true
		log.Errorf("failed to collect utilization for Memory capacity: %v", err)
	}
	if err := m.CollectSaturation(outputs); err != nil {
		gotErr = true
		log.Errorf("failed to collect saturation for Memory capacity: %v", err)
	}
	end := time.Now()
	metrics.Interval = end.Sub(start)

	if gotErr {
		err := "failed to collect all USE metrics for Memory Capacity. " +
			"Please check the logs for more information"
		return fmt.Errorf(err)
	}
	return nil
}

// StorageDevIO holds information about the Storage device I/O component:
// name and USE Metrics collected.
type StorageDevIO struct {
	name    string
	metrics *USEMetrics
}

// CollectUtilization collects the utilization score for the StorageDevIO component.
// It does this by getting the percentage of elapsed time during which I/O requests
// were issued to the devices. This value can be found on iostat's '%util' column.
func (d *StorageDevIO) CollectUtilization(outputs map[string]utils.ParsedOutput) error {
	cmd := "iostat"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return fmt.Errorf("missing output for iostat")
	}
	util, ok := parsedOutput["%util"]
	if !ok {
		return fmt.Errorf("mising iostat column util")
	}

	total, err := utils.SumParseFloat(util)
	if err != nil {
		return err
	}
	average := total / float64(len(util))
	d.metrics.Utilization = average
	return nil
}

// CollectSaturation collects the saturation value for the StorageDevIO component.
// It does this by comparing the average queue length of requests that were issued
// to the device with 1. If the queue length is greater than 1, then the Storage Device
// component is saturated. The value for the average queue length can be found on
// iostat's 'aqu-sz' column.
func (d *StorageDevIO) CollectSaturation(outputs map[string]utils.ParsedOutput) error {
	cmd := "iostat"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return fmt.Errorf("missig output for iostat")
	}
	queue, ok := parsedOutput["aqu-sz"]
	if !ok {
		return fmt.Errorf("mising iostat column 'aqu-sz'")
	}
	total, err := utils.SumParseFloat(queue)
	if err != nil {
		return err
	}
	average := total / float64(len(queue))
	d.metrics.Saturation = average > 1

	return nil
}

// CollectErrors collects errors for the Storage Device I/O component.
func (d *StorageDevIO) CollectErrors(outputs map[string]utils.ParsedOutput) error {
	// yet to be implemented
	return nil
}

// USEMetrics returns the USE Metrics for the Storage Device I/O Component.
func (d *StorageDevIO) USEMetrics() *USEMetrics {
	return d.metrics
}

// Name returns the name of the Storage Device I/O component.
func (d *StorageDevIO) Name() string {
	return d.name
}

// CollectUSEMetrics collects USE Metrics for the Storage Device I/O component.
func (d *StorageDevIO) CollectUSEMetrics(outputs map[string]utils.ParsedOutput) error {
	metrics := d.metrics
	metrics.Timestamp = time.Now()
	start := metrics.Timestamp

	var gotErr bool
	if err := d.CollectUtilization(outputs); err != nil {
		gotErr = true
		log.Errorf("failed to collect utilization for Storage Device I/O: %v", err)
	}
	if err := d.CollectSaturation(outputs); err != nil {
		gotErr = true
		log.Errorf("failed to collect saturation for Storage Device I/O: %v", err)
	}
	end := time.Now()
	metrics.Interval = end.Sub(start)

	if gotErr {
		err := "failed to collect all USE metrics for Storage Device I/O. " +
			"Please check the logs for more information"
		return fmt.Errorf(err)
	}
	return nil
}

// GenerateUSEReport generates USE Metrics for all the components
// as well as an analysis string to help the diagnose performance issues.
func GenerateUSEReport(components []Component, cmds []Command) (USEReport, error) {
	useReport := USEReport{Components: components}
	outputs := make(map[string]utils.ParsedOutput)

	for _, cmd := range cmds {
		output, err := cmd.Run()
		if err != nil {
			log.Errorf("failed to run %q command: %v", cmd.Name(), err)
			continue
		}
		name := cmd.Name()
		outputs[name] = output
	}
	var failed []string
	for _, s := range components {
		if err := s.CollectUSEMetrics(outputs); err != nil {
			log.Errorf("failed to collect USE metrics for %q", s.Name())
			failed = append(failed, s.Name())
		}
	}
	if len(failed) != 0 {
		err := "failed to generate USE report for %s components" +
			"Please check the logs for more information"
		return useReport, fmt.Errorf(err, failed)

	}
	return useReport, nil
}
