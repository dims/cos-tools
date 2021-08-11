package profiler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/utils"

	log "github.com/sirupsen/logrus"
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
		log.Info("delay cannot be 0. Value set to 1")
		v.delay = 1
	}
	if v.count == 0 {
		log.Info("count of 0 gives averages since reboot. Count set to 5")
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

	args := []string{"-n", interval, count}
	out, err := utils.RunCommand(v.Name(), args...)
	if err != nil {
		str := v.Name() + " " + strings.Join(args, " ")
		return nil, fmt.Errorf("failed to run the command %q: %v",
			str, err)
	}

	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	// ignore the first row in vmstat's output
	lines = lines[1:]
	// split first row into columns based on titles
	allTitles := strings.Fields(lines[0])
	// parse output by columns
	output, err := utils.ParseColumns(lines, allTitles, v.titles...)
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
		return nil, fmt.Errorf("failed to run the command '%s': %v", l.Name(), err)
	}
	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	// parse output by rows
	output, err := utils.ParseRows(lines, ":", l.titles...)
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
		cmd := f.Name() + " " + "-m"
		return nil, fmt.Errorf("failed to run the command %q: %v",
			cmd, err)
	}

	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	// parse output by rows and columns
	output, err := utils.ParseRowsAndColumns(lines, f.titles...)

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

	args := []string{i.flags, interval, count}
	out, err := utils.RunCommand(i.Name(), args...)
	if err != nil {
		str := i.Name() + " " + strings.Join(args, " ")
		return nil, fmt.Errorf("failed to run the command %q: %v",
			str, err)
	}
	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("failed to collect output of iostat, rows: %v", lines)
	}
	// ignore the first 2 lines in iostat's output so that the first line
	// is column titles.
	lines = lines[2:]

	// if the "-y" flag was specified, there will be an extra new line added
	// to indicate that the first report was ignored
	if first := strings.Fields(lines[0]); len(first) == 0 {
		lines = lines[1:]
	}

	// split first row into columns based on titles
	allTitles := strings.Fields(lines[0])
	// parse output by rows and columns
	output, err := utils.ParseColumns(lines, allTitles, i.titles...)
	return output, err
}

// df represents the command 'df'
type df struct {
	name   string
	flags  string
	titles []string
}

// NewDF function helps to initialize a df structure.
func NewDF(name string, flags string, titles []string) *df {
	return &df{
		name:   name,
		flags:  flags,
		titles: titles,
	}
}

// Name returns the name for the 'df' command
func (fs *df) Name() string {
	return fs.name
}

// Run executes the 'df' command, parses its output and returns a
// map of title(s) to their values.
func (fs *df) Run() (map[string][]string, error) {
	// get output in 1K size to make summing values direct
	out, err := utils.RunCommand(fs.Name(), fs.flags)
	if err != nil {
		cmd := fs.Name() + " " + fs.flags
		return nil, fmt.Errorf("failed to run the command %q: %v",
			cmd, err)
	}
	s := string(out)
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	// match all strings that start with an uppercase letter. This
	// pattern makes it possible to split df's titles row since some
	// titles are multi-worded (as seen with "Mounted on" below) so
	// splitting by whitespaces will result in incorrect titles slice.
	//
	// "Filesystem      Size  Used Avail Use% Mounted on" ->
	// ["Filesystem", "Size", "Used", "Avail", "Use%", "Mounted on"]
	re := regexp.MustCompile(`[0-9]?[A-Z]{1}[^A-Z0-9]*`)
	allTitles := re.FindAllString(lines[0], -1)
	// trim trailing or leading white spaces in all titles.
	allTitles = utils.TrimCharacter(allTitles, " ")
	// parse output by columns
	output, err := utils.ParseColumns(lines, allTitles, fs.titles...)
	return output, err
}
