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
	"fmt"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	testInstanceURL = "cos-review.googlesource.com"
)

func testCLLink(clID, instanceURL string) string {
	return fmt.Sprintf("<a href=\"%s/c/%s\" target=\"_blank\">CL %s</a>", instanceURL, clID, clID)
}

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
				t.Errorf("expected HTTP code %s, got %s", test.expectedCode, code)
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
				t.Errorf("expected HTTP code %s, got %s", test.expectedCode, code)
			}
		})
	}
}

func TestBothBuildsNotFound(t *testing.T) {
	source := "1a"
	target := "ddddddd"
	croslandURL := "https://www.google.com"
	expectedCode := "404"
	expectedErrHeader := "Build Not Found"
	expectedErrStr := strings.Join([]string{
		"The builds associated with input",
		source,
		"and",
		target,
		"cannot be found. It may be possible that the inputs are either invalid or both belong",
		"to pre-Cusky builds. If both of the inputs belong to pre-Cusky builds, note that this tool only supports changelogs",
		"between Cusky builds. Otherwise, please input valid build numbers (example: 13310.1035.0) or valid image names",
		"(example: cos-rc-85-13310-1034-0).",
	}, " ")
	expectedHTMLErrStr := fmt.Sprintf("%s %s and %s %s<br><br>%s %s <a href=%s target=\"_blank\">%s</a>. %s %s",
		"The builds associated with input",
		source,
		target,
		"could not be found.",
		"It may be possible that the inputs are either invalid or both belong to pre-Cusky builds.",
		"If both of the inputs belong to pre-Cusky builds, please check",
		croslandLink(croslandURL, source, target),
		croslandLink(croslandURL, source, target),
		"Otherwise, please input valid build numbers",
		"(example: 13310.1035.0) or valid image names (example: cos-rc-85-13310-1034-0).",
	)
	err := BothBuildsNotFound(croslandURL, source, target)
	if err.HTTPCode() != expectedCode {
		t.Errorf("expected HTTP code %s, got %s", expectedCode, err.HTTPCode())
	} else if err.Header() != expectedErrHeader {
		t.Errorf("expected error header \"%s\", got %s", expectedErrHeader, err.Header())
	} else if err.Error() != expectedErrStr {
		t.Errorf("expected error string %s, got %s", expectedErrStr, err.Error())
	} else if err.HTMLError() != expectedHTMLErrStr {
		t.Errorf("expected html error string %s, got %s", expectedErrStr, err.HTMLError())
	} else if err.Retryable() {
		t.Errorf("expected retryable = false, got true")
	}
}

func TestBuildNotFound(t *testing.T) {
	buildNumber := "15000.0.0"
	expectedCode := "404"
	expectedErrHeader := "Build Not Found"
	expectedErrStr := strings.Join([]string{
		"The build associated with input",
		buildNumber,
		"cannot be found. It may be possible that the input is either invalid or belongs to a",
		"pre-Cusky build. If you entered a pre-Cusky build number or image name, note that changelog between",
		"pre-Cusky and Cusky builds are not supported. Otherwise, please input a valid build number",
		"(example: 13310.1035.0) or a valid image name (example: cos-rc-85-13310-1034-0).",
	}, " ")
	expectedHTMLErrStr := fmt.Sprintf("%s %s %s<br><br>%s %s %s %s",
		"The build associated with input",
		buildNumber,
		"cannot be found.",
		"It may be possible that either the input is either invalid or belongs to a",
		"pre-Cusky build. If you entered a pre-Cusky build number or image name, note that changelog between",
		"pre-Cusky and Cusky builds are not supported. Otherwise, please input a valid build number",
		"(example: 13310.1035.0) or a valid image name (example: cos-rc-85-13310-1034-0).",
	)
	err := BuildNotFound(buildNumber)
	if err.HTTPCode() != expectedCode {
		t.Errorf("expected HTTP code %s, got %s", expectedCode, err.HTTPCode())
	} else if err.Header() != expectedErrHeader {
		t.Errorf("expected error header \"%s\", got %s", expectedErrHeader, err.Header())
	} else if err.Error() != expectedErrStr {
		t.Errorf("expected error string %s, got %s", expectedErrStr, err.Error())
	} else if err.HTMLError() != expectedHTMLErrStr {
		t.Errorf("expected html error string %s, got %s", expectedErrStr, err.HTMLError())
	} else if err.Retryable() {
		t.Errorf("expected retryable = false, got true")
	}
}

func TestCLNotFound(t *testing.T) {
	clID := "1540"
	expectedCode := "404"
	expectedErrHeader := "CL Not Found"
	expectedErrStr := fmt.Sprintf("No CL was found matching the identifier: %s. Please enter either the CL-number (example: 3206) or a Commit-SHA (example: I7e549d7753cc7acec2b44bb5a305347a97719ab9) of a submitted CL.", clID)
	err := CLNotFound(clID)
	if err.HTTPCode() != expectedCode {
		t.Errorf("expected HTTP code %s, got %s", expectedCode, err.HTTPCode())
	} else if err.Header() != expectedErrHeader {
		t.Errorf("expected error header \"%s\", got %s", expectedErrHeader, err.Header())
	} else if err.Error() != expectedErrStr {
		t.Errorf("expected error string %s, got %s", expectedErrStr, err.Error())
	} else if err.HTMLError() != expectedErrStr {
		t.Errorf("expected html error string %s, got %s", expectedErrStr, err.HTMLError())
	} else if err.Retryable() {
		t.Errorf("expected retryable = false, got true")
	}
}

func TestCLLandingNotFound(t *testing.T) {
	clID := "1540"
	expectedCode := "406"
	expectedErrHeader := "No Build Found"
	expectedErrStr := fmt.Sprintf("No build was found containing CL %s.", clID)
	link := testCLLink(clID, testInstanceURL)
	expectedHTMLErrStr := fmt.Sprintf("No build was found containing %s.", link)
	err := CLLandingNotFound(clID, testInstanceURL)
	if err.HTTPCode() != expectedCode {
		t.Errorf("expected HTTP code %s, got %s", expectedCode, err.HTTPCode())
	} else if err.Header() != expectedErrHeader {
		t.Errorf("expected error header \"%s\", got %s", expectedErrHeader, err.Header())
	} else if err.Error() != expectedErrStr {
		t.Errorf("expected error string %s, got %s", expectedErrStr, err.Error())
	} else if err.HTMLError() != expectedHTMLErrStr {
		t.Errorf("expected html error string %s, got %s", expectedHTMLErrStr, err.HTMLError())
	} else if !err.Retryable() {
		t.Errorf("expected retryable = true, got false")
	}
}

func TestCLNotUsed(t *testing.T) {
	clID := "1540"
	repo := "cos/tools"
	branch := "master"
	expectedCode := "406"
	expectedErrHeader := "CL Not Used"
	expectedErrStr := fmt.Sprintf("CL %s modifies the %s repository on the %s branch, which has not been used in COS builds since the CL's submission.", clID, repo, branch)
	link := testCLLink(clID, testInstanceURL)
	expectedHTMLErrStr := fmt.Sprintf("%s modifies the %s repository on the %s branch, which has not been used in COS builds since the CL's submission.", link, repo, branch)
	err := CLNotUsed(clID, repo, branch, testInstanceURL)
	if err.HTTPCode() != expectedCode {
		t.Errorf("expected HTTP code %s, got %s", expectedCode, err.HTTPCode())
	} else if err.Header() != expectedErrHeader {
		t.Errorf("expected error header \"%s\", got %s", expectedErrHeader, err.Header())
	} else if err.Error() != expectedErrStr {
		t.Errorf("expected error string %s, got %s", expectedErrStr, err.Error())
	} else if err.HTMLError() != expectedHTMLErrStr {
		t.Errorf("expected html error string %s, got %s", expectedHTMLErrStr, err.HTMLError())
	} else if err.Retryable() {
		t.Errorf("expected retryable = false, got true")
	}
}

func TestCLTooRecent(t *testing.T) {
	clID := "1540"
	expectedCode := "406"
	expectedErrHeader := "CL Too Recent"
	expectedErrStr := fmt.Sprintf("CL %s was submitted too recently to be included in any builds. Please wait a couple hours and try again.", clID)
	link := testCLLink(clID, testInstanceURL)
	expectedHTMLErrStr := fmt.Sprintf("%s was submitted too recently to be included in any builds. Please wait a couple hours and try again.", link)
	err := CLTooRecent(clID, testInstanceURL)
	if err.HTTPCode() != expectedCode {
		t.Errorf("expected HTTP code %s, got %s", expectedCode, err.HTTPCode())
	} else if err.Header() != expectedErrHeader {
		t.Errorf("expected error header \"%s\", got %s", expectedErrHeader, err.Header())
	} else if err.Error() != expectedErrStr {
		t.Errorf("expected error string %s, got %s", expectedErrStr, err.Error())
	} else if err.HTMLError() != expectedHTMLErrStr {
		t.Errorf("expected html error string %s, got %s", expectedHTMLErrStr, err.HTMLError())
	} else if err.Retryable() {
		t.Errorf("expected retryable = false, got true")
	}
}

func TestCLNotSubmitted(t *testing.T) {
	clID := "1540"
	expectedCode := "406"
	expectedErrHeader := "CL Not Submitted"
	expectedErrStr := fmt.Sprintf("CL %s has not been submitted yet. A CL will not enter any build until it is successfully submitted.", clID)
	link := testCLLink(clID, testInstanceURL)
	expectedHTMLErrStr := fmt.Sprintf("%s has not been submitted yet. A CL will not enter any build until it is successfully submitted.", link)
	err := CLNotSubmitted(clID, testInstanceURL)
	if err.HTTPCode() != expectedCode {
		t.Errorf("expected HTTP code %s, got %s", expectedCode, err.HTTPCode())
	} else if err.Header() != expectedErrHeader {
		t.Errorf("expected error header \"%s\", got %s", expectedErrHeader, err.Header())
	} else if err.Error() != expectedErrStr {
		t.Errorf("expected error string %s, got %s", expectedErrStr, err.Error())
	} else if err.HTMLError() != expectedHTMLErrStr {
		t.Errorf("expected html error string %s, got %s", expectedHTMLErrStr, err.HTMLError())
	}
}

func TestCLInvalidRelease(t *testing.T) {
	clID := "1540"
	release := "master"
	expectedCode := "406"
	expectedErrHeader := "Invalid Release Branch"
	expectedErrStr := fmt.Sprintf("CL %s maps to release %s, which is not a valid release", clID, release)
	link := testCLLink(clID, testInstanceURL)
	expectedHTMLErrStr := fmt.Sprintf("%s maps to release %s, which is not a valid release", link, release)
	err := CLInvalidRelease(clID, release, testInstanceURL)
	if err.HTTPCode() != expectedCode {
		t.Errorf("expected HTTP code %s, got %s", expectedCode, err.HTTPCode())
	} else if err.Header() != expectedErrHeader {
		t.Errorf("expected error header \"%s\", got %s", expectedErrHeader, err.Header())
	} else if err.Error() != expectedErrStr {
		t.Errorf("expected error string %s, got %s", expectedErrStr, err.Error())
	} else if err.HTMLError() != expectedHTMLErrStr {
		t.Errorf("expected html error string %s, got %s", expectedHTMLErrStr, err.HTMLError())
	} else if err.Retryable() {
		t.Errorf("expected retryable = false, got true")
	}
}
