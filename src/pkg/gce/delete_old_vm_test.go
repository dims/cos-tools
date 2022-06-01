// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gce

import (
	"testing"
	"time"

	"cos.googlesource.com/cos/tools.git/src/pkg/fakes"
	"google.golang.org/api/compute/v1"
)

func TestDeleteOldVMWithLabel(t *testing.T) {
	testData := []struct {
		name                   string
		project                string
		zone                   string
		labelKey               string
		labelValue             string
		ttl                    time.Duration
		expctedInstanceDeleted []string
		wantErr                bool
	}{
		{
			name:                   "DeleteByKey",
			project:                "project",
			zone:                   "zone",
			labelKey:               "key1",
			labelValue:             "",
			ttl:                    time.Hour * 24,
			expctedInstanceDeleted: []string{"instance1", "instance2"},
		},
		{
			name:                   "DeleteByKeyAndValue",
			project:                "project",
			zone:                   "zone",
			labelKey:               "key1",
			labelValue:             "value1",
			ttl:                    time.Hour * 24,
			expctedInstanceDeleted: []string{"instance1"},
		},
		{
			name:       "NoOldVM",
			project:    "project",
			zone:       "zone",
			labelKey:   "key1",
			labelValue: "",
			ttl:        time.Hour * 24 * 10,
		},
		{
			name:       "NoTargetLabelKey",
			project:    "project",
			zone:       "zone",
			labelKey:   "key123",
			labelValue: "",
			ttl:        time.Hour * 24,
		},
		{
			name:       "NoTargetLabelValue",
			project:    "project",
			zone:       "zone",
			labelKey:   "key1",
			labelValue: "vvvv",
			ttl:        time.Hour * 24,
		},
		{
			name:       "NoProject",
			project:    "",
			zone:       "zone",
			labelKey:   "key1",
			labelValue: "vvvv",
			ttl:        time.Hour * 24,
			wantErr:    true,
		},
		{
			name:       "NoZone",
			project:    "project",
			zone:       "",
			labelKey:   "key1",
			labelValue: "vvvv",
			ttl:        time.Hour * 24,
			wantErr:    true,
		},
		{
			name:       "NoLabelKey",
			project:    "project",
			zone:       "zone",
			labelKey:   "",
			labelValue: "vvvv",
			ttl:        time.Hour * 24,
			wantErr:    true,
		},
		{
			name:       "NoTTL",
			project:    "project",
			zone:       "zone",
			labelKey:   "key1",
			labelValue: "vvvv",
			wantErr:    true,
		},
		{
			name:       "TTLTooShort",
			project:    "project",
			zone:       "zone",
			labelKey:   "key1",
			labelValue: "vvvv",
			ttl:        time.Minute * 59,
			wantErr:    true,
		},
	}
	for _, test := range testData {
		timeNow = func() time.Time {
			t, _ := time.Parse(timeLayout, "2022-05-14T15:35:45.579-07:00")
			return t
		}
		gce, gceService := fakes.GCEForTest(t, "project")
		defer gce.Close()
		gce.Instances = []*compute.Instance{
			{
				Name:              "instance1",
				Labels:            map[string]string{"key1": "value1"},
				Zone:              "zone",
				CreationTimestamp: "2022-05-12T15:35:45.579-07:00",
			},
			{
				Name:              "instance2",
				Labels:            map[string]string{"key1": ""},
				Zone:              "zone",
				CreationTimestamp: "2022-05-12T15:35:45.579-07:00",
			},
		}
		err := DeleteOldVMWithLabel(gceService, test.project, test.zone, test.labelKey, test.labelValue, test.ttl)
		if gotErr := err != nil; gotErr != test.wantErr {
			t.Fatalf("%s: Unexpected error status. wantErr: %v, got err: %v", test.name, test.wantErr, err)
		}
		if err == nil {
			for _, instance := range gce.Instances {
				for _, expectDelete := range test.expctedInstanceDeleted {
					if instance.Name == expectDelete {
						t.Fatalf("%s: instance %q not deleted", test.name, expectDelete)
					}
				}
			}
		}
	}
}
