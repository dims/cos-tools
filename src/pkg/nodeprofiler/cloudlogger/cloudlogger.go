// Package cloudlogger provides functionality to log text or json data to
// Google Cloud logging backend.
package cloudlogger

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/profiler"

	log "github.com/sirupsen/logrus"
)

const defaultCommandTimeout = 300 * time.Second

// componentInfo contains Name and Metrics fields similar to Name and Metrics
// fields that each profiler component has. componentInfo helps to export
// component fields to log them to Google Cloud Logging backend.
type componentInfo struct {
	Name    string
	Metrics *profiler.USEMetrics
}

// LoggerOpts contains the options supported when logging the Profiler Report
// to Google Cloud Logging backend.
type LoggerOpts struct {
	// Specifies the project ID to write logs to.
	ProjID string
	// Specifies raw commands for which to log output.
	Command string
	// Specifies the number of times to run an arbitrary shell command.
	CmdCount int
	// Specifies the interval separating the number of times the user runs
	// an arbitrary shell command.
	CmdInterval time.Duration
	// Specifies the amount of time it will take for the a raw shell command to
	// timeout.
	CmdTimeOut time.Duration
	// Specifies the number of times to run the profiler.
	ProfilerCount int
	// Specifies the interval the profiler will run.
	ProfilerInterval time.Duration
	// Components on which to run profiler. It may contain CPU(s), Memory, etc.
	Components []profiler.Component
	// ProfilerCmds field specifies additional options needed to run the profiler
	ProfilerCmds []profiler.Command
}

// TextLogger defines the method required to log a text string to Google Cloud
// logging backend.
type TextLogger interface {
	Printf(text string, a ...interface{})
}

// StructuredLogger defines the method required to log anything that can be
// marshaled to JSON to Google Cloud logging backend and the method that blocks
// buffered log entries until previous log entries are sent to Google Cloud
// logging backend.
type StructuredLogger interface {
	Log(l logging.Entry)
	Flush() error
}

// Validate ensures that options are correctly set. If they aren't, the method
// returns a list of error messages encountered. To log the output of a shell
// command, the user must specify both the command, command count and interval
// configurations which specify the command, how often to log its output and the
// interval in seconds. By default, the commands will timeout after 300 seconds
// unless the user specified a different value using the commandTimeout
// configuration. Similarly, to run the profiler, the user must specify a
// ProfilerCount configuration if a ProfilerInterval configuration was specified.
func (l *LoggerOpts) Validate() error {
	// A valid project ID must be provided.
	if l.ProjID == "" {
		return fmt.Errorf("invalid Logger options: cannot run profiler tool if the Cloud Logging Project ID is not set")
	}
	// To run the profiler, the profilerCount configuration has to be set if
	// profilerInterval configuration is set.
	if l.ProfilerCount == 0 && l.ProfilerInterval != 0 {
		return fmt.Errorf("invalid Logger options: cannot set ProfilerInterval if ProfilerCount is not set")
	}
	// if the command configuration is nil, ensure CmdCount, CmdInterval, and
	// CmdTimeOut configurations are all 0.
	if l.Command == "" {
		if l.CmdCount != 0 || l.CmdInterval != 0 || l.CmdTimeOut != 0 {
			return fmt.Errorf("invalid Logger options: CmdCount, CmdInterval and CmdTimeout should not be set if Command is not set")
		}
	} else {
		// if the the command configuration was set, but the user did not specify
		// when the command will timeout, then the CmdTimeOut configuration will be
		// set to defaultCommandTimeout, which is 300 seconds.
		if l.CmdTimeOut == 0 {
			l.CmdTimeOut = defaultCommandTimeout
		}
		// if CmdCount config is not set, then CmdInterval must not be set.
		if l.CmdCount == 0 && l.CmdInterval != 0 {
			return fmt.Errorf("invalid Logger options: cannot set CmdInterval if CmdCount is not set")
		}
		// if the command configuration is set but no other options is set, run the
		// command once.
		if l.CmdCount == 0 && l.CmdInterval == 0 {
			l.CmdCount = 1
		}
	}
	return nil
}

// checkLogError returns nil if there was no error encountered while running the profiler/shell
// commands. If there was one or more error, a summary of errors encountered will be returned.
func checkLogError(emptyCmd bool, errArr []error) error {
	if len(errArr) == 0 {
		return nil
	}
	if emptyCmd == true {
		return fmt.Errorf("encountered %v errors while running the profiler: %v", len(errArr), errArr)
	}
	return fmt.Errorf("encountered %v errors while running the profiler and shell commands: %v", len(errArr), errArr)
}

// LogText writes the string infoToLog to a logging backend by calling the
// `Printf` method defined by the TextLogger interface.
// To log some text to Google Cloud Logging backend , pass in an instance of
// type *log.Logger.
func LogText(g TextLogger, infoToLog string) error {
	if infoToLog == "" {
		return fmt.Errorf("cannot log an empty string to Cloud Logging")
	}
	g.Printf(infoToLog)
	return nil
}

// logShellCmd writes a struct containing the name of an arbitrary shell
// command and its output to a logging backend by calling the `Log` method
// defined by the StructuredLogger interface. To Log a JSON Payload to Google
// Cloud Logging backend, pass in an instance of type *logging.Logger.
func logShellCommand(g StructuredLogger, cmdTimeOut time.Duration, cmd string, args ...string) error {
	// fullCommand string includes a main command and its options.
	// For `ps -aux` the cmd is `ps` the options are `-aux` thus the
	// fullCommand `ps -aux`
	var fullCommand string
	// formating commandName to not include blank spaces.
	if len(args) == 0 {
		fullCommand = cmd
	} else {
		fullCommand = cmd + " " + strings.Join(args, " ")
	}
	// Timeout after cmdTimeOut seconds.
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeOut)
	defer cancel()
	out, err := exec.CommandContext(ctx, cmd, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		err := "command timed out"
		log.Error(err)
		return fmt.Errorf("cannot log %v to Cloud Logging: %v", fullCommand, err)
	}

	if err != nil {
		return fmt.Errorf("cannot run %v command: %v", fullCommand, err)
	}
	entry := logging.Entry{
		Payload: struct {
			CommandName   string
			CommandOutput string
		}{
			CommandName:   fullCommand,
			CommandOutput: string(out),
		},
		Severity: logging.Debug,
	}
	g.Log(entry)
	return nil
}

// logUSEReport writes a USEReport to a logging backend by calling the `Log`
// method defined by the StructuredLogger interface. To log a JSON Payload to
// Google Cloud Logging backend, pass in an instance of type *logging.Logger.
func logUSEReport(g StructuredLogger, useReport *profiler.USEReport) error {
	if useReport == nil {
		return fmt.Errorf("cannot log an empty USEReport")
	}
	var cInfos []componentInfo
	for _, c := range useReport.Components {
		cInfos = append(cInfos, componentInfo{Name: c.Name(), Metrics: c.USEMetrics()})
	}
	entry := logging.Entry{
		// Log anything that can be marshaled to JSON.
		Payload: struct {
			Components []componentInfo
			Analysis   string
		}{
			Components: cInfos,
			Analysis:   useReport.Analysis,
		},
		Severity: logging.Debug,
	}
	g.Log(entry)
	return nil
}

// LogProfilerReport logs the output of logShellCommand and logUSEReport
// functions to a logger of type *logging.Logger. It takes as input the logger
// itself, and a pointer to a LoggerOpts struct that contains the shell command
// to execute, the number of time to execute that shell command, the interval
// between shell command executions, the time limit for shell command execution,
// the number of time to run the profiler tool, the interval between the number
// of times to run the profiler, and a profiler option struct that specify the
// component to generate USEReport for as well as any options associated to that
// component.
func LogProfilerReport(g StructuredLogger, opts *LoggerOpts) error {
	var emptyCmd bool
	errArr := []error{}
	log.Info("Validating logger options . . .")
	if err := opts.Validate(); err != nil {
		return err
	}
	log.Info("Done validating logger options.")
	log.Info("Running Profiler . . .")
	// Ensure logging entries are written to the cloud logging backend.
	defer g.Flush()
	// Only log shell command if the user specified a command.
	if len(opts.Command) == 0 {
		emptyCmd = true
	} else {
		emptyCmd = false
		log.Info("Running shell command . . .")
		// Fetching command from user input and populating the cmdArray
		// with the main command and its flags.
		cmdArray := strings.Split(opts.Command, " ")
		usrMainCmd := cmdArray[0]
		usrMainCmdFlags := cmdArray[1:]
		for i := 0; i < opts.CmdCount; i++ {
			if err := logShellCommand(g, opts.CmdTimeOut, usrMainCmd, usrMainCmdFlags...); err != nil {
				errArr = append(errArr, err)
				continue
			}
			// Delaying execution by cmdInterval seconds.
			time.Sleep(opts.CmdInterval)
		}
		log.Infof("Done running shell command.")
	}
	// Run the profiler profCount times. The default value is 1 time unless user
	// set the counter to a different number.
	for i := 0; i < opts.ProfilerCount; i++ {
		useReport, err := profiler.GenerateUSEReport(opts.Components, opts.ProfilerCmds)
		if err != nil {
			errArr = append(errArr, fmt.Errorf("cannot run profiler.GenerateUSEReport(%v) = %v", opts.Components, err))
			continue
		}
		if err := logUSEReport(g, &useReport); err != nil {
			errArr = append(errArr, err)
			continue
		}
		// Delaying execution by profilerInterval seconds.
		time.Sleep(opts.ProfilerInterval)
	}
	log.Info("Done running profiler.")
	return checkLogError(emptyCmd, errArr)
}
