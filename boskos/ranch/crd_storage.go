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
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/crds"
)

type CRDStorage struct {
	client crds.ClientInterface
}

func NewCRDStorage(client crds.ClientInterface) *CRDStorage {
	return &CRDStorage{
		client: client,
	}
}

func (cs *CRDStorage) Add(i common.Item) error {
	o := cs.client.NewObject()
	o.FromItem(i)
	_, err := cs.client.Create(o)
	return err
}

func (cs *CRDStorage) Delete(name string) error {
	return cs.client.Delete(name, v1.NewDeleteOptions(DeleteGracePeriodSeconds))
}

func (cs *CRDStorage) Update(i common.Item) error {
	o, err := cs.client.Get(i.GetName())
	if err != nil {
		return err
	}
	o.FromItem(i)
	_, err = cs.client.Update(o)
	return err
}

func (cs *CRDStorage) Get(name string) (common.Item, error) {
	o, err := cs.client.Get(name)
	if err != nil {
		return nil, err
	}
	return o.ToItem(), nil

}

func (cs *CRDStorage) List() ([]common.Item, error) {
	col, err := cs.client.List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var items []common.Item
	for _, i := range col.GetItems() {
		items = append(items, i.ToItem())
	}
	return items, nil
}
