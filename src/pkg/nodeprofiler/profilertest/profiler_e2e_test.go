package profilertest

import (
	"bufio"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/profiler"
)

// TestCPUOverload tests whether the CPU's CollectUSEMetrics in package profiler
// is working or not. It does this by overloading the CPU component using the
// "stress-ng" package upto a certain threshold and checking whether this was
// reflected in the component's metrics. The flags specificed to the shell command
// "stress-ng" include:
//        --cpu N: specifies the computer system the stress test is will be applied
//                 on - the CPU and specifcally, N number of cores
//        --cpu-load P: load CPU with P percent loading for the stress workers to
//                 set an approximate threshold on expetected utilization
//		  --fork N: continually fork child processes that exit to increase wait time
//                 for processes and thus make saturation true
//        -v: (verbose) show all debug, warnings and normal information output
//        -t N: stop stress after N units of time (also specified in N)
func TestCPUOverload(t *testing.T) {
	// initialize all commands needed and the cpu component
	titles := []string{"r", "us", "sy", "st"}
	vmstat := profiler.NewVMStat("vmstat", 1, 10, titles)

	titles = []string{"CPU(s)"}
	lscpu := profiler.NewLscpu("lscpu", titles)

	commands := []profiler.Command{vmstat, lscpu}

	cpu := profiler.NewCPU("CPU")
	components := []profiler.Component{cpu}
	// get number of cores in CPU
	n := runtime.NumCPU()
	// number of processes should be more than number of cores to make CPU saturated
	processes := n + 4

	// The stress test will be run for 1 minute, overloading the CPU cores upto 92% and creating
	// a number of dummy processes that will make the component busy.
	args := []string{"--cpu", strconv.Itoa(n), "--cpu-load",
		"92", "--fork", strconv.Itoa(processes), "-v", "-t", "60s"}
	cmd := exec.Command("stress-ng", args...)

	// get pipe that will be connected to command's standard output when comamnd starts.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Errorf("failed to connect to command's standard output: %v", err)
	}
	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	str := "stress-ng" + " " + strings.Join(args, " ")
	t.Logf("running %q", str)
	// starts the command but does not wait for it to complete.
	if err := cmd.Start(); err != nil {
		t.Errorf("failed to start the command %q: %v", str, err)
	}
	scanner := bufio.NewScanner(stdout)

	// print to stdout in real time
	go func() {
		for scanner.Scan() {
			m := scanner.Text()
			t.Log(m)
		}
	}()
	// generates USE report while stress test is running
	report, err := profiler.GenerateUSEReport(components, commands)
	t.Logf("USE Report generated for CPU :\n %+v", report.Components[0].USEMetrics())
	if err != nil {
		t.Errorf("failed to generate USE report for CPU component, %v", err)
	}

	if utilization := cpu.USEMetrics().Utilization; utilization < 90 {
		err := "overloaded the CPU upto 90 percent but utilization was less that 90: %v"
		t.Errorf(err, utilization)
	}
	if saturated := cpu.USEMetrics().Saturation; !saturated {
		err := "overloaded cpu cores with stress test processes but saturation was false"
		t.Errorf(err)
	}

	// wait for command to exit and release any resources associated with it.
	if err = cmd.Wait(); err != nil {
		t.Errorf("failed to finish running the command %q: %v", str, err)
	}
	t.Logf("finished running %q command successfully", str)
}
