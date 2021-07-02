package profiler

import (
	"fmt"
	"strconv"
	"strings"

	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/utils"
)

// Command interface defines functions that can be implemented by
// structs to execute shell commands.
type Command interface {
	Name() string
	Run(opts Options) (map[string][]string, error)
}

// Options stores the options that can be passed to a command and
// to parsing functions.
type Options struct {
	// Delay specifies times between updates in seconds.
	Delay int
	// Count specifies number of updates.
	Count int
	// Titles specifies the titles to get values for.
	Titles []string
}

// vmstat represents a vmstat command.
type vmstat struct {
	name string
}

// lscpu represents an lscpu command.
type lscpu struct {
	name string
}

// Name returns the name for vmstat command.
func (v *vmstat) Name() string {
	return v.name
}

// Run executes the vmstat command, parses the output and returns it as
// a map of titles to their values.
func (v *vmstat) Run(opts Options) (map[string][]string, error) {
	// delay and count not set
	if opts.Delay == 0 {
		opts.Delay = 1
	}
	if opts.Count == 0 {
		opts.Count = 5
	}
	interval := strconv.Itoa(opts.Delay)
	count := strconv.Itoa(opts.Count)
	out, err := utils.RunCommand(v.Name(), "-n", interval, count)
	if err != nil {
		return nil, fmt.Errorf("failed to run vmstat command: %v", err)
	}

	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	// ignore the first row in vmstat's output
	lines = lines[1:]
	titles := opts.Titles
	// parse output by columns
	output, err := utils.ParseColumns(lines, titles...)
	return output, err

}

// Name returns the name for the lscpu command.
func (l *lscpu) Name() string {
	return l.name
}

// Run executes the lscpu command, parses the output and returns a
// a map of title(s) to their values.
func (l *lscpu) Run(opts Options) (map[string][]string, error) {
	out, err := utils.RunCommand(l.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to run vmstat command: %v", err)
	}
	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	titles := opts.Titles
	// parse output by rows
	output, err := utils.ParseRows(lines, titles...)
	return output, err
}
