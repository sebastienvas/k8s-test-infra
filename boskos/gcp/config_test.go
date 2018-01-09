/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gcp

import (
	"reflect"
	"testing"

	"k8s.io/test-infra/boskos/ranch"
)

func TestParseConfig(t *testing.T) {
	expected := ResourceConfig{
		ProjectConfigs: []ProjectConfig{
			{
				Type: "type1",
				Clusters: []GKEClusterConfig{
					{
						MachineType: "n1-standard-2",
						NumNodes:    4,
						Version:     "1.7",
						Zone:        "us-central-1f",
					},
				},
				Vms: []GCEVMConfig{
					{
						MachineType: "n1-standard-4",
						Image:       "debian-9-drawfork",
						Zone:        "us-central-1f",
					},
				},
			},
		},
	}
	_, conf, err := ranch.ParseConfig("../test-config.yaml")
	if err != nil {
		t.Error("could not parse config")
	}
	config, err := configConverter(conf[0].Config.Content)
	if err != nil {
		t.Errorf("cannot parse object")
	} else {
		if !reflect.DeepEqual(expected, *config) {
			t.Error("Object differ")
		}
	}
}
