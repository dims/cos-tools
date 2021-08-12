package profiler

import (
	"fmt"
	"math"
	"strconv"
	"strings"
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
	// USEMetrics returns the USEMetrics of the component.
	USEMetrics() *USEMetrics
	// Name returns the name of the component.
	Name() string
	// AdditionalInformation returns additional information unique to each
	// component.
	AdditionalInformation() string
}

// CPU holds information about the CPU component:
// name and USE Metrics collected.
type CPU struct {
	name    string
	metrics *USEMetrics
}

// NewCPU holds information about the CPU component:
// this can be used to initialize CPU outside of the
// profiler package.
func NewCPU(name string) *CPU {

	return &CPU{
		name:    name,
		metrics: &USEMetrics{},
	}
}

// AdditionalInformation returns additional information unique to the
// the CPU component.
func (c *CPU) AdditionalInformation() string {
	return ""
}

// Name returns the name of the CPU component.
func (c *CPU) Name() string {
	return c.name
}

// USEMetrics returns USEMetrics for the CPU component.
func (c *CPU) USEMetrics() *USEMetrics {
	return c.metrics
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
		return fmt.Errorf("missing output for %q", cmd)
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

	if len(us) == 0 {
		return fmt.Errorf("no vmstat report collected")
	} else if len(us) == 1 {
		err := "only averages values since last reboot were collected. To calculate utilization value" +
			" reflecting current conditions of component, additional reports are needed"
		return fmt.Errorf(err)
	}

	// ignore the first values of 'us', 'sy' and 'st' since they reflect averages
	// since last reboot and can bring averages down
	us = us[1:]
	sy = sy[1:]
	st = st[1:]

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
		return 0, fmt.Errorf("missing output for %q", cmd)
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
		return fmt.Errorf("missing output for %q", cmd)
	}
	running, present := parsedOutput["r"]
	if !present {
		return fmt.Errorf("missing vmstat column 'r'")
	}

	if len(running) == 0 {
		return fmt.Errorf("no vmstat report collected")
	} else if len(running) == 1 {
		err := "only averages values since last reboot were collected. To calculate utilization value" +
			" reflecting current conditions of component, additional reports are needed"
		return fmt.Errorf(err)
	}

	// ignore the first values of 'r' since they reflect averages since last
	// reboot and can bring the average down
	running = running[1:]
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

// MemCap holds information about the Memory capacity component:
// name and USE Metrics collected.
type MemCap struct {
	name    string
	metrics *USEMetrics
}

// NewMemCap holds information about the Memory capacity component:
// this can be used to initialize MemCap outside of the
// profiler package.
func NewMemCap(name string) *MemCap {

	return &MemCap{
		name:    name,
		metrics: &USEMetrics{},
	}
}

// AdditionalInformation returns additional information unique to the
// the MemCap component.
func (m *MemCap) AdditionalInformation() string {
	info := "The utilization value for this component was calculated as a " +
		"percentage of total Main memory while saturation was calculated based on " +
		"a threshold placed on total Swap memory "
	return info
}

// Name returns the name of the Memory capacity component.
func (m *MemCap) Name() string {
	return m.name
}

// USEMetrics returns USEMetrics for the Memory capacity component.
func (m *MemCap) USEMetrics() *USEMetrics {
	return m.metrics
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
		return fmt.Errorf("missing output for %q", cmd)
	}
	memUsed, muPresent := parsedOutput["Mem:used"]
	if !muPresent {
		return fmt.Errorf("missing free's Mem row and used column")
	}

	memory := [][]string{memUsed}
	var used int
	for _, mem := range memory {
		sum, err := utils.SumAtoi(mem)
		if err != nil {
			return err
		}
		used += sum
	}
	// get total [main] memory
	total, err := m.calculateTotalMemory("Mem", outputs)
	if err != nil {
		return err
	}
	// get value as percentage and rount it off
	util := (float64(used) / float64(total)) * 100
	m.metrics.Utilization = math.Round((util)*1000) / 1000

	return nil
}

// calculateTotalMemory calculates the total main or swap memory on the system,
// depending on what string passed in: "Mem" or "Swap". If "Mem" is passed in,
// it returns the value found on free's "Mem" row + "total" column and if "Swap"
// is passed in, it returns the value on free's "Swap" row + "total" column.
func (m *MemCap) calculateTotalMemory(mem string, outputs map[string]utils.ParsedOutput) (int, error) {
	cmd := "free"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return 0, fmt.Errorf("missing output for %q", cmd)
	}
	memType := mem + ":total"
	memTotal, mtPresent := parsedOutput[memType]
	if !mtPresent {
		return 0, fmt.Errorf("missing free's %s row and total column", mem)
	}

	total, err := utils.SumAtoi(memTotal)
	if err != nil {
		return 0, err
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
	vmstatCmd := "vmstat"
	parsedOutput, ok := outputs[vmstatCmd]
	if !ok {
		return fmt.Errorf("missing output for %q", vmstatCmd)
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
	// get total [Swap] memory
	total, err := m.calculateTotalMemory("Swap", outputs)
	if err != nil {
		return err
	}
	// since metrics from free are in megabytes and those from vmstat are
	// in kilobytes
	totalBytes := total * 1024

	// ten percent of total swap memory
	log.Infof("swaps is %d and total swap memory is %d", swaps, totalBytes)

	var threshold float64
	// accounts for cases where swap memory is 0
	if totalBytes == 0 {
		// threshold set as 95 percent utilization
		threshold = 95
		m.metrics.Saturation = m.metrics.Utilization > threshold
	} else {
		// threshold set as 10 percent of total swap memory
		threshold = 0.1 * float64(totalBytes)
		m.metrics.Saturation = float64(swaps) > threshold
	}
	return nil
}

// CollectErrors collects errors for the MemCap component.
func (m *MemCap) CollectErrors(outputs map[string]utils.ParsedOutput) error {
	// Not yet implemented.
	return nil
}

// StorageDevIO holds information about the Storage device I/O component:
// name and USE Metrics collected.
type StorageDevIO struct {
	name    string
	metrics *USEMetrics
}

// NewStorageDevIO holds information about the Storage device I/O component:
// this can be used to initialize Storage device I/O outside of the
// profiler package.
func NewStorageDevIO(name string) *StorageDevIO {

	return &StorageDevIO{
		name:    name,
		metrics: &USEMetrics{},
	}
}

// AdditionalInformation returns additional information unique to the
// the StorageDevIO component.
func (d *StorageDevIO) AdditionalInformation() string {
	return ""

}

// Name returns the name of the Storage device I/O component.
func (d *StorageDevIO) Name() string {
	return d.name
}

// USEMetrics returns USEMetrics for the Storage Device I/O Component.
func (d *StorageDevIO) USEMetrics() *USEMetrics {
	return d.metrics
}

// CollectUtilization collects the utilization score for the StorageDevIO component.
// It does this by getting the percentage of elapsed time during which I/O requests
// were issued to the devices. This value can be found on iostat's '%util' column.
func (d *StorageDevIO) CollectUtilization(outputs map[string]utils.ParsedOutput) error {
	cmd := "iostat"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return fmt.Errorf("missing output for %q", cmd)
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
		return fmt.Errorf("missig output for %q", cmd)
	}
	queue, ok := parsedOutput["aqu-sz"]
	if !ok {
		return fmt.Errorf("missing iostat column 'aqu-sz'")
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

// StorageCap holds information about the Storage Capacity component:
// name and USE Metrics collected.
type StorageCap struct {
	name    string
	metrics *USEMetrics
	devices []string
}

// NewStorageCap holds information about the StorageCap component:
// this can be used to initialize StorageCap outside of the
// profiler package.
func NewStorageCap(name string) *StorageCap {

	return &StorageCap{
		name:    name,
		metrics: &USEMetrics{},
		devices: []string{},
	}
}

// AdditionalInformation returns additional information unique to the
// the StorageCap component.
func (s *StorageCap) AdditionalInformation() string {
	info := "The utilization value for this component was measured using the " +
		"following devices: " + strings.Join(s.devices, ",")
	return info
}

// sets the boot disk as default if no devices are specified
func (s *StorageCap) setDefaults() {
	if len(s.devices) == 0 {
		s.devices = []string{"/dev/sda"}
	}
}

// CollectUtilization calculates the utilization value for Storage Capacity.
// It does this by getting disk usage of particular devices on the file system.
// Disk usage on a particular device can be found using the 'df' command by
// getting the 'Used' value of that device divided by its total size, found
// on the column specifying metrics of block size. In this case, this column is
// "1K-blocks", since "-k" was passed as a flag to 'df'. The devices to collect
// disk usage for are found on StorageCap's devices field. If this field is not
// set, "/dev/sda", i.e. the boot disk, is used as default.
func (s *StorageCap) CollectUtilization(outputs map[string]utils.ParsedOutput) error {
	// if devices are not set
	s.setDefaults()

	dfCmd := "df"
	parsedOutput, ok := outputs[dfCmd]
	if !ok {
		return fmt.Errorf("missing output for %q", dfCmd)
	}
	usedBlocks, uPresent := parsedOutput["Used"]
	if !uPresent {
		return fmt.Errorf("missing df column 'Used'")
	}
	// total column is represented by the column displaying metrics of block size,
	// in this case "1K-blocks"
	totalBlocks, tPresent := parsedOutput["1K-blocks"]
	if !tPresent {
		return fmt.Errorf("missing df column '1K-blocks'")
	}
	fsystems, fsPresent := parsedOutput["Filesystem"]
	if !fsPresent {
		return fmt.Errorf("missing column 'Filesystem'")
	}
	// loop over all devices, if a device was specified by the struct,
	// get its index and use that to find its "Used" and "total" values
	var fUsed int
	var fSize int
	hasDevice := make([]bool, len(s.devices))
	for index, fsystem := range fsystems {
		for i, device := range s.devices {
			if strings.HasPrefix(fsystem, device) {
				// keep track of valid devices to collect statitics from
				hasDevice[i] = true
				s := usedBlocks[index]
				val, err := strconv.Atoi(s)
				if err != nil {
					return fmt.Errorf("failed to convert %q to int: %v", val, err)
				}
				fUsed += val

				s = totalBlocks[index]
				val, err = strconv.Atoi(s)
				if err != nil {
					return fmt.Errorf("failed to convert %q to int: %v", val, err)
				}
				fSize += val
			}
		}
	}
	// check if there are missing devices
	for i, ok := range hasDevice {
		if !ok {
			return fmt.Errorf("failed to find the device %q", s.devices[i])
		}
	}
	utiil := (float64(fUsed) / float64(fSize)) * 100
	fsUtilization := math.Round((utiil)*100) / 100

	s.metrics.Utilization = fsUtilization
	return nil
}

// CollectSaturation collects the saturation value for Storage Capacity.
func (s *StorageCap) CollectSaturation(outputs map[string]utils.ParsedOutput) error {
	// Not yet implemented
	return nil
}

// CollectErrors collects errors for the Storage Capacity component.
func (s *StorageCap) CollectErrors(outputs map[string]utils.ParsedOutput) error {
	// Not yet implemented
	return nil
}

func (s *StorageCap) USEMetrics() *USEMetrics {
	return s.metrics
}

func (s *StorageCap) Name() string {
	return s.name
}

// CollectUSEMetrics collects USE Metrics for the component specified. It does this by calling
// the necessary methods to collect utilization, saturation and errors.
func CollectUSEMetrics(component Component, outputs map[string]utils.ParsedOutput) error {

	metrics := component.USEMetrics()
	metrics.Timestamp = time.Now()
	start := metrics.Timestamp

	var gotErr bool
	if err := component.CollectUtilization(outputs); err != nil {
		gotErr = true
		log.Errorf("failed to collect utilization for %q: %v", component.Name(), err)
	}
	if err := component.CollectSaturation(outputs); err != nil {
		gotErr = true
		log.Errorf("failed to collect saturation for %q: %v", component.Name(), err)
	}
	end := time.Now()
	metrics.Interval = end.Sub(start)

	if gotErr {
		err := "failed to collect all USE metrics for %q. " +
			"Please check the logs for more information"
		return fmt.Errorf(err, component.Name())
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
		if err := CollectUSEMetrics(s, outputs); err != nil {
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
