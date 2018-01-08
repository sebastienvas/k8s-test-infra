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
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/common"
)

type Item interface {
	GetName() string
}

type mapStorage struct {
	items    map[string]Item
	notFound func(name string) error
	lock     sync.RWMutex
}

func (im *mapStorage) Add(i Item) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	im.items[i.GetName()] = i
	return nil
}

func (im *mapStorage) Delete(name string) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	_, ok := im.items[name]
	if !ok {
		return im.notFound(name)
	}
	delete(im.items, name)
	return nil
}

func (im *mapStorage) Update(i Item) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	_, ok := im.items[i.GetName()]
	if !ok {
		return im.notFound(i.GetName())
	}
	im.items[i.GetName()] = i
	return nil
}

func (im *mapStorage) Get(name string) (Item, error) {
	im.lock.Lock()
	defer im.lock.Unlock()
	i, ok := im.items[name]
	if !ok {
		return nil, im.notFound(name)
	}
	return i, nil
}

func (im *mapStorage) List() ([]Item, error) {
	im.lock.Lock()
	defer im.lock.Unlock()
	var items []Item
	for _, i := range im.items {
		items = append(items, i)
	}
	return items, nil
}

type InMemStorage struct {
	resources mapStorage
	configs   mapStorage
}

func itemToResource(i interface{}) (common.Resource, error) {
	res, ok := i.(common.Resource)
	if !ok {
		return common.Resource{}, fmt.Errorf("cannot construct Resource from received object %v", i)
	}
	return res, nil
}

func itemToResourceConfig(i interface{}) (common.ResourceConfig, error) {
	conf, ok := i.(common.ResourceConfig)
	if !ok {
		return common.ResourceConfig{}, fmt.Errorf("cannot construct Resource from received object %v", i)
	}
	return conf, nil
}

func NewInMemStorage(storage string) (*InMemStorage, error) {
	im := &InMemStorage{
		resources: mapStorage{
			items:    map[string]Item{},
			notFound: func(name string) error { return &ResourceNotFound{name} },
		},
		configs: mapStorage{
			items:    map[string]Item{},
			notFound: func(name string) error { return &ResourceConfigNotFound{name} },
		},
	}
	var data struct {
		resources []common.Resource
	}
	if storage != "" {
		buf, err := ioutil.ReadFile(storage)
		if err == nil {
			logrus.Infof("Current state: %v.", buf)
			err = json.Unmarshal(buf, data)
			if err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	for _, res := range data.resources {
		im.AddResource(res)
	}
	return im, nil
}

func (im *InMemStorage) AddResource(resource common.Resource) error {
	return im.resources.Add(resource)
}

func (im *InMemStorage) DeleteResource(name string) error {
	return im.resources.Delete(name)
}

func (im *InMemStorage) UpdateResource(resource common.Resource) error {
	return im.resources.Update(resource)
}

func (im *InMemStorage) GetResource(name string) (common.Resource, error) {
	i, err := im.resources.Get(name)
	if err != nil {
		return common.Resource{}, err
	}
	var res common.Resource
	res, err = itemToResource(i)
	if err != nil {
		return common.Resource{}, err
	}
	return res, nil
}

func (im *InMemStorage) GetResources() ([]common.Resource, error) {
	var resources []common.Resource
	items, err := im.resources.List()
	if err != nil {
		return resources, err
	}
	for _, i := range items {
		var res common.Resource
		res, err = itemToResource(i)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res)
	}
	sort.Stable(ByUpdateTime(resources))
	return resources, nil
}

func (im *InMemStorage) AddConfig(conf common.ResourceConfig) error {
	return im.configs.Add(conf)
}

func (im *InMemStorage) DeleteConfig(name string) error {
	return im.configs.Delete(name)
}

func (im *InMemStorage) UpdateConfig(conf common.ResourceConfig) error {
	return im.configs.Update(conf)
}

func (im *InMemStorage) GetConfig(name string) (common.ResourceConfig, error) {
	i, err := im.configs.Get(name)
	if err != nil {
		return common.ResourceConfig{}, err
	}
	var conf common.ResourceConfig
	conf, err = itemToResourceConfig(i)
	if err != nil {
		return common.ResourceConfig{}, err
	}
	return conf, nil
}

func (im *InMemStorage) GetConfigs() ([]common.ResourceConfig, error) {
	var configs []common.ResourceConfig
	items, err := im.configs.List()
	if err != nil {
		return configs, err
	}
	for _, i := range items {
		var conf common.ResourceConfig
		conf, err = itemToResourceConfig(i)
		if err != nil {
			return nil, err
		}
		configs = append(configs, conf)
	}
	return configs, nil
}
