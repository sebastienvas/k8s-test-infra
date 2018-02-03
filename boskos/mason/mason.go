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
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/client"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/storage"
	"github.com/hashicorp/go-multierror"
)

const (
	owner            = "Mason"
	defaultSleepTime = 10 * time.Second
	// LeasedResources is a common.UserData entry
	LeasedResources  = "LEASED_RESOURCES"
)

var (
	channelBufferSize = flag.Int("channel-buffer-size", 10, "Size of the channel buffer")
	rTypes            common.ResTypes
)

func init() {
	flag.Var(&rTypes, "resource-types", "comma-separated list of resources need to be cleaned up")
}

// Masonable should be implemented by all configurations
type Masonable interface {
	Construct(*common.Resource, common.TypeToResources) (common.UserData, error)
}

// ConfigConverter converts a string into a Masonable
type ConfigConverter func(string) (Masonable, error)

type boskosClient interface {
	Acquire(rtype, state, dest string) (*common.Resource, error)
	AcquireByState(state, dest string) ([]common.Resource, error)
	ReleaseOne(name, dest string) error
	UpdateOne(name, state string, userData common.UserData) error
}

// Mason uses config to convert dirty resources to usable one
type Mason struct {
	client                      boskosClient
	storage                     Storage
	pending, fulfilled, cleaned chan requirements
	typesToClean                []string
	sleepTime                   time.Duration
	wg                          sync.WaitGroup
	quit                        chan bool
	configConverters            map[string]ConfigConverter
	shutdown                    bool
}

// requirements for a given resource
type requirements struct {
	resource    common.Resource
	needs       common.ResourceNeeds
	fulfillment common.TypeToResources
}

func (r requirements) isFulFilled() bool {
	for rType, count := range r.needs {
		resources, ok := r.fulfillment[rType]
		if !ok {
			return false
		}
		if len(resources) != count {
			return false
		}
	}
	return true
}

// ParseConfig reads data stored in given config path
func ParseConfig(configPath string) ([]common.ResourcesConfig, error) {
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var data common.MasonConfig
	err = yaml.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}
	return data.Configs, nil
}

// ValidateConfig validates config with existing resources
func ValidateConfig(configs []common.ResourcesConfig, resources []common.Resource) error {
	resourcesNeeds := map[string]int{}
	actualResources := map[string]int{}

	configNames := map[string]map[string]int{}
	for _, c := range configs {
		_, alreadyExists := configNames[c.Name]
		if alreadyExists {
			return fmt.Errorf("config %s already exists", c.Name)
		}
		configNames[c.Name] = c.Needs
	}

	for _, res := range resources {
		_, useConfig := configNames[res.Type]
		if useConfig {
			c, ok := configNames[res.Type]
			if !ok {
				err := fmt.Errorf("resource type %s does not have associated config", res.Type)
				logrus.WithError(err).Error("using useconfig implies associated config")
				return err
			}
			// Updating resourceNeeds
			for k, v := range c {
				resourcesNeeds[k] += v
			}
		}
		actualResources[res.Type]++
	}

	for rType, needs := range resourcesNeeds {
		actual, ok := actualResources[rType]
		if !ok {
			err := fmt.Errorf("need for resource %s that does not exist", rType)
			logrus.WithError(err).Errorf("invalid configuration")
			return err
		}
		if needs > actual {
			err := fmt.Errorf("not enough resource of type %s for provisioning", rType)
			logrus.WithError(err).Errorf("invalid configuration")
			return err
		}
	}
	return nil
}

// NewMasonFromFlags creates a new Mason from flags
func NewMasonFromFlags() *Mason {
	boskosClient := client.NewClient(owner, *client.BoskosURL)
	masonClient := NewClient(boskosClient)
	logrus.Info("Initialized boskos client!")

	return newMason(rTypes, *channelBufferSize, masonClient, defaultSleepTime)
}

func newMason(rtypes []string, channelSize int, client boskosClient, sleepTime time.Duration) *Mason {
	return &Mason{
		client:           client,
		storage:          *newStorage(storage.NewMemoryStorage()),
		pending:          make(chan requirements, channelSize),
		cleaned:          make(chan requirements, channelSize),
		fulfilled:        make(chan requirements, channelSize),
		typesToClean:     rtypes,
		sleepTime:        sleepTime,
		configConverters: map[string]ConfigConverter{},
		quit:             make(chan bool),
	}
}

// RegisterConfigConverter is used to register a new Masonable interface
func (m *Mason) RegisterConfigConverter(name string, fn ConfigConverter) error {
	_, ok := m.configConverters[name]
	if ok {
		return fmt.Errorf("a converter for %s already exists", name)
	}
	m.configConverters[name] = fn
	return nil
}

func (m *Mason) convertConfig(configEntry *common.ResourcesConfig) (Masonable, error) {
	fn, ok := m.configConverters[configEntry.Config.Type]
	if !ok {
		return nil, fmt.Errorf("config type %s is not supported", configEntry.Name)
	}
	return fn(configEntry.Config.Content)
}

func (m *Mason) cleanAll() {
	defer func() {
		close(m.cleaned)
		logrus.Info("Exiting cleanAll Thread")
		m.wg.Done()
	}()
	for req := range m.fulfilled {
		if err := m.cleanOne(&req.resource, req.fulfillment); err != nil {
			err = m.client.ReleaseOne(req.resource.Name, common.Dirty)
			if err != nil {
				logrus.WithError(err).Errorf("Unable to release resource %s", req.resource.Name)
			}
		} else {
			m.cleaned <- req
		}
		time.Sleep(m.sleepTime)
	}
}

func (m *Mason) cleanOne(res *common.Resource, leasedResources common.TypeToResources) error {
	configEntry, err := m.storage.GetConfig(res.Type)
	if err != nil {
		logrus.WithError(err).Errorf("failed to get config for resource %s", res.Type)
		return err
	}
	config, err := m.convertConfig(&configEntry)
	if err != nil {
		logrus.WithError(err).Errorf("failed to convert config type %s - \n%s", configEntry.Config.Type, configEntry.Config.Content)
		return err
	}
	userData, err := config.Construct(res, leasedResources)
	if err != nil {
		logrus.WithError(err).Errorf("failed to construct resource %s", res.Name)
		return err
	}
	if err := m.client.UpdateOne(res.Name, res.State, userData); err != nil {
		logrus.WithError(err).Error("unable to update user data")
		return err
	}
	logrus.Infof("Resource %s is cleaned", res.Name)
	return nil
}

func (m *Mason) freeAll() {
	defer func() {
		logrus.Info("Exiting freeAll Thread")
		m.wg.Done()
	}()
	for req := range m.cleaned {
		if err := m.freeOne(&req.resource); err != nil {
			m.cleaned <- req
		}
		time.Sleep(m.sleepTime)
	}
}

func (m *Mason) freeOne(res *common.Resource) error {
	// Update Resource with Leased Resource and UserData
	if err := m.client.UpdateOne(res.Name, res.State, res.UserData); err != nil {
		logrus.WithError(err).Errorf("failed to update resource %s", res.Name)
		return err
	}
	// Finally return the resource as free
	if err := m.client.ReleaseOne(res.Name, common.Free); err != nil {
		logrus.WithError(err).Errorf("failed to release resource %s", res.Name)
		return err
	}
	var leasedResources common.LeasedResources
	if err := res.UserData.Extract(LeasedResources, &leasedResources); err != nil {
		logrus.WithError(err).Errorf("failed to extract %s", LeasedResources)
		return err
	}
	// And release leased resources as res.Name state
	var allErrors error
	for _, name := range leasedResources {
		if err := m.client.ReleaseOne(name, res.Name); err != nil {
			logrus.WithError(err).Errorf("unable to release %s to state %s", name, res.Name)
			allErrors = multierror.Append(allErrors, err)
		}
	}
	if allErrors != nil {
		return allErrors
	}
	logrus.Infof("Resource %s has been freed", res.Name)
	return nil
}

func (m *Mason) recycleAll() {
	defer func() {
		close(m.pending)
		logrus.Info("Exiting recycleAll Thread")
		m.wg.Done()
	}()
	for {
		select {
		case <-m.quit:
			return
		case <-time.After(m.sleepTime):
			for _, r := range m.typesToClean {
				if res, err := m.client.Acquire(r, common.Dirty, common.Cleaning); err != nil {
					logrus.WithError(err).Debug("boskos acquire failed!")
				} else {
					if req, err := m.recycleOne(res); err != nil {
						logrus.WithError(err).Errorf("unable to recycle resource %s", res.Name)
						err := m.client.ReleaseOne(res.Name, common.Dirty)
						if err != nil {
							logrus.WithError(err).Errorf("Unable to release resources %s", res.Name)
						}
					} else {
						m.pending <- *req
					}
				}
			}
		}
	}
}

func (m *Mason) recycleOne(res *common.Resource) (*requirements, error) {
	configEntry, err := m.storage.GetConfig(res.Type)
	if err != nil {
		logrus.WithError(err).Errorf("could not get config for resource type %s", res.Type)
		return nil, err
	}

	var leasedResources common.LeasedResources
	if err := res.UserData.Extract(LeasedResources, &leasedResources); err != nil {
		if _, ok := err.(*common.UserDataNotFound); !ok {
			logrus.WithError(err).Warning("cannot parse %s from User Data", LeasedResources)
			resources, err := m.client.AcquireByState(res.Name, common.Leased)
			if err != nil {
				logrus.WithError(err).Warningf("could not acquire any leased resources for %s", res.Name)
			}
			for _, r := range resources {
				leasedResources = append(leasedResources, r.Name)
			}
		}
	}

	// Releasing leased resources
	var allErrors error
	for _, lr := range leasedResources {
		if err := m.client.ReleaseOne(lr, common.Dirty); err != nil {
			logrus.WithError(err).Warning("could not release resource %s", lr)
		}
	}
	if allErrors != nil {
		return nil, allErrors
	}
	// Deleting Leased Resources
	delete(res.UserData, LeasedResources)
	if err := m.client.UpdateOne(res.Name, res.State, common.UserData{LeasedResources: ""}); err != nil {
		logrus.WithError(err).Errorf("could not update resource %s with freed leased resources", res.Name)
	}
	logrus.Infof("Resource %s is being recycled", res.Name)
	return &requirements{
		fulfillment: common.TypeToResources{},
		needs:       configEntry.Needs,
		resource:    *res,
	}, nil
}

func (m *Mason) fulfillAll() {
	defer func() {
		close(m.fulfilled)
		logrus.Info("Exiting fulfillAll Thread")
		m.wg.Done()
	}()
	for req := range m.pending {
		if err := m.fulfillOne(&req); err != nil {
			for _, resources := range req.fulfillment {
				for _, res := range resources {
					if err := m.client.ReleaseOne(res.Name, common.Free); err != nil {
						logrus.WithError(err).Errorf("failed to release resource %s", res.Name)
					}
					logrus.Infof("Released resource %s", res.Name)
				}
			}
		} else {
			m.fulfilled <- req
		}
		time.Sleep(m.sleepTime)
	}
}

func (m *Mason) fulfillOne(req *requirements) error {
	// Making a copy
	needs := common.ResourceNeeds{}
	for k, v := range req.needs {
		needs[k] = v
	}
	for rType := range needs {
		for needs[rType] > 0 {
			if m.shutdown {
				return fmt.Errorf("shutting down")
			}
			if res, err := m.client.Acquire(rType, common.Free, common.Leased); err != nil {
				logrus.WithError(err).Debug("boskos acquire failed!")
			} else {
				req.fulfillment[rType] = append(req.fulfillment[rType], res)
				needs[rType]--
			}
			time.Sleep(m.sleepTime)
		}
	}
	if req.isFulFilled() {
		var leasedResources common.LeasedResources
		for _, lr := range req.fulfillment {
			for _, r := range lr {
				leasedResources = append(leasedResources, r.Name)
			}
		}
		userData := common.UserData{}
		if err := userData.Set(LeasedResources, &leasedResources); err != nil {
			logrus.WithError(err).Error("failed to add %s user data", LeasedResources)
			return err
		}
		if err := m.client.UpdateOne(req.resource.Name, req.resource.State, userData); err != nil {
			logrus.WithError(err).Errorf("Unable to release resource %s", req.resource.Name)
			return err
		}
		logrus.Infof("requirements for release %s is fulfilled", req.resource.Name)
		return nil
	}
	return nil
}

// UpdateConfigs updates configs from storage path
func (m *Mason) UpdateConfigs(storagePath string) error {
	configs, err := ParseConfig(storagePath)
	if err != nil {
		logrus.WithError(err).Error("unable to parse config")
		return err
	}
	return m.storage.SyncConfigs(configs)
}

func (m *Mason) start(fn func()) {
	go fn()
	m.wg.Add(1)
}

// Start Mason
func (m *Mason) Start() {
	m.start(m.freeAll)
	m.start(m.fulfillAll)
	m.start(m.cleanAll)
	m.start(m.recycleAll)
	logrus.Info("Mason started")
}

// Stop Mason
func (m *Mason) Stop() {
	logrus.Info("Stopping Mason")
	m.shutdown = true
	m.quit <- true
	m.wg.Wait()
	logrus.Info("Mason stopped")
}
