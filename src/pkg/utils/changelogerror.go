// Copyright 2020 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This package contains the error interface returned by changelog and
// findbuild packages. It includes functions to retrieve HTTP status codes
// from Gerrit and Gitiles errors, and functions to create ChangelogErrors
// relevant to the changelog and findbuild features.

package utils

import (
	"errors"
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	grpcCodeToHTTP = map[string]string{
		codes.Unknown.String():            "500",
		codes.InvalidArgument.String():    "400",
		codes.NotFound.String():           "404",
		codes.PermissionDenied.String():   "403",
		codes.Unauthenticated.String():    "401",
		codes.ResourceExhausted.String():  "429",
		codes.FailedPrecondition.String(): "400",
		codes.OutOfRange.String():         "400",
		codes.Internal.String():           "500",
		codes.Unavailable.String():        "503",
		codes.DataLoss.String():           "500",
	}
	httpStatus = map[string]string{
		"400": "400 Bad Request",
		"401": "401 Unauthorized",
		"403": "403 Forbidden",
		"404": "404 Not Found",
		"406": "406 Not Acceptable",
		"429": "429 Too Many Requests",
		"500": "500 Internal Server Error",
		"503": "503 Service Unavailable",
	}
	// Gitiles specific error messages
	// Specifies a string format to allow the build number to be inserted
	httpGitilesErrorFmt = map[string]string{
		"404": "Build number %s not found. Please input a valid build number (example: 13310.1035.0).",
	}
	// Gerrit specific error messages
	// Specifies a string format to allow CL number to be inserted
	httpGerritErrorFmt = map[string]string{
		"400": "%s is not a recognized CL identifier. Please enter either a CL-Number (example: 3206) or a Commit-SHA (example: I7e549d7753cc7acec2b44bb5a305347a97719ab9) for a submitted CL.",
		"404": "No build found for a CL with ID %s. Please enter either the CL-Number (example: 3206) or a Commit-SHA (example: I7e549d7753cc7acec2b44bb5a305347a97719ab9) of a submitted CL.",
		"406": "CL identifier %s maps to a CL that has not been submitted yet. A CL will not enter any build until it is successfully submitted. Please provide the CL identifier for a submitted CL.",
	}
	// Standard error messages
	httpErrorReplacement = map[string]string{
		"403": "This account does not have access to internal repositories. Please retry with an authorized account, or select the external button to query from publically accessible builds.",
		"429": "Our servers are currently experiencing heavy load. Please retry in a couple minutes.",
		"500": "An unexpected error occurred while retrieving the requested information.",
	}
	gitiles403ErrMsg = "unexpected HTTP 403 from Gitiles"
	gerritErrCodeRe  = regexp.MustCompile("status code\\s*(\\d+)")

	// InternalError is a ChangelogError object indicating an internal error
	InternalError = newError("500", httpErrorReplacement["500"])
)

// ChangelogError is the error type used by the changelog and findbuild package
type ChangelogError interface {
	error
	HTTPCode() string
	HTTPStatus() string
}

// UtilChangelogError implements the ChangelogError interface
type UtilChangelogError struct {
	httpCode   string
	httpStatus string
	err        string
}

// HTTPCode retrieves the HTTP error code associated with the error
// ex. 400
func (e *UtilChangelogError) HTTPCode() string {
	return e.httpCode
}

// HTTPStatus retrieves the full HTTP status associated with the error
// ex. 400 Bad Request
func (e *UtilChangelogError) HTTPStatus() string {
	return e.httpStatus
}

func (e *UtilChangelogError) Error() string {
	return e.err
}

func unwrapError(err error) error {
	innerErr := err
	for errors.Unwrap(innerErr) != nil {
		innerErr = errors.Unwrap(innerErr)
	}
	return innerErr
}

// newError creates a new UtilChangelogError
func newError(httpCode, errString string) *UtilChangelogError {
	output := UtilChangelogError{
		httpCode: httpCode,
		err:      errString,
	}
	if header, ok := httpStatus[httpCode]; ok {
		output.httpStatus = header
	} else {
		log.Errorf("No HTTP status mapping for HTTP code %s", httpCode)
		output.httpStatus = httpStatus["500"]
	}
	return &output
}

// FromChangelogError creates Changelog errors that are relevant to Changelog
// functionality.
func FromChangelogError(httpCode, buildNum string) *UtilChangelogError {
	if errFmt, ok := httpGitilesErrorFmt[httpCode]; ok {
		errStr := fmt.Sprintf(errFmt, buildNum)
		return newError("404", errStr)
	} else if replacementErr, ok := httpErrorReplacement[httpCode]; ok {
		return newError(httpCode, replacementErr)
	}
	return InternalError
}

// FromFindBuildError creates Changelog errors that are relevant to FindBuild
// functionality.
func FromFindBuildError(httpCode string, clID string) *UtilChangelogError {
	if errFmt, ok := httpGerritErrorFmt[httpCode]; ok {
		errStr := fmt.Sprintf(errFmt, clID)
		return newError(httpCode, errStr)
	} else if replacementErr, ok := httpErrorReplacement[httpCode]; ok {
		return newError(httpCode, replacementErr)
	}
	return InternalError
}

// GitilesErrCode parses a Gitiles error message and returns an HTTP error code
// associated with the error. Returns 500 if no error code is found.
func GitilesErrCode(err error) string {
	rpcStatus, ok := status.FromError(err)
	if !ok {
		return "500"
	}
	code, text := rpcStatus.Code(), rpcStatus.Message()
	// RPC status code misclassifies 403 error as 500 error for Gitiles requests
	if code == codes.Internal && text == gitiles403ErrMsg {
		code = codes.PermissionDenied
	}
	if httpCode, ok := grpcCodeToHTTP[code.String()]; ok {
		return httpCode
	}
	return "500"
}

// GerritErrCode parse a Gerrit error and returns an HTTP error code associated
// with the error. Returns 500 if no error code is found.
func GerritErrCode(err error) string {
	matches := gerritErrCodeRe.FindStringSubmatch(err.Error())
	if len(matches) != 2 {
		return "500"
	}
	return matches[1]
}
