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
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/mason"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"

	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

var (
	serviceAccount = flag.String("service-account", "", "Path to projects service account")
)

const (
	ResourceConfigType = "GCPResourceConfig"
	done               = "DONE"
	persistent         = "PERSISTENT"
	oneToOneNAT        = "ONE_TO_ONE_NAT"
)

type GKEClusterConfig struct {
	MachineType           string `json:"machinetype,omitempty"`
	NumNodes              int64  `json:"numnodes,omitempty"`
	Version               string `json:"version,omitempty"`
	Zone                  string `json:"zone,ompitempty"`
	EnableKubernetesAlpha bool   `json:"enablekubernetesalpha"`
}

type GCEVMConfig struct {
	MachineType string   `json:"machinetype,omitempty"`
	SourceImage string   `json:"sourceimage,omitempty"`
	Zone        string   `json:"zone,ompitempty"`
	Tags        []string `json:"tags,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
}

type ProjectConfig struct {
	Type     string             `json:"type,omitempty"`
	Clusters []GKEClusterConfig `json:"clusters,omitempty"`
	Vms      []GCEVMConfig      `json:"vms,omitempty"`
}

type ResourceConfig struct {
	ProjectConfigs []ProjectConfig `json:"projectconfigs,omitempty"`
}

type InstanceInfo struct {
	Name string `json:"name"`
}

type ProjectInfo struct {
	Name     string         `json:"name"`
	Clusters []InstanceInfo `json:"clusters,omitempty"`
	VMs      []InstanceInfo `json:"vms,omitempty"`
}

type ResourceInfo struct {
	ProjectsInfo []ProjectInfo `json:"projectsinfo,omitempty"`
}

type client struct {
	gkeService *container.Service
	gceService *compute.Service
}

func (rc *ResourceConfig) Construct(res *common.Resource, types common.TypeToResources) (*common.TypedContent, error) {
	typedContent := common.TypedContent{Type: ResourceConfigType}
	info := ResourceInfo{}
	var err error

	gcpClient, err := newClient()
	if err != nil {
		return nil, err
	}

	// Here we know that resources are of project type
	for _, pc := range rc.ProjectConfigs {
		var project *common.Resource
		project, types[pc.Type] = types[pc.Type][0], types[pc.Type][1:]
		projectInfo := ProjectInfo{Name: project.Name}
		for _, cl := range pc.Clusters {
			var clusterInfo *InstanceInfo
			clusterInfo, err = gcpClient.createCluster(project.Name, cl)
			if err != nil {
				return nil, err
			}
			projectInfo.Clusters = append(projectInfo.Clusters, *clusterInfo)
		}
		for _, vm := range pc.Vms {
			var vmInfo *InstanceInfo
			vmInfo, err = gcpClient.createVM(project.Name, vm)
			if err != nil {
				return nil, err
			}
			projectInfo.VMs = append(projectInfo.VMs, *vmInfo)
		}
		info.ProjectsInfo = append(info.ProjectsInfo, projectInfo)
	}

	typedContent.Content, err = info.ToString()
	return &typedContent, err
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

func (ri *ResourceInfo) ToString() (string, error) {
	out, err := yaml.Marshal(ri)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func newClient() (*client, error) {
	var (
		oauthClient *http.Client
		err         error
	)
	if *serviceAccount != "" {
		var data []byte
		data, err = ioutil.ReadFile(*serviceAccount)
		if err != nil {
			return nil, err
		}
		var conf *jwt.Config
		conf, err = google.JWTConfigFromJSON(data, compute.CloudPlatformScope)
		if err != nil {
			return nil, err
		}
		oauthClient = conf.Client(context.Background())
	} else {
		oauthClient, err = google.DefaultClient(context.Background(), compute.CloudPlatformScope)
		if err != nil {
			return nil, err
		}
	}
	gkeService, err := container.New(oauthClient)
	if err != nil {
		return nil, err
	}
	gceService, err := compute.New(oauthClient)
	if err != nil {
		return nil, err
	}
	return &client{
		gceService: gceService,
		gkeService: gkeService,
	}, nil
}

func findVersionMatch(version string, supportedVersion []string) (string, error) {
	for _, v := range supportedVersion {
		if strings.HasPrefix(v, version) {
			return v, nil
		}
	}
	return "", nil
}

func generateName(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, time.Now().Format("0102150405"))
}

func (cc *client) checkGKEOperation(project, zone, id string) error {
	newOp, err := cc.gkeService.Projects.Zones.Operations.Get(project, zone, id).Do()
	if err != nil {
		return err
	}
	if newOp.Status == done {
		if newOp.StatusMessage == "" {
			return nil
		}
		return fmt.Errorf(newOp.StatusMessage)
	}
	time.Sleep(10 * time.Second)
	return cc.checkGKEOperation(project, zone, id)
}

func (cc *client) checkGCEOperation(project, zone, id string) error {
	newOp, err := cc.gceService.ZoneOperations.Get(project, zone, id).Do()
	if err != nil {
		return err
	}
	if newOp.Status == done {
		if newOp.StatusMessage == "" {
			return nil
		}
		return fmt.Errorf(newOp.StatusMessage)
	}
	time.Sleep(10 * time.Second)
	return cc.checkGKEOperation(project, zone, id)
}

func (cc *client) createCluster(project string, config GKEClusterConfig) (*InstanceInfo, error) {
	var version string
	name := generateName("gke")
	serverConfig, err := cc.gkeService.Projects.Zones.GetServerconfig(project, config.Zone).Do()
	if err != nil {
		return nil, err
	}
	if config.Version == "" {
		version = serverConfig.DefaultClusterVersion
	} else {
		version, err = findVersionMatch(config.Version, serverConfig.ValidMasterVersions)
		if err != nil {
			return nil, err
		}
	}
	clusterRequest := &container.CreateClusterRequest{
		Cluster: &container.Cluster{
			Name: name,
			InitialClusterVersion: version,
			InitialNodeCount:      config.NumNodes,
			NodeConfig: &container.NodeConfig{
				MachineType: config.MachineType,
			},
			EnableKubernetesAlpha: config.EnableKubernetesAlpha,
		},
	}

	op, err := cc.gkeService.Projects.Zones.Clusters.Create(project, config.Zone, clusterRequest).Do()
	if err != nil {
		return nil, err
	}
	if err = cc.checkGKEOperation(project, config.Zone, op.Name); err != nil {
		return nil, err
	}
	return &InstanceInfo{Name: name}, nil
}

func newComputeInstance(config GCEVMConfig, project, name string) *compute.Instance {
	// Inconsistency between compute and container APIs
	machineType := fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", project, config.Zone, config.MachineType)
	instance := &compute.Instance{
		Name:         name,
		Zone:         config.Zone,
		MachineType:  machineType,
		CanIpForward: true,
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       persistent,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskName:    name,
					SourceImage: config.SourceImage,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Name: "External NAT",
						Type: oneToOneNAT,
					},
				},
				Subnetwork: fmt.Sprintf("projects/%s/regions/%s/subnetworks/default", project, config.Zone),
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email:  "default",
				Scopes: config.Scopes,
			},
		},
	}
	if config.Tags != nil {
		instance.Tags = &compute.Tags{Items: config.Tags}
	}
	return instance
}

func (cc *client) createVM(project string, config GCEVMConfig) (*InstanceInfo, error) {
	name := generateName("gce")
	instance := newComputeInstance(config, project, name)
	call := cc.gceService.Instances.Insert(project, config.Zone, instance)
	op, err := call.Do()
	if err != nil {
		return nil, err
	}
	if err = cc.checkGCEOperation(project, config.Zone, op.Name); err != nil {
		return nil, err
	}
	return &InstanceInfo{Name: name}, nil
}
