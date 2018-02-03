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

// Client extends boskos client with support with resource with leased resources
type Client struct {
	basic     boskosClient
	resources map[string]common.Resource
	lock      sync.RWMutex
}

// NewClient creates a new client from a boskosClient interface
func NewClient(boskosClient boskosClient) *Client {
	return &Client{
		basic:     boskosClient,
		resources: map[string]common.Resource{},
	}
}

// Acquire gets a resource with associated leased resources
func (c *Client) Acquire(rtype, state, dest string) (*common.Resource, error) {
	var resourcesToRelease []common.Resource
	releaseOnFailure := func() {
		for _, r := range resourcesToRelease {
			if err := c.basic.ReleaseOne(r.Name, common.Dirty); err != nil {
				logrus.WithError(err).Warning("failed to release resource %s", r.Name)
			}
		}
	}
	res, err := c.basic.Acquire(rtype, state, dest)
	if err != nil {
		return nil, err
	}
	resourcesToRelease = append(resourcesToRelease, *res)
	resources, err := c.basic.AcquireByState(res.Name, dest)
	if err != nil {
		releaseOnFailure()
		return nil, err
	}
	resourcesToRelease = append(resourcesToRelease, resources...)
	if err := c.validateResource(res, resources); err != nil {
		releaseOnFailure()
		return nil, err
	}
	c.updateResource(*res)
	return res, nil
}

// validateResource is helpful to see if some leased resources are missing,
// meaning that the main resource using them is in bad state.
func (c *Client) validateResource(res *common.Resource, resources []common.Resource) error {
	var leasedResources common.LeasedResources
	if err := res.UserData.Extract(LeasedResources, &leasedResources); err != nil {
		if _, ok := err.(*common.UserDataNotFound); !ok {
			logrus.WithError(err).Errorf("cannot parse %s from User Data", LeasedResources)
			return err
		}
	}
	if len(leasedResources) != len(resources) {
		return fmt.Errorf("resource aquired (%d) does not match expected resources (%d)",
			len(resources), len(leasedResources))
	}
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
		return err
	}
	return nil
}

// ReleaseOne will release a resource as well as leased resources associated to it
func (c *Client) ReleaseOne(name, dest string) (allErrors error) {
	res := c.getResource(name)
	resourceNames := []string{name}
	var leasedResources common.LeasedResources
	if err := res.UserData.Extract(LeasedResources, &leasedResources); err != nil {
		if _, ok := err.(*common.UserDataNotFound); !ok {
			logrus.WithError(err).Errorf("cannot parse %s from User Data", LeasedResources)
			allErrors = multierror.Append(allErrors, err)
			if err := c.basic.ReleaseOne(name, dest); err != nil {
				logrus.WithError(err).Warning("failed to release resource %s", name)
				allErrors = multierror.Append(allErrors, err)
			}
			return
		}
	}
	resourceNames = append(resourceNames, leasedResources...)
	for _, n := range resourceNames {
		if err := c.basic.ReleaseOne(n, dest); err != nil {
			logrus.WithError(err).Warning("failed to release resource %s", n)
			allErrors = multierror.Append(allErrors, err)
		}
	}
	c.deleteResource(name)
	return
}

func (c *Client) updateResource(r common.Resource) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.resources[r.Name] = r
}

func (c *Client) getResource(name string) *common.Resource {
	c.lock.Lock()
	defer c.lock.Unlock()
	res, ok := c.resources[name]
	if !ok {
		return nil
	}
	return &res
}

func (c *Client) deleteResource(name string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.resources, name)
}
