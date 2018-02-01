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

type ResourcesConfig struct {
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

func (rc *ResourcesConfig) Construct(res *common.Resource, types common.TypeToResources) (common.UserData, error) {
	info := ResourceInfo{}
	var err error

	gcpClient, err := newClient()
	if err != nil {
		return nil, err
	}
	// Copy
	typesCopy := types

	popProject := func(rType string) *common.Resource {
		if len(typesCopy[rType]) == 0 {
			return nil
		}
		r := typesCopy[rType][len(typesCopy[rType]) -1]
		typesCopy[rType] = typesCopy[rType][:len(typesCopy[rType])-1]
		return r
	}

	// Here we know that resources are of project type
	for _, pc := range rc.ProjectConfigs {
		project := popProject(pc.Type)
		if project == nil {
			err := fmt.Errorf("running out of project while creating resources")
			logrus.WithError(err).Errorf("unable to create resources")
			return nil, err
		}
		projectInfo := ProjectInfo{Name: project.Name}
		for _, cl := range pc.Clusters {
			var clusterInfo *InstanceInfo
			clusterInfo, err = gcpClient.createCluster(project.Name, cl)
			if err != nil {
				logrus.WithError(err).Errorf("unable to create cluster on project %s", project.Name)
				return nil, err
			}
			projectInfo.Clusters = append(projectInfo.Clusters, *clusterInfo)
		}
		for _, vm := range pc.Vms {
			var vmInfo *InstanceInfo
			vmInfo, err = gcpClient.createVM(project.Name, vm)
			if err != nil {
				logrus.WithError(err).Errorf("unable to create vm on project %s", project.Name)
				return nil, err
			}
			projectInfo.VMs = append(projectInfo.VMs, *vmInfo)
		}
		info.ProjectsInfo = append(info.ProjectsInfo, projectInfo)
	}
	userData := common.UserData{}
	if err := userData.Set(ResourceConfigType, &info); err != nil {
		logrus.WithError(err).Errorf("unable to set %s user data", ResourceConfigType)
		return nil, err
	}
	return userData, nil
}

func (rc *ResourcesConfig) GetName() string {
	return ResourceConfigType
}

func configConverter(in string) (*ResourcesConfig, error) {
	var config ResourcesConfig
	if err := yaml.Unmarshal([]byte(in), &config); err != nil {
		logrus.WithError(err).Errorf("unable to parse %s", in)
		return nil, err
	}
	return &config, nil
}

func ConfigConverter(in string) (mason.Masonable, error) {
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
	return fmt.Sprintf("%s-%s", prefix, time.Now().Format("0102150405"))
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
	logrus.Infof("Instance %s created via operation %s", clusterRequest.Cluster.Name, op.Name)
	return &InstanceInfo{Name: name}, nil
}

func newComputeInstance(config GCEVMConfig, project, name string) *compute.Instance {
	// Inconsistency between compute and container APIs
	machineType := fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", project, config.Zone, config.MachineType)
	zone := fmt.Sprintf("projects/%s/zones/%s", project, config.Zone)
	instance := &compute.Instance{
		Name:         name,
		Zone:         zone,
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
	logrus.Infof("Instance %s created via operation %s", instance.Name, op.Name)
	return &InstanceInfo{Name: name}, nil
}
