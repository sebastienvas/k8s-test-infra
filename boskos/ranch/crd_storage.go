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

package ranch

import (
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/crds"
)

type CRDStorage struct {
	resourceClient crds.CRDClientInterface
	configClient   crds.CRDClientInterface
}

func NewCRDStorage(resourceClient, configClient crds.CRDClientInterface) *CRDStorage {
	return &CRDStorage{
		resourceClient: resourceClient,
		configClient:   configClient,
	}
}

func (cs *CRDStorage) AddResource(resource common.Resource) error {
	var r crds.Resource
	r.FromResource(resource)
	if _, err := cs.resourceClient.Create(&r); err != nil {
		return err
	}
	return nil
}

func (cs *CRDStorage) DeleteResource(name string) error {
	if err := cs.resourceClient.Delete(name, v1.NewDeleteOptions(DeleteGracePeriodSeconds)); err != nil {
		return err
	}
	return nil
}

func (cs *CRDStorage) UpdateResource(resource common.Resource) error {
	o, err := cs.resourceClient.Get(resource.Name)
	if err != nil {
		return err
	}
	var r *crds.Resource
	r, err = crds.RuntimeObjectToResource(o)
	if err != nil {
		return err
	}
	r.FromResource(resource)
	if _, err = cs.resourceClient.Update(r); err != nil {
		return err
	}
	return nil
}

func (cs *CRDStorage) GetResource(name string) (common.Resource, error) {
	o, err := cs.resourceClient.Get(name)
	if err != nil {
		return common.Resource{}, err
	}
	var r *crds.Resource
	r, err = crds.RuntimeObjectToResource(o)
	if err != nil {
		return common.Resource{}, err
	}
	return r.ToResource(), nil
}

func (cs *CRDStorage) GetResources() ([]common.Resource, error) {
	var resources []common.Resource
	o, err := cs.resourceClient.List(v1.ListOptions{})
	if err != nil {
		return resources, err
	}
	var l *crds.ResourceList
	l, err = crds.RuntimeObjectToResourceList(o)
	if err != nil {
		return resources, err
	}
	for _, r := range l.Items {
		resources = append(resources, r.ToResource())
	}
	sort.Stable(ByUpdateTime(resources))
	return resources, nil
}

func (cs *CRDStorage) AddConfig(config common.ResourceConfig) error {
	var r crds.ResourceConfig
	r.FromResource(config)
	if _, err := cs.configClient.Create(&r); err != nil {
		return err
	}
	return nil
}

func (cs *CRDStorage) DeleteConfig(name string) error {
	if err := cs.configClient.Delete(name, v1.NewDeleteOptions(DeleteGracePeriodSeconds)); err != nil {
		return err
	}
	return nil
}

func (cs *CRDStorage) UpdateConfig(config common.ResourceConfig) error {
	o, err := cs.configClient.Get(config.Name)
	if err != nil {
		return err
	}
	var r *crds.ResourceConfig
	r, err = crds.RuntimeObjectToResourceConfig(o)
	if err != nil {
		return err
	}
	r.FromResource(config)
	if _, err = cs.configClient.Update(r); err != nil {
		return err
	}
	return nil
}

func (cs *CRDStorage) GetConfig(name string) (common.ResourceConfig, error) {
	o, err := cs.configClient.Get(name)
	if err != nil {
		return common.ResourceConfig{}, err
	}
	var r *crds.ResourceConfig
	r, err = crds.RuntimeObjectToResourceConfig(o)
	if err != nil {
		return common.ResourceConfig{}, err
	}
	return r.ToResourceConfig(), nil
}

func (cs *CRDStorage) GetConfigs() ([]common.ResourceConfig, error) {
	var configs []common.ResourceConfig
	o, err := cs.configClient.List(v1.ListOptions{})
	if err != nil {
		return configs, err
	}
	var l *crds.ResourceConfigList
	l, err = crds.RuntimeObjectToResourceConfigList(o)
	if err != nil {
		return configs, err
	}

	for _, r := range l.Items {
		configs = append(configs, r.ToResourceConfig())
	}
	return configs, nil
}
