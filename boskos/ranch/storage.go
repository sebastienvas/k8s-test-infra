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
	"encoding/json"
	"io/ioutil"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/common"
)

type StorageInterface interface {
	Add(i common.Item) error
	Delete(name string) error
	Update(i common.Item) error
	Get(name string) (common.Item, error)
	List() ([]common.Item, error)
}

type Storage struct {
	resources StorageInterface
	configs   StorageInterface
}

func NewStorage(c, r StorageInterface, storage string) (*Storage, error) {
	s := &Storage{
		resources: r,
		configs:   c,
	}

	if storage != "" {
		var data struct {
			resources []common.Resource
		}
		buf, err := ioutil.ReadFile(storage)
		if err == nil {
			logrus.Infof("Current state: %v.", buf)
			err = json.Unmarshal(buf, &data)
			if err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		for _, res := range data.resources {
			if err := s.AddResource(res); err != nil {
				return nil, err
			}
		}
	}
	return s, nil
}

func (s *Storage) AddResource(resource common.Resource) error {
	return s.resources.Add(resource)
}

func (s *Storage) DeleteResource(name string) error {
	return s.resources.Delete(name)
}

func (s *Storage) UpdateResource(resource common.Resource) error {
	return s.resources.Update(resource)
}

func (s *Storage) GetResource(name string) (common.Resource, error) {
	i, err := s.resources.Get(name)
	if err != nil {
		return common.Resource{}, err
	}
	var res common.Resource
	res, err = common.ItemToResource(i)
	if err != nil {
		return common.Resource{}, err
	}
	return res, nil
}

func (s *Storage) GetResources() ([]common.Resource, error) {
	var resources []common.Resource
	items, err := s.resources.List()
	if err != nil {
		return resources, err
	}
	for _, i := range items {
		var res common.Resource
		res, err = common.ItemToResource(i)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res)
	}
	sort.Stable(ResourceByUpdateTime(resources))
	return resources, nil
}

func (s *Storage) AddConfig(conf common.ResourceConfig) error {
	return s.configs.Add(conf)
}

func (s *Storage) DeleteConfig(name string) error {
	return s.configs.Delete(name)
}

func (s *Storage) UpdateConfig(conf common.ResourceConfig) error {
	return s.configs.Update(conf)
}

func (s *Storage) GetConfig(name string) (common.ResourceConfig, error) {
	i, err := s.configs.Get(name)
	if err != nil {
		return common.ResourceConfig{}, err
	}
	var conf common.ResourceConfig
	conf, err = common.ItemToResourceConfig(i)
	if err != nil {
		return common.ResourceConfig{}, err
	}
	return conf, nil
}

func (s *Storage) GetConfigs() ([]common.ResourceConfig, error) {
	var configs []common.ResourceConfig
	items, err := s.configs.List()
	if err != nil {
		return configs, err
	}
	for _, i := range items {
		var conf common.ResourceConfig
		conf, err = common.ItemToResourceConfig(i)
		if err != nil {
			return nil, err
		}
		configs = append(configs, conf)
	}
	return configs, nil
}
