// Package main is the entry point for the cos_node_profiler application that
// imports the cloudlogger, profiler, and utils packages that respectively
// write logs to Google Cloud Logging backend, fetch debugging information from
// a Linux system and provide the interface between cloudlogger and profiler
// packages.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/cloudlogger"
	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/profiler"

	log "github.com/sirupsen/logrus"
)

const cloudLoggerName = "cos_node_profiler"

var (
	configFile       = flag.String("config-file", "", "specifies the path of the configuration file. If path is not set, then it is assumed that command line flags will be passed to configure the Node Profiler.")
	projID           = flag.String("project", "", "specifies the GCP project where logs will be added.")
	command          = flag.String("cmd", "", "specifies raw commands for which to log output.")
	cmdCount         = flag.Int("cmd-count", 0, "specifies the number of times to run an arbitrary shell command.")
	cmdInterval      = flag.Int("cmd-interval", 0, "specifies the interval (in seconds) separating the number of times the user runs an arbitrary shell command.")
	cmdTimeOut       = flag.Int("cmd-timeout", 300, "specifies the amount of time (in seconds) it will take for the a raw command to timeout and be killed.")
	profilerCount    = flag.Int("profiler-count", 1, "specifies the number of times to collect USE Report.")
	profilerInterval = flag.Int("profiler-interval", 0, "specifies the interval (in seconds) separating the number of times the user collects USE Report.")
)

func main() {
	var opts *cloudlogger.LoggerOpts
	var err error
	flag.Parse()
	if *configFile != "" {
		opts, err = loadConfig(*configFile)
		if err != nil {
			log.Fatalf("%v", err)
		}
	} else {
		opts = loadFlags()
	}
	// [START client setup]
	ctx := context.Background()
	client, err := logging.NewClient(ctx, opts.ProjID)
	if err != nil {
		log.Fatalf("failed to create logging client: %v", err)
	}
	defer client.Close()
	client.OnError = func(err error) {
		// Log an error to the local log if any function call failed.
		// For example, print an error if Flush() failed.
		log.Errorf("client.OnError: %v", err)
	}
	// [END client setup]
	log.Info("Begin logging profiler report...")
	logger := client.Logger(cloudLoggerName)
	if err = cloudlogger.LogProfilerReport(logger, opts); err != nil {
		log.Fatalf("%v", err)
	}
	log.Info("Successfully logged profiler report.")
}

// generateProfilerOpts is a helper function used to generate the components
// array as well as the profiler options used to call the
// profiler.GenerateUSEReport function from the profiler package.
func generateProfilerOpts() ([]profiler.Component, []profiler.Command) {
	// [Begin generating ProfilerOpts from Profiler Package]
	// Getting Components
	cpu := profiler.NewCPU("CPU")
	memcap := profiler.NewMemCap("MemCap")
	sDevIO := profiler.NewStorageDevIO("StorageDevIO")
	sCap := profiler.NewStorageCap("StorageCap")
	components := []profiler.Component{cpu, memcap, sDevIO, sCap}
	// End Getting Components
	// Getting Commands
	vmstat := profiler.NewVMStat("vmstat", 1, 5, []string{"us", "sy", "st", "si", "so", "r"})
	lscpu := profiler.NewLscpu("lscpu", []string{"CPU(s)"})
	free := profiler.NewFree("free", []string{"Mem:used", "Mem:total", "Swap:used", "Swap:total"})
	iostat := profiler.NewIOStat("iostat", "-xdz", 1, 5, []string{"aqu-sz", "%util"})
	df := profiler.NewDF("df", "-k", []string{})
	commands := []profiler.Command{vmstat, lscpu, free, iostat, df}
	// End Getting Commands
	// [End generating ProfilerOpts from Profiler Package]
	return components, commands
}

// loadflags helps to use command line flags as configuration to the Node
// Profiler tool.
func loadFlags() *cloudlogger.LoggerOpts {
	// Getting Profiler Options.
	components, commands := generateProfilerOpts()
	shCmds := []cloudlogger.ShellCmdOpts{
		cloudlogger.ShellCmdOpts{
			Command:     *command,
			CmdCount:    *cmdCount,
			CmdInterval: time.Duration(*cmdInterval) * time.Second,
			CmdTimeOut:  time.Duration(*cmdTimeOut) * time.Second,
		},
	}
	// populating LoggerOpts struct with configurations from user.
	opts := &cloudlogger.LoggerOpts{
		ProjID:           *projID,
		ShCmds:           shCmds,
		ProfilerCount:    *profilerCount,
		ProfilerInterval: time.Duration(*profilerInterval) * time.Second,
		Components:       components,
		ProfilerCmds:     commands,
	}
	return opts
}

// loadConfig helps to use a json configuration file in the current directory
// as configuration to the Node Profiler Tool.
func loadConfig(filename string) (*cloudlogger.LoggerOpts, error) {
	var logger cloudlogger.LoggerOpts
	configFile, err := os.Open(filename)
	defer configFile.Close()
	if err != nil {
		return &logger, fmt.Errorf("failed to open config file %v: %v", filename, err)
	}
	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&logger); err != nil {
		return &logger, fmt.Errorf("failed to parse config file %v: %v", filename, err)
	}
	for i := 0; i < len(logger.ShCmds); i++ {
		logger.ShCmds[i].CmdInterval = logger.ShCmds[i].CmdInterval * time.Second
		logger.ShCmds[i].CmdTimeOut = logger.ShCmds[i].CmdTimeOut * time.Second
	}
	logger.ProfilerInterval = logger.ProfilerInterval * time.Second
	components, commands := generateProfilerOpts()
	logger.Components = components
	logger.ProfilerCmds = commands
	return &logger, err
}
