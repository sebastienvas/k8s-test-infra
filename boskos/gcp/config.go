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
	"encoding/json"

	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/mason"
)

const (
	ResourceConfigType = "GCPResourceConfig"
)

type GKEClusterConfig struct {
	MachineType string `json:"machineType,omitempty"`
	NumNodes    int    `json:"numNodes,omitempty"`
	Version     string `json:"version,omitempty"`
	Zone        string `json:"zone,ompitempty"`
}

type GCEVMConfig struct {
	MachineType string `json:"machineType,omitempty"`
	Image       string `json:"image,omitempty"`
}

type ProjectConfig struct {
	Type     string             `json:"type,omitempty"`
	Clusters []GKEClusterConfig `json:"clusters,omitempty"`
	Vms      []GCEVMConfig      `json:"vms,omitempty"`
}

type ResourceConfig struct {
	ProjectConfigs []ProjectConfig `json:"projectconfigs,omitempty"`
}

type GKEClusterInfo struct {
	kubeconfig string `json:"kubeconfig"`
}

type GCEVMInfo struct {
	SSHKey string `json:"sshkey"`
	IP     string `json:"ip"`
}

type ProjectInfo struct {
	Name     string           `json:"name"`
	Clusters []GKEClusterInfo `json:"clusters,omitempty"`
	VMs      []GCEVMInfo      `json:"vms,omitempty"`
}

type ResourceInfo struct {
	ProjectsInfo []ProjectInfo `json:projectsinfo,omitempty`
}

func (rc *ResourceConfig) Construct(res *common.Resource, types common.TypeToResources) (*common.ResourceInfo, error) {
	return nil, nil
}
func (rc *ResourceConfig) GetName() string {
	return ResourceConfigType
}

func ConfigConverter(in []byte) (mason.Config, error) {
	var config *ResourceConfig
	if err := json.Unmarshal(in, &config); err != nil {
		return nil, err
	}
	return config, nil
}
