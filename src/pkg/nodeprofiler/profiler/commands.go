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
	Run() (map[string][]string, error)
}

// vmstat represents a vmstat command.
type vmstat struct {
	name string
	// delay specifies times between updates in seconds.
	delay int
	// count specifies number of updates.
	count int
	// titles specifies the titles to get values for.
	titles []string
}

// NewVMStat function helps to initialize a vmstat structure.
func NewVMStat(name string, delay int, count int, titles []string) *vmstat {
	return &vmstat{
		name:   name,
		delay:  delay,
		count:  count,
		titles: titles,
	}
}

// Name returns the name for vmstat command.
func (v *vmstat) Name() string {
	return v.name
}

func (v *vmstat) setDefaults() {
	if v.delay == 0 {
		v.delay = 1
	}
	if v.count == 0 {
		v.count = 5
	}
}

// Run executes the vmstat command, parses the output and returns it as
// a map of titles to their values.
func (v *vmstat) Run() (map[string][]string, error) {
	// if delay and count not set
	v.setDefaults()
	interval := strconv.Itoa(v.delay)
	count := strconv.Itoa(v.count)
	out, err := utils.RunCommand(v.Name(), "-n", interval, count)
	if err != nil {
		return nil, fmt.Errorf("failed to run the command 'vmstat': %v", err)
	}

	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	// ignore the first row in vmstat's output
	lines = lines[1:]
	// split first row into columns based on titles
	allTitles := strings.Fields(lines[0])
	wantTitles := v.titles
	// parse output by columns
	output, err := utils.ParseColumns(lines, allTitles, wantTitles...)
	return output, err
}

// lscpu represents an lscpu command.
type lscpu struct {
	name string
	// titles specifies the titles to get values for.
	titles []string
}

// NewLscpu function helps to initialize a lscpu structure.
func NewLscpu(name string, titles []string) *lscpu {
	return &lscpu{
		name:   name,
		titles: titles,
	}
}

// Name returns the name for the lscpu command.
func (l *lscpu) Name() string {
	return l.name
}

// Run executes the lscpu command, parses the output and returns a
// a map of title(s) to their values.
func (l *lscpu) Run() (map[string][]string, error) {
	out, err := utils.RunCommand(l.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to run the command 'lscpu': %v", err)
	}
	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	titles := l.titles
	// parse output by rows
	output, err := utils.ParseRows(lines, ":", titles...)
	return output, err
}

// free represents a free command.
type free struct {
	name string
	// titles specifies the titles to get values for.
	titles []string
}

// NewFree function helps to initialize a free structure.
func NewFree(name string, titles []string) *free {
	return &free{
		name:   name,
		titles: titles,
	}
}

// Name returns the name for the free command.
func (f *free) Name() string {
	return f.name
}

// Run executes the free commands, parses the output and returns a
// a map of title(s) to their values.
func (f *free) Run() (map[string][]string, error) {
	out, err := utils.RunCommand(f.Name(), "-m")
	if err != nil {
		return nil, fmt.Errorf("failed to run the command 'free': %v", err)
	}

	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	titles := f.titles
	// parse output by rows and columns
	output, err := utils.ParseRowsAndColumns(lines, titles...)

	return output, err
}

// iostat represents an iostat command
type iostat struct {
	name string
	// flags specify the flags to be passed into the command.
	flags string
	// delay specifies times between updates in seconds.
	delay int
	// count specifies number of updates.
	count int
	// titles specifies the titles to get values for.
	titles []string
}

// NewIOStat function helps to initialize a iostat structure.
func NewIOStat(name string, flags string, delay int, count int, titles []string) *iostat {
	return &iostat{
		name:   name,
		flags:  flags,
		delay:  delay,
		count:  count,
		titles: titles,
	}
}

// Name returns the name for the iostat command.
func (i *iostat) Name() string {
	return i.name
}

func (i *iostat) setDefaults() {
	if i.delay == 0 {
		i.delay = 1
	}
	if i.count == 0 {
		i.count = 5
	}
}

// Run executes the iostat commands, parses the output and returns a
// a map of title(s) to their values.
func (i *iostat) Run() (map[string][]string, error) {
	// if delay and count not set
	i.setDefaults()
	interval := strconv.Itoa(i.delay)
	count := strconv.Itoa(i.count)
	out, err := utils.RunCommand(i.Name(), i.flags, interval, count)
	if err != nil {
		return nil, fmt.Errorf("failed to run the command 'iostat': %v", err)
	}
	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	// ignore the first 2 lines in iostat's output so that the first line
	// is column titles.
	lines = lines[2:]

	// split first row into columns based on titles
	allTitles := strings.Fields(lines[0])
	wantTitles := i.titles
	// parse output by rows and columns
	output, err := utils.ParseColumns(lines, allTitles, wantTitles...)
	return output, err
}
