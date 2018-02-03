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

package storage

import (
	"sync"

	"fmt"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/crds"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	deleteGracePeriodSeconds = 10
)

// PersistenceLayer defines a simple interface to persists Boskos Information
type PersistenceLayer interface {
	Add(i common.Item) error
	Delete(name string) error
	Update(i common.Item) error
	Get(name string) (common.Item, error)
	List() ([]common.Item, error)
}

type inClusterStorage struct {
	client crds.ClientInterface
}


type inMemoryStore struct {
	items map[string]common.Item
	lock  sync.RWMutex
}

// NewMemoryStorage creates an in memory persistence layer
func NewMemoryStorage() PersistenceLayer {
	return &inMemoryStore{
		items: map[string]common.Item{},
	}
}

// NewCRDStorage creates a Custom Resource Definition persistence layer
func NewCRDStorage(client crds.ClientInterface) PersistenceLayer {
	return &inClusterStorage{
		client: client,
	}
}

func (im *inMemoryStore) Add(i common.Item) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	im.items[i.GetName()] = i
	return nil
}

func (im *inMemoryStore) Delete(name string) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	_, ok := im.items[name]
	if !ok {
		return fmt.Errorf("cannot find item %s", name)
	}
	delete(im.items, name)
	return nil
}

func (im *inMemoryStore) Update(i common.Item) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	_, ok := im.items[i.GetName()]
	if !ok {
		return fmt.Errorf("cannot find item %s", i.GetName())
	}
	im.items[i.GetName()] = i
	return nil
}

func (im *inMemoryStore) Get(name string) (common.Item, error) {
	im.lock.Lock()
	defer im.lock.Unlock()
	i, ok := im.items[name]
	if !ok {
		return nil, fmt.Errorf("cannot find item %s", name)
	}
	return i, nil
}

func (im *inMemoryStore) List() ([]common.Item, error) {
	im.lock.Lock()
	defer im.lock.Unlock()
	var items []common.Item
	for _, i := range im.items {
		items = append(items, i)
	}
	return items, nil
}

func (cs *inClusterStorage) Add(i common.Item) error {
	o := cs.client.NewObject()
	o.FromItem(i)
	_, err := cs.client.Create(o)
	return err
}

func (cs *inClusterStorage) Delete(name string) error {
	return cs.client.Delete(name, v1.NewDeleteOptions(deleteGracePeriodSeconds))
}

func (cs *inClusterStorage) Update(i common.Item) error {
	o, err := cs.client.Get(i.GetName())
	if err != nil {
		return err
	}
	o.FromItem(i)
	_, err = cs.client.Update(o)
	return err
}

func (cs *inClusterStorage) Get(name string) (common.Item, error) {
	o, err := cs.client.Get(name)
	if err != nil {
		return nil, err
	}
	return o.ToItem(), nil

}

func (cs *inClusterStorage) List() ([]common.Item, error) {
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
