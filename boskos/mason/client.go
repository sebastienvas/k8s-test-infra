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

package mason

import (
	"fmt"
	"sync"

	"k8s.io/test-infra/boskos/common"

	"github.com/deckarep/golang-set"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

type BoskosClient interface {
	Acquire(rtype, state, dest string) (*common.Resource, error)
	ReleaseOne(name, dest string) error
	UpdateOne(name, state string, userData common.UserData) error
	AcquireByState(state, dest string) ([]common.Resource, error)
}

type Client struct {
	basic BoskosClient
	// Main resources
	resources map[string]common.Resource
	// Resources leased my main resources, key is main resource name
	leasedResources map[string][]common.Resource
	lock            sync.RWMutex
}

func NewClient(boskosClient BoskosClient) *Client {
	return &Client{
		basic:           boskosClient,
		resources:       map[string]common.Resource{},
		leasedResources: map[string][]common.Resource{},
	}
}

func (c *Client) AcquireLeasedResource(name, rtype, state, dest string) (*common.Resource, error) {
	res, err := c.basic.Acquire(rtype, state, dest)
	if err == nil {
		c.updateResources(name, *res)
	}
	return res, err
}

func (c *Client) AcquireLeasedResources(rtype, state, dest string) (*common.Resource, []common.Resource, error) {
	res, err := c.basic.Acquire(rtype, state, dest)
	if err != nil {
		return nil, nil, err
	}
	c.updateResource(*res)
	var leasedResources common.LeasedResources
	if err := res.UserData.Extract(LeasedResources, &leasedResources); err != nil {
		if _, ok := err.(*common.UserDataNotFound); !ok {
			logrus.WithError(err).Errorf("cannot parse %s from User Data", LeasedResources)
			return res, nil, err
		}
	}
	if len(leasedResources) == 0 {
		return res, nil, nil
	}
	resources, err := c.basic.AcquireByState(res.Name, dest)
	if err != nil {
		return res, nil, err
	}
	c.updateResources(res.Name, resources...)
	leasedResourcesSet := mapset.NewSet()
	for _, name := range leasedResources {
		leasedResourcesSet.Add(name)
	}
	resourcesByStateSet := mapset.NewSet()
	for _, r := range resources {
		resourcesByStateSet.Add(r.Name)
	}
	if !leasedResourcesSet.Intersect(resourcesByStateSet).Equal(leasedResourcesSet) {
		err := fmt.Errorf("resources are is missing")
		logrus.Errorf(err.Error())
		return nil, nil, err
	}
	return res, resources, nil
}

func (c *Client) ReleaseLeasedResources(name, dest string) error {
	var allErrors error
	res, resources := c.getResources(name)
	if res == nil {
		return fmt.Errorf("resource %s could not be found", name)
	}
	if err := c.basic.ReleaseOne(res.Name, dest); err != nil {
		allErrors = multierror.Append(allErrors, err)
	}
	for _, r := range resources {
		err := c.basic.ReleaseOne(r.Name, res.Name)
		if err != nil {
			allErrors = multierror.Append(allErrors, err)
		}
		logrus.Infof("resource %s has been released to %s", r.Name, res.Name)
	}
	return allErrors
}

func (c *Client) ReleaseOne(name, dest string) error {
	return c.basic.ReleaseOne(name, dest)
}

func (c *Client) UpdateOne(name, state string, userData common.UserData) error {
	return c.basic.UpdateOne(name, state, userData)
}

func (c *Client) updateResource(r common.Resource) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.resources[r.Name] = r
}

func (c *Client) updateResources(name string, resources ...common.Resource) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.leasedResources[name] = resources
}

func (c *Client) getResources(name string) (*common.Resource, []common.Resource) {
	c.lock.Lock()
	defer c.lock.Unlock()
	res, ok := c.resources[name]
	if !ok {
		return nil, nil
	}
	resources := c.leasedResources[name]
	delete(c.resources, name)
	delete(c.leasedResources, name)
	return &res, resources
}
