// Package cloudlogger provides functionality to log text or json data to
// Google Cloud logging backend.

package cloudlogger

import (
	"errors"

	"cloud.google.com/go/logging"
	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/utils"
)

// Defines the method required to log a text string to Google Cloud logging
// backend.
type TextLogger interface {
	Printf(text string, a ...interface{})
}

// Defines the method required to log anything that can be marshaled to JSON
// to Google Cloud logging backend.
type StructuredLogger interface {
	Log(l logging.Entry)
}

// LogText writes the string infoToLog to a logging backend by calling the
// `Printf` method defined by the TextLogger interface.
// To log some text to Google Cloud Logging backend , pass in an instance of
// type *log.Logger.
func LogText(g TextLogger, infoToLog string) error {
	if infoToLog == "" {
		return errors.New("logging empty string")
	}
	g.Printf(infoToLog)
	return nil
}

// LogUSEReport writes an array of USEMetrics to a logging backend by calling
// the `Log` method defined by the StructuredLogger interface.
// To log a JSON Payload to Google Cloud Logging backend, pass in an instance
// of type *logging.Logger.
func LogUSEReport(g StructuredLogger, useReport *utils.USEReport) error {
	if useReport == nil {
		return errors.New("logging empty useReport")
	}
	entry := logging.Entry{
		// Log anything that can be marshaled to JSON.
		Payload: struct {
			Components []utils.USEMetrics
			Analysis   string
		}{
			Components: useReport.Components,
			Analysis:   useReport.Analysis,
		},
		Severity: logging.Debug,
	}
	g.Log(entry)
	return nil
}
