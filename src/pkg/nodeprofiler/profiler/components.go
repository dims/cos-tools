package profiler

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/utils"
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

func sumAtoi(a []string) (int, error) {
	var sum int
	for _, str := range a {
		val, err := strconv.Atoi(str)
		if err != nil {
			return 0, fmt.Errorf("could not convert %s to an int: %v", str, err)
		}
		sum += val
	}
	return sum, nil
}

// CollectUtilization calculates the utilization score for the CPU Component.
func (c *CPU) CollectUtilization(outputs map[string]utils.ParsedOutput) error {
	cmd := "vmstat"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return fmt.Errorf("missing output for vmstat")
	}
	us, usPresent := parsedOutput["us"]
	sy, syPresent := parsedOutput["sy"]
	st, stPresent := parsedOutput["st"]
	if !usPresent || !syPresent || !stPresent {
		return fmt.Errorf("missing some vmstat columns")
	}
	columns := [][]string{us, sy, st}
	var total int
	// loop over us, sy, st columns and sum their values
	for _, column := range columns {
		sum, err := sumAtoi(column)
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
func calculateCPUCount(outputs map[string]utils.ParsedOutput) (int, error) {
	cmd := "lscpu"
	parsedOutput, ok := outputs[cmd]
	if !ok {
		return 0, fmt.Errorf("missing output for lscpu")
	}
	val := parsedOutput["CPU(s):"]
	count, err := strconv.Atoi(val[0])
	if err != nil {
		return 0, fmt.Errorf("could not convert %s to an int: %v", val[0], err)
	}
	return count, nil
}

// CollectSaturation calcualates the saturation value for the CPU component.
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
	sum, err := sumAtoi(running)
	if err != nil {
		return err
	}

	num := len(running)
	runningProcs := sum / num
	count, err := calculateCPUCount(outputs)
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
	if err := c.CollectUtilization(outputs); err != nil {
		return fmt.Errorf("failed to collect utilization score: %v", err)
	}
	if err := c.CollectSaturation(outputs); err != nil {
		return fmt.Errorf("failed to collect saturation score: %v", err)
	}
	end := time.Now()
	metrics.Interval = end.Sub(start)
	return nil
}

// GenerateUSEReport generates USE Metrics for all the components
// as well as an analysis string to help the diagnose performance issues.
func GenerateUSEReport(components []Component, opts Options) (USEReport, error) {
	useReport := USEReport{Components: components}
	cmds := []Command{&vmstat{"vmstat"}, &lscpu{"lscpu"}}
	outputs := make(map[string]utils.ParsedOutput)

	for _, cmd := range cmds {
		output, err := cmd.Run(opts)
		if err != nil {
			return useReport, fmt.Errorf("failed to run %s command: %v", cmd.Name(), err)
		}
		name := cmd.Name()
		outputs[name] = output
	}
	for _, s := range components {
		s.CollectUSEMetrics(outputs)
	}
	return useReport, nil
}
