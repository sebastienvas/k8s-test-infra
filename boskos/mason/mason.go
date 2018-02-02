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
)

const (
	Owner            = "Mason"
	DefaultSleepTime = 10 * time.Second
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

type ConfigConverter func(string) (Masonable, error)

type boskosClient interface {
	AcquireLeasedResource(name, rtype, state, dest string) (*common.Resource, error)
	AcquireLeasedResources(rtype, state, dest string) (*common.Resource, []common.Resource, error)
	ReleaseLeasedResources(name, dest string) error
	ReleaseOne(name, dest string) error
	UpdateOne(name, state string, userData common.UserData) error
}

type Mason struct {
	client                      boskosClient
	storage                     Storage
	pending, fulfilled, cleaned chan Requirement
	typesToClean                []string
	sleepTime                   time.Duration
	wg                          sync.WaitGroup
	quit                        chan bool
	configConverters            map[string]ConfigConverter
	shutdown                    bool
}

type Requirement struct {
	resource    common.Resource
	needs       common.ResourceNeeds
	fulfillment common.TypeToResources
}

func (r Requirement) IsFulFilled() bool {
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
		actualResources[res.Type] += 1
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

func NewMasonFromFlags() *Mason {
	boskosClient := client.NewClient(Owner, *client.BoskosURL)
	masonClient := NewClient(boskosClient)
	logrus.Info("Initialized boskos client!")

	return NewMason(rTypes, *channelBufferSize, masonClient, DefaultSleepTime)
}

func NewMason(rtypes []string, channelSize int, client boskosClient, sleepTime time.Duration) *Mason {
	return &Mason{
		client:           client,
		storage:          *NewStorage(storage.NewMemoryStorage()),
		pending:          make(chan Requirement, channelSize),
		cleaned:          make(chan Requirement, channelSize),
		fulfilled:        make(chan Requirement, channelSize),
		typesToClean:     rtypes,
		sleepTime:        sleepTime,
		configConverters: map[string]ConfigConverter{},
		quit:             make(chan bool),
	}
}

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
	// Finally return the resource as freeOne
	if err := m.client.ReleaseLeasedResources(res.Name, common.Free); err != nil {
		logrus.WithError(err).Errorf("failed to release resource %s", res.Name)
		return err
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
				if res, resources, err := m.client.AcquireLeasedResources(r, common.Dirty, common.Cleaning); err != nil {
					logrus.WithError(err).Debug("boskos acquire failed!")
				} else {
					if req, err := m.recycleOne(res, resources); err != nil {
						logrus.WithError(err).Errorf("unable to recycle resource %s", res.Name)
						err := m.client.ReleaseLeasedResources(res.Name, common.Dirty)
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

func (m *Mason) recycleOne(res *common.Resource, resources []common.Resource) (*Requirement, error) {
	configEntry, err := m.storage.GetConfig(res.Type)
	if err != nil {
		logrus.WithError(err).Errorf("could not get config for resource type %s", res.Type)
		return nil, err
	}

	// Releasing leased resources
	for _, lr := range resources {
		if err := m.client.ReleaseOne(lr.Name, common.Dirty); err != nil {
			logrus.WithError(err).Errorf("could not release resource %s", lr.Name)
			return nil, err
		}
	}
	newUserData := common.UserData{LeasedResources: ""}
	if err := m.client.UpdateOne(res.Name, res.State, newUserData); err != nil {
		logrus.WithError(err).Errorf("could not update resource %s with freed leased resources", res.Name)
	}
	delete(res.UserData, LeasedResources)
	logrus.Infof("Resource %s is being recycled", res.Name)
	return &Requirement{
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

func (m *Mason) fulfillOne(req *Requirement) error {
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
			if res, err := m.client.AcquireLeasedResource(req.resource.Name, rType, common.Free, common.Leased); err != nil {
				logrus.WithError(err).Debug("boskos acquire failed!")
			} else {
				req.fulfillment[rType] = append(req.fulfillment[rType], res)
				needs[rType]--
			}
			time.Sleep(m.sleepTime)
		}
	}
	if req.IsFulFilled() {
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
		logrus.Infof("Requirement for release %s is fulfilled", req.resource.Name)
		return nil
	}
	return nil
}

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

func (m *Mason) Start() {
	m.start(m.freeAll)
	m.start(m.fulfillAll)
	m.start(m.cleanAll)
	m.start(m.recycleAll)
	logrus.Info("Mason started")
}

func (m *Mason) Stop() {
	logrus.Info("Stopping Mason")
	m.shutdown = true
	m.quit <- true
	m.wg.Wait()
	logrus.Info("Mason stopped")
}
