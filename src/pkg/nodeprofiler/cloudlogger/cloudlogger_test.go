package cloudlogger

import (
	"testing"
	"time"

	"cloud.google.com/go/logging"
	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/utils"
	"github.com/google/go-cmp/cmp"
)

// fakeTextLogger is a struct that implements the TextLogger interface.
type fakeTextLogger struct {
	logged string
}

// fakeStructuredLogger is a struct that implements the StructuredLogger interface.
type fakeStructuredLogger struct {
	logged logging.Entry
}

// Printf behavior with regards to type fakeTextLogger.
func (f *fakeTextLogger) Printf(text string, a ...interface{}) {
	f.logged = text
}

// Log behavior with regards to type fakeStructuredLogger.
func (f *fakeStructuredLogger) Log(entry logging.Entry) {
	f.logged = entry
}

// For every input, the logged string must be the expected output unless the
// the input is an empty string. In that case, nothing get logged.
func TestTableLogText(t *testing.T) {
	var tests = []struct {
		name       string
		input      string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "non-empty string log",
			input:      "Node Profiler",
			wantOutput: "Node Profiler",
			wantErr:    false,
		},
		{
			name:       "empty log",
			input:      "",
			wantOutput: "",
			wantErr:    true,
		},
	}

	for _, test := range tests {
		var f *fakeTextLogger = &fakeTextLogger{}
		err := LogText(f, test.input)
		// err will not be nil if the user attempted to log an empty string.
		// ignoring the case in which the user logged empty string.
		if gotErr := err != nil; gotErr != test.wantErr {

			t.Errorf("LogText(%v, %v) = %q, wantErr %v", f, test.input, err, test.wantErr)
		}
		if diff := cmp.Diff(test.wantOutput, f.logged); diff != "" {
			t.Errorf("Ran LogText(%v, %v), but got mismatch between got and want (-got, +want): \n diff %s", f, test.input, diff)
		}

	}
}
func TestTableLogUSEReport(t *testing.T) {
	// [Begin Making of mocked out instances]
	useMetrics := []utils.USEMetrics{

		{
			Timestamp:   time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
			Interval:    time.Duration(100) * time.Millisecond,
			Utilization: 4,
			Saturation:  false,
			Errors:      int64(0),
		},
		{
			Timestamp:   time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
			Interval:    time.Duration(100) * time.Millisecond,
			Utilization: 7,
			Saturation:  false,
			Errors:      int64(0),
		},
	}
	analysis := "No action required"
	useReport := &utils.USEReport{Components: useMetrics, Analysis: analysis}
	// [End Making of mocked out instances]

	var empty *utils.USEReport
	var tests = []struct {
		name       string
		input      *utils.USEReport
		wantOutput logging.Entry
		wantErr    bool
	}{
		{
			name:  "non-empty json payload log",
			input: useReport,
			wantOutput: logging.Entry{
				Payload: struct {
					Components []utils.USEMetrics
					Analysis   string
				}{
					Components: useReport.Components,
					Analysis:   useReport.Analysis,
				},
				Severity: logging.Debug,
			},
			wantErr: false,
		},
		{
			name:       "empty log",
			input:      empty,
			wantOutput: logging.Entry{},
			wantErr:    true,
		},
	}
	for _, test := range tests {
		var f *fakeStructuredLogger = &fakeStructuredLogger{}
		err := LogUSEReport(f, test.input)
		if gotErr := err != nil; gotErr != test.wantErr {

			t.Errorf("LogUSEReport(%v, %v) = %q, wantErr %v", f, test.input, err, test.wantErr)
		}
		if diff := cmp.Diff(test.wantOutput, f.logged); diff != "" {
			t.Errorf("Ran LogUSEReport(%v,%v), but got mismatch between got and want (-got, +want): \n diff %s", f.logged, test.input, diff)
		}

	}
}
