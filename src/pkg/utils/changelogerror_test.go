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

package utils

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGerritErrCode(t *testing.T) {
	tests := map[string]struct {
		inputErr     error
		expectedCode string
	}{
		"Empty Error": {
			inputErr:     errors.New(""),
			expectedCode: "500",
		},
		"Mapped Error Code": {
			inputErr:     errors.New("failed to fetch \"https://cos-internal-review.googlesource.com/a/changes/?n=1&o=CURRENT_REVISION&q=1\", status code 403"),
			expectedCode: "403",
		},
		"Irregular Code": {
			inputErr:     errors.New("failed to fetch \"https://cos-internal-review.googlesource.com/a/changes/?n=1&o=CURRENT_REVISION&q=1\", status code 689"),
			expectedCode: "689",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			code := GerritErrCode(test.inputErr)
			if code != test.expectedCode {
				t.Errorf("GerritErrCode failed, expected HTTP code %s, got %s", test.expectedCode, code)
			}
		})
	}
}

func TestFindBuildError(t *testing.T) {
	tests := map[string]struct {
		inputCode      string
		clID           string
		expectedCode   string
		expectedStatus string
		expectedErrStr string
	}{
		"No Code": {
			inputCode:      "500",
			clID:           "1",
			expectedCode:   "500",
			expectedStatus: "500 Internal Server Error",
			expectedErrStr: "An unexpected error occurred while retrieving the requested information.",
		},
		"Unformatted Error": {
			inputCode:      "403",
			clID:           "1",
			expectedCode:   "403",
			expectedStatus: "403 Forbidden",
			expectedErrStr: "This account does not have access to internal repositories. Please retry with an authorized account, or select the external button to query from publically accessible builds.",
		},
		"Formatted Error": {
			inputCode:      "404",
			clID:           "1214",
			expectedCode:   "404",
			expectedStatus: "404 Not Found",
			expectedErrStr: "No build found for a CL with ID 1214. Please enter either the CL-Number (example: 3206) or a Commit-SHA (example: I7e549d7753cc7acec2b44bb5a305347a97719ab9) of a submitted CL.",
		},
		"Unmapped Code": {
			inputCode:      "500",
			clID:           "1",
			expectedCode:   "500",
			expectedStatus: "500 Internal Server Error",
			expectedErrStr: "An unexpected error occurred while retrieving the requested information.",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := FromFindBuildError(test.inputCode, test.clID)
			if err.HTTPCode() != test.expectedCode {
				t.Errorf("FromFindBuildError failed, expected HTTP code %s, got %s", test.expectedCode, err.HTTPCode())
			} else if err.HTTPStatus() != test.expectedStatus {
				t.Errorf("FromFindBuildError failed, expected HTTP status %s, got %s", test.expectedStatus, err.HTTPStatus())
			} else if err.Error() != test.expectedErrStr {
				t.Errorf("FromFindBuildError failed, expected error string %s, got %s", test.expectedErrStr, err.Error())
			}
		})
	}
}

func TestGitilesErrCode(t *testing.T) {
	tests := map[string]struct {
		inputErr     error
		expectedCode string
	}{
		"Default Error Code": {
			inputErr:     errors.New("code = e desc = not a desc"),
			expectedCode: "500",
		},
		"Mapped Error Code": {
			inputErr:     status.New(codes.NotFound, "not found").Err(),
			expectedCode: "404",
		},
		"403 Code Edge Case": {
			inputErr:     status.New(codes.Internal, "unexpected HTTP 403 from Gitiles").Err(),
			expectedCode: "403",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			code := GitilesErrCode(test.inputErr)
			if code != test.expectedCode {
				t.Errorf("GitilesErrCode failed, expected HTTP code %s, got %s", test.expectedCode, code)
			}
		})
	}
}

func TestFromChangelogError(t *testing.T) {
	tests := map[string]struct {
		inputCode      string
		clID           string
		expectedCode   string
		expectedStatus string
		expectedErrStr string
	}{
		"No Code": {
			inputCode:      "500",
			clID:           "1",
			expectedCode:   "500",
			expectedStatus: "500 Internal Server Error",
			expectedErrStr: "An unexpected error occurred while retrieving the requested information.",
		},
		"Unformatted Error": {
			inputCode:      "403",
			clID:           "1",
			expectedCode:   "403",
			expectedStatus: "403 Forbidden",
			expectedErrStr: "This account does not have access to internal repositories. Please retry with an authorized account, or select the external button to query from publically accessible builds.",
		},
		"Formatted Error": {
			inputCode:      "404",
			clID:           "15000.1.0",
			expectedCode:   "404",
			expectedStatus: "404 Not Found",
			expectedErrStr: "Build number 15000.1.0 not found. Please input a valid build number (example: 13310.1035.0).",
		},
		"Unmapped Code": {
			inputCode:      "500",
			clID:           "1",
			expectedCode:   "500",
			expectedStatus: "500 Internal Server Error",
			expectedErrStr: "An unexpected error occurred while retrieving the requested information.",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := FromChangelogError(test.inputCode, test.clID)
			if err.HTTPCode() != test.expectedCode {
				t.Errorf("FromChangelogError failed, expected HTTP code %s, got %s", test.expectedCode, err.HTTPCode())
			} else if err.HTTPStatus() != test.expectedStatus {
				t.Errorf("FromChangelogError failed, expected HTTP status %s, got %s", test.expectedStatus, err.HTTPStatus())
			} else if err.Error() != test.expectedErrStr {
				t.Errorf("FromChangelogError failed, expected error string %s, got %s", test.expectedErrStr, err.Error())
			}
		})
	}
}
