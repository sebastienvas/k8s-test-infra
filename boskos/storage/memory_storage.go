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
)

type MemoryStorage struct {
	items map[string]common.Item
	lock  sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		items: map[string]common.Item{},
	}
}

func (im *MemoryStorage) Add(i common.Item) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	im.items[i.GetName()] = i
	return nil
}

func (im *MemoryStorage) Delete(name string) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	_, ok := im.items[name]
	if !ok {
		return fmt.Errorf("cannot find item %s", name)
	}
	delete(im.items, name)
	return nil
}

func (im *MemoryStorage) Update(i common.Item) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	_, ok := im.items[i.GetName()]
	if !ok {
		return fmt.Errorf("cannot find item %s", i.GetName())
	}
	im.items[i.GetName()] = i
	return nil
}

func (im *MemoryStorage) Get(name string) (common.Item, error) {
	im.lock.Lock()
	defer im.lock.Unlock()
	i, ok := im.items[name]
	if !ok {
		return nil, fmt.Errorf("cannot find item %s", name)
	}
	return i, nil
}

func (im *MemoryStorage) List() ([]common.Item, error) {
	im.lock.Lock()
	defer im.lock.Unlock()
	var items []common.Item
	for _, i := range im.items {
		items = append(items, i)
	}
	return items, nil
}
