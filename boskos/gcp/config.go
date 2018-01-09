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
	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/mason"
)

const (
	ResourceConfigType = "GCPResourceConfig"
)

type GKEClusterConfig struct {
	MachineType string `json:"machinetype,omitempty"`
	NumNodes    int    `json:"numnodes,omitempty"`
	Version     string `json:"version,omitempty"`
	Zone        string `json:"zone,ompitempty"`
}

type GCEVMConfig struct {
	MachineType string `json:"machinetype,omitempty"`
	Image       string `json:"image,omitempty"`
	Zone        string `json:"zone,ompitempty"`
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
	Kubeconfig string `json:"kubeconfig"`
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
	ProjectsInfo []ProjectInfo `json:"projectsinfo,omitempty"`
}

func (rc *ResourceConfig) Construct(res *common.Resource, types common.TypeToResources) (*common.ResourceInfo, error) {
	return nil, nil
}
func (rc *ResourceConfig) GetName() string {
	return ResourceConfigType
}

func configConverter(in string) (*ResourceConfig, error) {
	var config ResourceConfig
	if err := yaml.Unmarshal([]byte(in), &config); err != nil {
		logrus.WithError(err).Errorf("unable to parse %s", in)
		return nil, err
	}
	return &config, nil
}

func ConfigConverter(in string) (mason.Config, error) {
	return configConverter(in)
}
