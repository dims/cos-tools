// Copyright 2022 Google LLC
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

// Package gce contains high-level functionality for manipulating GCE resources.
package gce

import (
	"fmt"
	"time"

	"google.golang.org/api/compute/v1"
)

const timeLayout = "2006-01-02T15:04:05.999-07:00"

// For test overwriting.
var timeNow = time.Now

// DeleteOldVmWithLabel deletes all old VMs in the target project in the target zone with
// the given label. If the value of the label is empty, all VMs with the provided key of
// the label will be deleted. ttl must be at least 1 hour.
func DeleteOldVMWithLabel(gceService *compute.Service, project, zone, labelKey, labelValue string, ttl time.Duration) error {
	if project == "" || zone == "" || labelKey == "" {
		return fmt.Errorf("project name, zone, and labelKey cannot be empty. project: %s, zone: %s, labelKey: %s", project, zone, labelKey)
	}
	if ttl < time.Hour {
		return fmt.Errorf("ttl must be at least 1 hour, ttl: %v", ttl)
	}
	instancesListCall := gceService.Instances.List(project, zone)
	instancesList, err := instancesListCall.Do()
	if err != nil {
		return fmt.Errorf("failed to list instances in project %q in zone %q, err: %v", project, zone, err)
	}
	for _, instance := range instancesList.Items {
		if value, found := instance.Labels[labelKey]; found {
			if labelValue != "" && value != labelValue {
				continue
			}
			creationTime, err := time.Parse(timeLayout, instance.CreationTimestamp)
			if err != nil {
				return fmt.Errorf("failed to parse instanceCreationTimestamp %q, err: %v", instance.CreationTimestamp, err)
			}
			if timeNow().Before(creationTime.Add(ttl)) {
				continue
			}
			instancesDeleteCall := gceService.Instances.Delete(project, zone, instance.Name)
			if _, err := instancesDeleteCall.Do(); err != nil {
				return fmt.Errorf("failed to delete instance %q in project %q in zone %q, err: %v", instance.Name, project, zone, err)
			}
		}
	}
	return nil
}
