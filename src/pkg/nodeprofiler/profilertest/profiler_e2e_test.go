package profilertest

import (
	"bufio"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/profiler"
)

// TestCPUOverload tests whether package profiler is able to collect USEMetrics
// for the CPU component. It does this by overloading the CPU component using the
// "stress-ng" package upto a certain threshold and checking whether this was
// reflected in the component's metrics. The flags specificed to the shell command
// "stress-ng" include:
//        --cpu N: specifies the component the stress test will be applied on - the
//                 CPU and specifcally, N number of cores
//        --cpu-load P: load CPU with P percent loading for the stress workers to
//                 set an approximate threshold on expetected utilization
//		  --fork N: continually fork child processes that exit to increase wait time
//                 for processes and thus make saturation true
//        -v: (verbose) show all debug, warnings and normal information output
// //        -t N: stop stress after N units of time (also specified in N)
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

// TestStorageDevOverload tests whether package profiler is able to collect USEMEtrics
// for the StorageDevIO component. It does this by overloading the storage devices I/O
// using the "stress-ng" package. It then checks that this was reflected in the USEMetrics
// collected. The flags specified to the shell command "stress-ng" include:
//        --iomix N: start N workers that will perform a mix of I/O operations
//        --iomix-bytes N: write N bytes for each iomix worker process. In this case N is
//                         specified as a percemtage of the free space on the file system
//        -t N: stop stress after N units of time (units also specified in N)
func TestStorageDevOverload(t *testing.T) {

	// initialize all commands needed and the mem cap component
	titles := []string{"%util", "aqu-sz"}
	iostat := profiler.NewIOStat("iostat", "-dxyz", 1, 10, titles)

	commands := []profiler.Command{iostat}

	dev := profiler.NewStorageDevIO("StorageDevIO")
	components := []profiler.Component{dev}

	// stress test will run for 1 minute perfoming a number of I/O operations
	// that will take up 80% of free space on  the file system
	args := []string{"--iomix", "1", "--iomix-bytes", "80%", "-t", "1m"}
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
	t.Logf("USE Report generated:\n %+v", report.Components[0].USEMetrics())
	if err != nil {
		t.Errorf("failed to generate USE report for StorageDevIO component, %v", err)
	}
	if utilization := dev.USEMetrics().Utilization; utilization < 80 {
		err := "overloaded the StorageDevIO component but utilization was less that 80: %v"
		t.Errorf(err, utilization)
	}
	if saturated := dev.USEMetrics().Saturation; !saturated {
		err := "overloaded the StorageDevIO component but saturation was false"
		t.Errorf(err)
	}
	// wait for command to exit and release any resources associated with it.
	if err = cmd.Wait(); err != nil {
		t.Errorf("failed to finish running the command %q: %v", str, err)
	}
	t.Logf("finished running %q command successfully", str)
}

// TestMemOverload tests whether package profiler is able to collect USE Metrics
// for the MemCap component. It does this by overloading the MemCap component using
// the "stress-ng" package upto a certain threshold and checking whether this was reflected
// in the component's metrics. The flags specified to the shell command "stress-ng" include:
//        --vm-bytes N: allocate N bytes for use by the memory stressors. In this case N is
//                      specified as a percentage of total available memory
//        --vm N: starts N workers that will write to the allocated memory
//        --brk N: starts N workers that grow the data segment by one page at
//                 a time using mulitple brk calls. This stresses swapping
//        --bigheap N: starts N workers that grow their heaps by reallocating memory
//                      stressing both memory and swapping
//        -t N: stop stress after N units of time (units also specified in N)
func TestMemOverload(t *testing.T) {
	// initialize all commands needed and the mem cap component
	titles := []string{"Mem:used", "Mem:total", "Swap:used", "Swap:total"}
	free := profiler.NewFree("free", titles)

	titles = []string{"si", "so"}
	vmstat := profiler.NewVMStat("vmstat", 1, 75, titles)

	commands := []profiler.Command{vmstat, free}

	mem := profiler.NewMemCap("MemCap")
	components := []profiler.Component{mem}

	// stress test will run for 2 minutes writing to the allocated memory (90%)
	// and growing the data segment as well as the heap
	args := []string{"--vm-bytes", "90%", "-vm", "1", "--brk", "2", "--bigheap", "2", "-t", "2m"}
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
	// allow portion of memory to be written
	t.Log("Sleep for 30s to allow memory to be overloaded")
	time.Sleep(30 * time.Second)

	// generates USE report while stress test is running
	report, err := profiler.GenerateUSEReport(components, commands)
	t.Logf("USE Report generated:\n %+v", report.Components[0].USEMetrics())
	if err != nil {
		t.Errorf("failed to generate USE report for MemCap component, %v", err)
	}
	if utilization := mem.USEMetrics().Utilization; utilization < 90 {
		err := "overloaded the MemCap component but utilization was less that 90: %v"
		t.Errorf(err, utilization)
	}
	if saturated := mem.USEMetrics().Saturation; !saturated {
		err := "overloaded the MemCap component but saturation was false"
		t.Errorf(err)
	}
	// wait for command to exit and release any resources associated with it.
	if err = cmd.Wait(); err != nil {
		t.Errorf("failed to finish running the command %q: %v", str, err)
	}
	t.Logf("finished running %q command successfully", str)
}
