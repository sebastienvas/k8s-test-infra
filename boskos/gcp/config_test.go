package gcp

import (
	"testing"

	"k8s.io/test-infra/boskos/ranch"
	"reflect"
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
