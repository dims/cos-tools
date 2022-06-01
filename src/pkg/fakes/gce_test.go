// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fakes

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func TestImageList(t *testing.T) {
	testImageListData := []struct {
		testName string
		images   *compute.ImageList
	}{
		{
			"Two images",
			&compute.ImageList{Items: []*compute.Image{{Name: "test-1"}, {Name: "test-2"}}},
		},
		{
			"No images",
			&compute.ImageList{},
		},
		{
			"Image with family",
			&compute.ImageList{Items: []*compute.Image{{Name: "test-1", Family: "test-family"}}},
		},
	}
	fakeGCE, client := GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	for _, input := range testImageListData {
		t.Run(input.testName, func(t *testing.T) {
			fakeGCE.Images = input.images
			actual, err := client.Images.List("test-project").Do()
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(actual.Items, input.images.Items) {
				t.Errorf("actual: %v expected: %v", actual.Items, input.images.Items)
			}
		})
	}
}

func TestImageGet(t *testing.T) {
	testImageGetData := []struct {
		testName string
		images   []*compute.Image
		name     string
		httpCode int
	}{
		{
			"ImageExists",
			[]*compute.Image{{Name: "im-1"}},
			"im-1",
			http.StatusOK,
		},
		{
			"ImageDoesntExist",
			nil,
			"im-2",
			http.StatusNotFound,
		},
	}
	fakeGCE, client := GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	for _, input := range testImageGetData {
		t.Run(input.testName, func(t *testing.T) {
			fakeGCE.Images.Items = input.images
			actualIm, err := client.Images.Get("test-project", input.name).Do()
			if apiErr, ok := err.(*googleapi.Error); ok {
				if apiErr.Code != input.httpCode {
					t.Errorf("actual: %d expected: %d", apiErr.Code, input.httpCode)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if actualIm.Name != input.name {
				t.Errorf("actual: %s expected: %s", actualIm.Name, input.name)
			}
		})
	}
}

func TestDeprecate(t *testing.T) {
	testDeprecateData := []struct {
		testName  string
		images    []*compute.Image
		name      string
		status    *compute.DeprecationStatus
		operation *compute.Operation
		httpCode  int
	}{
		{
			"SetStatusReturnDone",
			[]*compute.Image{{Name: "test-1"}},
			"test-1",
			&compute.DeprecationStatus{State: "DEPRECATED"},
			&compute.Operation{Name: "op-1", Status: "DONE"},
			http.StatusOK,
		},
		{
			"ClearStatusReturnRunning",
			[]*compute.Image{{Name: "test-2"}},
			"test-2",
			&compute.DeprecationStatus{},
			&compute.Operation{Name: "op-1", Status: "RUNNING"},
			http.StatusOK,
		},
		{
			"ImageNotFound",
			nil,
			"test-3",
			&compute.DeprecationStatus{},
			nil,
			http.StatusNotFound,
		},
	}
	fakeGCE, client := GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	for _, input := range testDeprecateData {
		t.Run(input.testName, func(t *testing.T) {
			fakeGCE.Images.Items = input.images
			fakeGCE.Operations = []*compute.Operation{input.operation}
			actualOp, err := client.Images.Deprecate("test-project", input.name, input.status).Do()
			if apiErr, ok := err.(*googleapi.Error); ok {
				if apiErr.Code != input.httpCode {
					t.Errorf("actual: %d expected: %d", apiErr.Code, input.httpCode)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if actualOp.Name != input.operation.Name {
				t.Errorf("actual: %s expected: %s", actualOp.Name, input.operation.Name)
			}
			if actualOp.Status != input.operation.Status {
				t.Errorf("actual: %s expected: %s", actualOp.Status, input.operation.Status)
			}
			if actualStatus, ok := fakeGCE.Deprecated[input.name]; !ok {
				t.Errorf("deprecated images: %v expected element: %s", fakeGCE.Deprecated, input.name)
			} else if !cmp.Equal(actualStatus, input.status) {
				t.Errorf("actual: %v expected: %v", actualStatus, input.status)
			}
		})
	}
}

func TestGetOperation(t *testing.T) {
	testGetOperationData := []struct {
		testName   string
		operations []*compute.Operation
	}{
		{
			"OneOperation",
			[]*compute.Operation{{Name: "op-1"}},
		},
		{
			"TwoOperations",
			[]*compute.Operation{{Name: "op-2"}, {Name: "op-3"}},
		},
	}
	fakeGCE, client := GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	for _, input := range testGetOperationData {
		t.Run(input.testName, func(t *testing.T) {
			fakeGCE.Operations = make([]*compute.Operation, len(input.operations))
			copy(fakeGCE.Operations, input.operations)
			for _, expectedOp := range input.operations {
				actualOp, err := client.GlobalOperations.Get("test-project", "").Do()
				if err != nil {
					t.Error(err)
					continue
				}
				if actualOp.Name != expectedOp.Name {
					t.Errorf("actual: %s expected: %s", actualOp.Name, expectedOp.Name)
				}
			}
		})
	}
}

func TestListInstances(t *testing.T) {
	gce, svc := GCEForTest(t, "project")
	defer gce.Close()
	gce.Instances = []*compute.Instance{
		{
			Name: "instance1",
			Zone: "zone1",
		},
	}
	testData := []struct {
		name            string
		zone            string
		project         string
		expctedInstance []string
		expectErr       bool
	}{
		{
			name:            "Found",
			zone:            "zone1",
			project:         "project",
			expctedInstance: []string{"instance1"},
		},
		{
			name:            "WrongZone",
			zone:            "zone2",
			project:         "project",
			expctedInstance: []string{},
		},
		{
			name:            "WrongProject",
			zone:            "zone1",
			project:         "project2",
			expctedInstance: []string{},
			expectErr:       true,
		},
	}
	for _, input := range testData {
		t.Run(input.name, func(t *testing.T) {
			instancesListCall := svc.Instances.List(input.project, input.zone)
			instancesList, err := instancesListCall.Do()
			if gotErr := err != nil; gotErr != input.expectErr {
				t.Fatalf("Unexpected error status, err: %v, expect err: %v", err, input.expectErr)
			}
			if err == nil {
				if len(instancesList.Items) != len(input.expctedInstance) {
					t.Fatalf("Wrong number of instances: %d, expected: %d", len(instancesList.Items), len(input.expctedInstance))
				}
				for idx, instance := range instancesList.Items {
					if instance.Name != input.expctedInstance[idx] {
						t.Fatalf("Wrong instance name. idx: %d, name: %s, expected: %s", idx, instance.Name, input.expctedInstance[idx])
					}
				}
			}
		})
	}
}

func TestDeleteInstances(t *testing.T) {
	testData := []struct {
		name         string
		zone         string
		project      string
		instanceName string
		expectErr    bool
	}{
		{
			name:         "SuccessfullyDeleted",
			zone:         "zone1",
			project:      "project",
			instanceName: "instance1",
		},
		{
			name:         "WrongZone",
			zone:         "zone2",
			project:      "project",
			instanceName: "instance1",
			expectErr:    true,
		},
		{
			name:         "WrongProject",
			zone:         "zone1",
			project:      "project2",
			instanceName: "instance1",
			expectErr:    true,
		},
		{
			name:         "WrongName",
			zone:         "zone1",
			project:      "project",
			instanceName: "instance2",
			expectErr:    true,
		},
	}
	for _, input := range testData {
		t.Run(input.name, func(t *testing.T) {
			gce, svc := GCEForTest(t, "project")
			defer gce.Close()
			gce.Instances = []*compute.Instance{
				{
					Name: "instance1",
					Zone: "zone1",
				},
			}
			instancesDeleteCall := svc.Instances.Delete(input.project, input.zone, input.instanceName)
			_, err := instancesDeleteCall.Do()
			if gotErr := err != nil; gotErr != input.expectErr {
				t.Fatalf("Unexpected error status, err: %v, expect err: %v", err, input.expectErr)
			}
			if err == nil && len(gce.Instances) != 0 {
				t.Fatalf("Instance not deleted. Number of instnaces left: %d", len(gce.Instances))
			}
		})
	}
}
