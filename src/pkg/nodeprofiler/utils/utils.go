// Package utils defines parsing and run command functions
// that can be used outside nodeprofiler.
package utils

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// RunCommand is a wrapper function for exec.Command that will run the command
// specified return its output and/or error.
func RunCommand(cmd string, args ...string) ([]byte, error) {
	str := cmd + " " + strings.Join(args, " ")
	log.Infof("running %q", str)
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run %q: %v", str, err)
	}
	log.Infof("finished running %q command successfully", str)
	return out, nil
}

// SumAtoi converts all the strings in a slice to integers, sums them up and returns
// the result. A non-nil error is returned if an error occurred.
func SumAtoi(a []string) (int, error) {
	var sum int
	for _, str := range a {
		val, err := strconv.Atoi(str)
		if err != nil {
			return 0, fmt.Errorf("could not convert %q to an int: %v", str, err)
		}
		sum += val
	}
	return sum, nil
}

// SumParseFloat converts all the strings in a slice to floating points, sums them up, and
// returns the result as a floating point. A non-nil error is returned if an error occurred.
func SumParseFloat(a []string) (float64, error) {
	var sum float64
	for _, str := range a {
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return 0, fmt.Errorf("could not convert %q to float64", str)
		}
		sum += val
	}
	return sum, nil
}

// TrimCharacter trims the specified character from each string in a slice of strings and
// returns a slice with the trimmed strings.
func TrimCharacter(a []string, char string) []string {
	var res []string
	for _, str := range a {
		str = strings.Trim(str, char)
		res = append(res, str)
	}
	return res
}
