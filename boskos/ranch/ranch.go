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
	"gopkg.in/yaml.v2"
	"io/ioutil"

	"sync"
	"time"

	"github.com/deckarep/golang-set"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/crds"
	"reflect"
)

const (
	DeleteGracePeriodSeconds = 10
)

// Ranch is the place which all of the Resource objects lives.
type Ranch struct {
	Resources      []common.Resource
	ResourceClient crds.CRDClientInterface
	ConfigClient   crds.CRDClientInterface
	lock           sync.RWMutex
}

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

// ConfigNotFound will be returned if requested config does not exist.
type ConfigNotFound struct {
	name string
}

func (r ConfigNotFound) Error() string {
	return fmt.Sprintf("Config %s not exist", r.name)
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
func NewRanch(config string, client *crds.CRDclient) (*Ranch, error) {

	newRanch := &Ranch{
		ResourceClient: client,
	}

	newRanch.RestoreResources()

	if err := newRanch.SyncConfig(config); err != nil {
		return nil, err
	}

	newRanch.LogStatus()

	return newRanch, nil
}

func (r *Ranch) RestoreResources() error {
	o, err := r.ResourceClient.List(v1.ListOptions{})
	if err != nil {
		return err
	}
	l := o.(*crds.ResourceInstanceList)
	for _, res := range l.Items {
		r.Resources = append(r.Resources, res.ToResource())
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
func (r *Ranch) Acquire(rtype, state, dest, owner string) (*common.Resource, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for idx := range r.Resources {
		res := r.Resources[idx]
		if rtype == res.Type && state == res.State && res.Owner == "" {
			res.LastUpdate = time.Now()
			res.Owner = owner
			res.State = dest
			copy(r.Resources[idx:], r.Resources[idx+1:])
			r.Resources[len(r.Resources)-1] = res
			if err := r.updateResourceInstance(res); err != nil {
				return nil, err
			}
			for _, rName := range res.Info.LeasedResources {
				lr := r.searchByName(rName)
				if lr != nil {
					lr.LastUpdate = time.Now()
					if err := r.updateResourceInstance(*lr); err != nil {
						return &res, err
					}
				}
			}
			return &res, nil
		}
	}
	return nil, &ResourceNotFound{rtype}
}

// GetConfig returns the StructInfo for a given config name.
// In: name - name of the target config
// Out:
//   A valid  StructInfo object on success, or
//   ConfigNotFound error if target named config does not exist.
func (r *Ranch) GetConfig(name string) (*common.ConfigEntry, error) {
	o, err := r.ConfigClient.Get(name)
	if err != nil {
		logrus.WithError(err).Errorf("Unable to get config for %s", name)
		return nil, ConfigNotFound{name}
	}
	c := o.(*crds.ResourceConfig)
	configEntry := common.ConfigEntry{
		Needs:        c.Spec.Needs,
		Name:         c.Name,
		TypedContent: c.Spec.TypedContent,
	}
	return &configEntry, nil
}

// Release unsets owner for target resource and move it to a new state.
// In: name - name of the target resource
//     dest - destination state of the resource
//     owner - owner of the resource
// Out: nil on success, or
//      OwnerNotMatch error if owner does not match current owner of the resource, or
//      ResourceNotFound error if target named resource does not exist.
func (r *Ranch) Release(name, dest, owner string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	res := r.searchByName(name)
	if res == nil {
		return &ResourceNotFound{name}
	}
	if owner != res.Owner {
		return &OwnerNotMatch{res.Owner, owner}
	}
	res.LastUpdate = time.Now()
	res.Owner = ""
	res.State = dest
	if err := r.updateResourceInstance(*res); err != nil {
		return err
	}

	for _, rName := range res.Info.LeasedResources {
		lr := r.searchByName(rName)
		if lr != nil {
			lr.LastUpdate = time.Now()
			if err := r.updateResourceInstance(*lr); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Ranch) updateResourceInstance(res common.Resource) error {
	o, err := r.ResourceClient.Get(res.Name)
	if err != nil {
		logrus.WithError(err).Errorf("unable to find ResourceInstance %s", res.Name)
		return err
	}

	ri := o.(*crds.ResourceInstance)
	ri.FromResource(res)
	logrus.Infof("Updating ResourceInstance %s", ri.Name)
	if _, err := r.ResourceClient.Update(ri); err != nil {
		logrus.WithError(err).Errorf("failed to update ResourceInstance %s", ri.Name)
		return err
	}
	return nil
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
	r.lock.Lock()
	defer r.lock.Unlock()

	res := r.searchByName(name)
	if res == nil {
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
	if err := r.updateResourceInstance(*res); err != nil {
		return err
	}

	for _, rName := range res.Info.LeasedResources {
		lr := r.searchByName(rName)
		if lr != nil {
			res.LastUpdate = time.Now()
			if err := r.updateResourceInstance(*lr); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Ranch) searchByName(name string) *common.Resource {
	for idx := range r.Resources {
		res := &r.Resources[idx]
		if name == res.Name {
			return res
		}
	}
	return nil
}

// Reset unstucks a type of stale resource to a new state.
// In: rtype - type of the resource
//     state - current state of the resource
//     expire - duration before resource's last update
//     dest - destination state of expired resources
// Out: map of resource name - resource owner.
func (r *Ranch) Reset(rtype, state string, expire time.Duration, dest string) (map[string]string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	ret := make(map[string]string)

	for idx := range r.Resources {
		res := &r.Resources[idx]
		if rtype == res.Type && state == res.State && res.Owner != "" {
			if time.Now().Sub(res.LastUpdate) > expire {
				res.LastUpdate = time.Now()
				ret[res.Name] = res.Owner
				res.Owner = ""
				res.State = dest
				if err := r.updateResourceInstance(*res); err != nil {
					return ret, err
				}
				for _, rName := range res.Info.LeasedResources {
					lr := r.searchByName(rName)
					if lr != nil {
						res.LastUpdate = time.Now()
						if err := r.updateResourceInstance(*lr); err != nil {
							return ret, err
						}
					}
				}
			}
		}
	}
	return ret, nil
}

// LogStatus outputs current status of all resources
func (r *Ranch) LogStatus() {
	r.lock.RLock()
	defer r.lock.RUnlock()

	resJSON, err := json.Marshal(r.Resources)
	if err != nil {
		logrus.WithError(err).Errorf("Fail to marshal Resources. %v", r.Resources)
	}
	logrus.Infof("Current Resources : %v", string(resJSON))
}

// SyncConfig updates resource list from a file
func (r *Ranch) SyncConfig(config string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

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
func ParseConfig(configPath string) ([]common.Resource, []crds.ResourceConfig, error) {
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
	var configs []crds.ResourceConfig
	for _, c := range data.Configs {
		configs = append(configs, crds.ResourceConfig{
			ObjectMeta: v1.ObjectMeta{Name: c.Name},
			Spec: crds.ResourceConfigSpec{
				Needs:        c.Needs,
				TypedContent: c.TypedContent,
			},
		})
		configNames[c.Name] = c.Needs
	}

	var resources []common.Resource
	for _, res := range data.Resources {

		if res.UseConfig {
			if c, ok := configNames[res.Type]; !ok {
				err := fmt.Errorf("resource type %s does not have associated config", res.Type)
				logrus.WithError(err).Error("using useconfig implies associated config")
				return nil, nil, err
			} else {
				// Updating resourceNeeds
				for k, v := range c {
					resourcesNeeds[k] += v
				}
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

func (r *Ranch) syncConfigs(newconfigs []crds.ResourceConfig) error {
	o, err := r.ConfigClient.List(v1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Error("cannot list config")
		return err
	}
	currentConfigs := o.(*crds.ResourceConfigList).Items

	currentSet := mapset.NewSet()
	newSet := mapset.NewSet()
	toUpdate := mapset.NewSet()

	var configs map[string]crds.ResourceConfig

	for _, c := range currentConfigs {
		currentSet.Add(c.Name)
		configs[c.Name] = *c
	}

	for _, c := range newconfigs {
		newSet.Add(c.Name)
		if old, exists := configs[c.Name]; exists {
			if !reflect.DeepEqual(old.Spec, c.Spec) {
				toUpdate.Add(c.Name)
				configs[c.Name] = c
			}
		}
	}

	var finalError error

	toDelete := currentSet.Difference(newSet)
	toAdd := newSet.Difference(currentSet)

	for _, n := range toDelete.ToSlice() {
		if err := r.ConfigClient.Delete(n.(string), v1.NewDeleteOptions(DeleteGracePeriodSeconds)); err != nil {
			finalError = multierror.Append(finalError, err)
		}
	}

	for _, n := range toAdd.ToSlice() {
		var rc crds.ResourceConfig
		rc = configs[n.(string)]
		if _, err := r.ConfigClient.Create(&rc); err != nil {
			finalError = multierror.Append(finalError, err)
		}
	}

	for _, n := range toUpdate.ToSlice() {
		var rc crds.ResourceConfig
		rc = configs[n.(string)]
		if _, err := r.ConfigClient.Update(&rc); err != nil {
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
	var finalError error

	// delete non-exist resource
	valid := 0
	for _, res := range r.Resources {
		// If currently busy, yield deletion to later cycles.
		if res.Owner != "" {
			r.Resources[valid] = res
			valid++
			continue
		}
		toDelete := true
		for _, newRes := range data {
			if res.Name == newRes.Name {
				r.Resources[valid] = res
				valid++
				toDelete = false
				break
			}
		}
		if toDelete {
			logrus.Infof("Deleting resource %s", res.Name)
			if err := r.ResourceClient.Delete(res.Name, v1.NewDeleteOptions(DeleteGracePeriodSeconds)); err != nil {
				finalError = multierror.Append(finalError, err)
				logrus.WithError(err).Errorf("unable to delete resource %s", res.Name)
			}
		}
	}
	r.Resources = r.Resources[:valid]

	// add new resource
	for _, p := range data {
		found := false
		for idx := range r.Resources {
			exist := r.Resources[idx]
			if p.Name == exist.Name {
				found = true
				logrus.Infof("Keeping resource %s", p.Name)
				// If the resource already exists, we want to use the latest configuration
				if p.UseConfig != exist.UseConfig || p.Type != exist.Type {
					exist.UseConfig = p.UseConfig
					exist.Type = p.Type
					if err := r.updateResourceInstance(exist); err != nil {
						finalError = multierror.Append(finalError, err)
					}
				}
				break
			}
		}

		if !found {
			if p.State == "" {
				p.State = "free"
			}
			logrus.Infof("Adding resource %s", p.Name)
			r.Resources = append(r.Resources, p)
			ri := new(crds.ResourceInstance)
			ri.FromResource(p)
			if _, err := r.ResourceClient.Create(ri); err != nil {
				logrus.WithError(err).Errorf("unable to add resource %s", ri.Name)
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

	for _, res := range r.Resources {
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
