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
	"reflect"
	"time"

	"github.com/deckarep/golang-set"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/test-infra/boskos/common"
	"sync"
)

const (
	DeleteGracePeriodSeconds = 10
)

// Ranch is the place which all of the Resource objects lives.
type Ranch struct {
	Storage       StorageInterface
	resourcesLock sync.RWMutex
	configsLock   sync.RWMutex
}

type StorageInterface interface {
	AddResource(config common.Resource) error
	DeleteResource(string) error
	UpdateResource(common.Resource) error
	GetResource(string) (common.Resource, error)
	GetResources() ([]common.Resource, error)
	AddConfig(common.ResourceConfig) error
	DeleteConfig(string) error
	UpdateConfig(common.ResourceConfig) error
	GetConfig(string) (common.ResourceConfig, error)
	GetConfigs() ([]common.ResourceConfig, error)
}

type ByUpdateTime []common.Resource

func (ut ByUpdateTime) Len() int           { return len(ut) }
func (ut ByUpdateTime) Swap(i, j int)      { ut[i], ut[j] = ut[j], ut[i] }
func (ut ByUpdateTime) Less(i, j int) bool { return ut[i].LastUpdate.Before(ut[j].LastUpdate) }

type ByName []common.Resource

func (ut ByName) Len() int           { return len(ut) }
func (ut ByName) Swap(i, j int)      { ut[i], ut[j] = ut[j], ut[i] }
func (ut ByName) Less(i, j int) bool { return ut[i].Name < ut[j].Name }

// Public errors:

// OwnerNotMatch will be returned if request owner does not match current owner for target resource.
type OwnerNotMatch struct {
	request string
	owner   string
}

func (o OwnerNotMatch) Error() string {
	return fmt.Sprintf("OwnerNotMatch - request by %s, currently owned by %s", o.request, o.owner)
}

// ResourceNotFound will be returned if requested resource does not exist.
type ResourceNotFound struct {
	name string
}

func (r ResourceNotFound) Error() string {
	return fmt.Sprintf("Resource %s not exist", r.name)
}

// ResourceConfigNotFound will be returned if requested config does not exist.
type ResourceConfigNotFound struct {
	name string
}

func (r ResourceConfigNotFound) Error() string {
	return fmt.Sprintf("ResourceConfig %s not exist", r.name)
}

// StateNotMatch will be returned if requested state does not match current state for target resource.
type StateNotMatch struct {
	expect  string
	current string
}

func (s StateNotMatch) Error() string {
	return fmt.Sprintf("StateNotMatch - expect %v, current %v", s.expect, s.current)
}

// NewRanch creates a new Ranch object.
// In: config - path to resource file
//     storage - path to where to save/restore the state data
// Out: A Ranch object, loaded from config/storage, or error
func NewRanch(config string, storage StorageInterface) (*Ranch, error) {

	newRanch := &Ranch{
		Storage:       storage,
		configsLock:   sync.RWMutex{},
		resourcesLock: sync.RWMutex{},
	}

	if config != "" {
		if err := newRanch.SyncConfig(config); err != nil {
			return nil, err
		}
	}

	newRanch.LogStatus()

	return newRanch, nil
}

func (r *Ranch) updateResourceAndLeasedResources(res common.Resource) error {
	if err := r.Storage.UpdateResource(res); err != nil {
		logrus.WithError(err).Errorf("could not update resource %s", res.Name)
		return err
	}

	for _, rName := range res.Info.LeasedResources {
		lr, err := r.Storage.GetResource(rName)
		if err != nil {
			logrus.WithError(err).Errorf("could not find leased resource %s", rName)
			return err
		}
		res.LastUpdate = time.Now()
		if err := r.Storage.UpdateResource(lr); err != nil {
			logrus.WithError(err).Errorf("could not update leased resource %s", rName)
			return err
		}
	}
	return nil
}

// Acquire checks out a type of resource in certain state without an owner,
// and move the checked out resource to the end of the resource list.
// In: rtype - name of the target resource
//     state - current state of the requested resource
//     dest - destination state of the requested resource
//     owner - requester of the resource
// Out: A valid Resource object on success, or
//      ResourceNotFound error if target type resource does not exist in target state.
func (r *Ranch) Acquire(rType, state, dest, owner string) (*common.Resource, error) {
	r.resourcesLock.Lock()
	defer r.resourcesLock.Unlock()

	resources, err := r.Storage.GetResources()
	if err != nil {
		logrus.WithError(err).Errorf("could not get resources")
		return nil, &ResourceNotFound{rType}
	}

	for idx := range resources {
		res := resources[idx]
		if rType == res.Type && state == res.State && res.Owner == "" {
			res.LastUpdate = time.Now()
			res.Owner = owner
			res.State = dest
			if err = r.updateResourceAndLeasedResources(res); err != nil {
				return nil, err
			}
			return &res, nil
		}
	}
	return nil, &ResourceNotFound{rType}
}

// GetConfig returns the StructInfo for a given config name.
// In: name - name of the target config
// Out:
//   A valid  StructInfo object on success, or
//   ResourceConfigNotFound error if target named config does not exist.
func (r *Ranch) GetConfig(name string) (*common.ResourceConfig, error) {
	r.configsLock.Lock()
	defer r.configsLock.Unlock()

	conf, err := r.Storage.GetConfig(name)
	if err != nil {
		logrus.WithError(err).Errorf("Unable to get config for %s", name)
		return nil, ResourceConfigNotFound{name}
	}
	return &conf, nil
}

// Release unsets owner for target resource and move it to a new state.
// In: name - name of the target resource
//     dest - destination state of the resource
//     owner - owner of the resource
// Out: nil on success, or
//      OwnerNotMatch error if owner does not match current owner of the resource, or
//      ResourceNotFound error if target named resource does not exist.
func (r *Ranch) Release(name, dest, owner string) error {
	r.resourcesLock.Lock()
	defer r.resourcesLock.Unlock()

	res, err := r.Storage.GetResource(name)
	if err != nil {
		logrus.WithError(err).Errorf("unable to release resource %s", name)
		return &ResourceNotFound{name}
	}
	if owner != res.Owner {
		return &OwnerNotMatch{res.Owner, owner}
	}
	res.LastUpdate = time.Now()
	res.Owner = ""
	res.State = dest
	return r.updateResourceAndLeasedResources(res)
}

// Update updates the timestamp of a target resource.
// In: name  - name of the target resource
//     state - current state of the resource
//     owner - current owner of the resource
// 	   info  - information on how to use the resource
// Out: nil on success, or
//      OwnerNotMatch error if owner does not match current owner of the resource, or
//      ResourceNotFound error if target named resource does not exist, or
//      StateNotMatch error if state does not match current state of the resource.
func (r *Ranch) Update(name, owner, state string, info *common.ResourceInfo) error {
	r.resourcesLock.Lock()
	defer r.resourcesLock.Unlock()

	res, err := r.Storage.GetResource(name)
	if err != nil {
		logrus.WithError(err).Errorf("could not find resource %s for update", name)
		return &ResourceNotFound{name}
	}
	if owner != res.Owner {
		return &OwnerNotMatch{res.Owner, owner}
	}
	if state != res.State {
		return &StateNotMatch{res.State, state}
	}
	if info != nil {
		res.Info = *info
	}
	res.LastUpdate = time.Now()
	return r.updateResourceAndLeasedResources(res)
}

// Reset unstucks a type of stale resource to a new state.
// In: rtype - type of the resource
//     state - current state of the resource
//     expire - duration before resource's last update
//     dest - destination state of expired resources
// Out: map of resource name - resource owner.
func (r *Ranch) Reset(rtype, state string, expire time.Duration, dest string) (map[string]string, error) {
	r.resourcesLock.Lock()
	defer r.resourcesLock.Unlock()

	ret := make(map[string]string)

	resources, err := r.Storage.GetResources()
	if err != nil {
		logrus.WithError(err).Errorf("cannot find resources")
		return nil, err
	}

	for idx := range resources {
		res := resources[idx]
		if rtype == res.Type && state == res.State && res.Owner != "" {
			if time.Since(res.LastUpdate) > expire {
				res.LastUpdate = time.Now()
				ret[res.Name] = res.Owner
				res.Owner = ""
				res.State = dest
				if err = r.updateResourceAndLeasedResources(res); err != nil {
					return ret, err
				}
			}
		}
	}
	return ret, nil
}

// LogStatus outputs current status of all resources
func (r *Ranch) LogStatus() {
	resources, err := r.Storage.GetResources()

	if err != nil {
		return
	}

	resJSON, err := json.Marshal(resources)
	if err != nil {
		logrus.WithError(err).Errorf("Fail to marshal Resources. %v", resources)
	}
	logrus.Infof("Current Resources : %v", string(resJSON))
}

// SyncConfig updates resource list from a file
func (r *Ranch) SyncConfig(config string) error {
	resources, configs, err := ParseConfig(config)
	if err != nil {
		return err
	}
	if err := r.syncResources(resources); err != nil {
		return err
	}
	if err := r.syncConfigs(configs); err != nil {
		return err
	}
	return nil
}

// ParseConfig reads in configPath and returns a list of resource objects
// on success.
func ParseConfig(configPath string) ([]common.Resource, []common.ResourceConfig, error) {
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, nil, err
	}

	var data common.BoskosConfig
	err = yaml.Unmarshal(file, &data)
	if err != nil {
		return nil, nil, err
	}

	resourcesNeeds := map[string]int{}
	actualResources := map[string]int{}

	configNames := map[string]map[string]int{}
	configs := data.Configs
	for _, c := range configs {
		configNames[c.Name] = c.Needs
	}

	var resources []common.Resource
	for _, res := range data.Resources {
		if res.UseConfig {
			c, ok := configNames[res.Type]
			if !ok {
				err := fmt.Errorf("resource type %s does not have associated config", res.Type)
				logrus.WithError(err).Error("using useconfig implies associated config")
				return nil, nil, err
			}
			// Updating resourceNeeds
			for k, v := range c {
				resourcesNeeds[k] += v
			}
		}
		for _, name := range res.Names {
			resources = append(resources, common.Resource{
				Type:      res.Type,
				State:     res.State,
				Name:      name,
				UseConfig: res.UseConfig,
			})
		}
		actualResources[res.Type] = len(res.Names)
	}

	for rType, needs := range resourcesNeeds {
		actual, ok := actualResources[rType]
		if !ok {
			err := fmt.Errorf("need for resource %s that does not exist", rType)
			logrus.WithError(err).Errorf("invalid configuration")
			return nil, nil, err
		}
		if needs > actual {
			err := fmt.Errorf("not enough resource of type %s for provisioning", rType)
			logrus.WithError(err).Errorf("invalid configuration")
			return nil, nil, err
		}
	}
	return resources, configs, nil
}

func (r *Ranch) syncConfigs(newConfigs []common.ResourceConfig) error {
	r.configsLock.Lock()
	defer r.configsLock.Unlock()

	currentConfigs, err := r.Storage.GetConfigs()
	if err != nil {
		logrus.WithError(err).Error("cannot find configs")
		return err
	}

	currentSet := mapset.NewSet()
	newSet := mapset.NewSet()
	toUpdate := mapset.NewSet()

	configs := map[string]common.ResourceConfig{}

	for _, c := range currentConfigs {
		currentSet.Add(c.Name)
		configs[c.Name] = c
	}

	for _, c := range newConfigs {
		newSet.Add(c.Name)
		if old, exists := configs[c.Name]; exists {
			if !reflect.DeepEqual(old, c) {
				toUpdate.Add(c.Name)
				configs[c.Name] = c
			}
		} else {
			configs[c.Name] = c
		}
	}

	var finalError error

	toDelete := currentSet.Difference(newSet)
	toAdd := newSet.Difference(currentSet)

	for _, n := range toDelete.ToSlice() {
		if err := r.Storage.DeleteConfig(n.(string)); err != nil {
			logrus.WithError(err).Errorf("failed to delete config %s", n)
			finalError = multierror.Append(finalError, err)
		}
	}

	for _, n := range toAdd.ToSlice() {
		rc := configs[n.(string)]
		if err := r.Storage.AddConfig(rc); err != nil {
			logrus.WithError(err).Errorf("failed to create resources %s", n)
			finalError = multierror.Append(finalError, err)
		}
	}

	for _, n := range toUpdate.ToSlice() {
		rc := configs[n.(string)]
		if err := r.Storage.UpdateConfig(rc); err != nil {
			logrus.WithError(err).Errorf("failed to update resources %s", n)
			finalError = multierror.Append(finalError, err)
		}
	}

	return finalError
}

// Boskos resource config will be updated every 10 mins.
// It will append newly added resources to ranch.Resources,
// And try to remove newly deleted resources from ranch.Resources.
// If the newly deleted resource is currently held by a user, the deletion will
// yield to next update cycle.
func (r *Ranch) syncResources(data []common.Resource) error {
	r.resourcesLock.Lock()
	defer r.resourcesLock.Unlock()

	resources, err := r.Storage.GetResources()
	if err != nil {
		logrus.WithError(err).Error("cannot find resources")
		return err
	}

	var finalError error

	// delete non-exist resource
	valid := 0
	for _, res := range resources {
		// If currently busy, yield deletion to later cycles.
		if res.Owner != "" {
			resources[valid] = res
			valid++
			continue
		}
		toDelete := true
		for _, newRes := range data {
			if res.Name == newRes.Name {
				resources[valid] = res
				valid++
				toDelete = false
				break
			}
		}
		if toDelete {
			logrus.Infof("Deleting resource %s", res.Name)
			for _, name := range res.Info.LeasedResources {
				lr, err := r.Storage.GetResource(name)
				if err != nil {
					logrus.WithError(err).Errorf("cannot release resource %s for resource to be deleted %s", name, res.Name)
					return err
				}
				lr.State = common.Dirty
				if err = r.updateResourceAndLeasedResources(lr); err != nil {
					logrus.WithError(err).Errorf("unable to release resource as dirty")
				}
			}
			if err := r.Storage.DeleteResource(res.Name); err != nil {
				finalError = multierror.Append(finalError, err)
				logrus.WithError(err).Errorf("unable to delete resource %s", res.Name)
			}
		}
	}
	resources = resources[:valid]

	// add new resource
	for _, p := range data {
		found := false
		for idx := range resources {
			exist := resources[idx]
			if p.Name == exist.Name {
				found = true
				logrus.Infof("Keeping resource %s", p.Name)
				// If the resource already exists and is not being used, we want to use the latest configuration
				if exist.Owner == "" {
					if p.UseConfig != exist.UseConfig || p.Type != exist.Type {
						exist.UseConfig = p.UseConfig
						exist.Type = p.Type
						if err := r.Storage.UpdateResource(exist); err != nil {
							finalError = multierror.Append(finalError, err)
						}
					}
				}

				break
			}
		}

		if !found {
			if p.State == "" {
				p.State = common.Free
			}
			logrus.Infof("Adding resource %s", p.Name)
			resources = append(resources, p)
			if err := r.Storage.AddResource(p); err != nil {
				logrus.WithError(err).Errorf("unable to add resource %s", p.Name)
				finalError = multierror.Append(finalError, err)
			}
		}
	}
	return finalError
}

// Metric returns a metric object with metrics filled in
func (r *Ranch) Metric(rtype string) (common.Metric, error) {
	metric := common.Metric{
		Type:    rtype,
		Current: map[string]int{},
		Owners:  map[string]int{},
	}

	resources, err := r.Storage.GetResources()
	if err != nil {
		logrus.WithError(err).Error("cannot find resources")
		return metric, &ResourceNotFound{rtype}
	}

	for _, res := range resources {
		if res.Type != rtype {
			continue
		}

		if _, ok := metric.Current[res.State]; !ok {
			metric.Current[res.State] = 0
		}

		if _, ok := metric.Owners[res.Owner]; !ok {
			metric.Owners[res.Owner] = 0
		}

		metric.Current[res.State]++
		metric.Owners[res.Owner]++
	}

	if len(metric.Current) == 0 && len(metric.Owners) == 0 {
		return metric, &ResourceNotFound{rtype}
	}

	return metric, nil
}
