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
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/client"
	"k8s.io/test-infra/boskos/common"
)

const (
	Dirty            = "dirty"
	Cleaning         = "cleaning"
	Leased           = "leased"
	Free             = "free"
	Owner            = "Mason"
	URL              = "http://boskos"
	DefaultSleepTime = time.Minute
)

var (
	channelBufferSize = flag.Int("channel-buffer-size", 10, "Size of the channel buffer")
	rTypes            common.ResTypes
)

func init() {
	flag.Var(&rTypes, "resource-types", "comma-separated list of resources need to be cleaned up")
}

// Config should be implemented by all configurations
type Config interface {
	Construct(*common.Resource, common.TypeToResources) (*common.ResourceInfo, error)
}

type ConfigConverter func(string) (Config, error)

type boskosClient interface {
	Acquire(rtype, state, dest string) (*common.Resource, error)
	ReleaseOne(name, dest string) error
	UpdateOne(name, state string, info *common.ResourceInfo) error
	GetConfig(name string) (*common.ConfigEntry, error)
}

type Mason struct {
	client                      boskosClient
	pending, fulfilled, cleaned chan Requirement
	typesToClean                []string
	sleepTime                   time.Duration
	wg                          sync.WaitGroup
	quit                        chan bool
	configConverters            map[string]ConfigConverter
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

func NewMasonFromFlags() *Mason {
	boskos := client.NewClient(Owner, URL)
	logrus.Info("Initialized boskos client!")
	return NewMason(rTypes, *channelBufferSize, boskos, DefaultSleepTime)
}

func NewMason(rtypes []string, channelSize int, client boskosClient, sleepTime time.Duration) *Mason {
	return &Mason{
		client:           client,
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

func (m *Mason) convertConfig(configEntry *common.ConfigEntry) (Config, error) {
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
			m.client.ReleaseOne(req.resource.Name, Dirty)
		} else {
			m.cleaned <- req
		}
		time.Sleep(m.sleepTime)
	}

}

func (m *Mason) cleanOne(res *common.Resource, leasedResources common.TypeToResources) error {
	configEntry, err := m.client.GetConfig(res.Type)
	if err != nil {
		logrus.WithError(err).Errorf("failed to get config for resource %s", res.Type)
		return err
	}
	config, err := m.convertConfig(configEntry)
	if err != nil {
		logrus.WithError(err).Errorf("failed to convert config type %s - \n%s", configEntry.Config.Type, configEntry.Config.Content)
		return err
	}
	info, err := config.Construct(res, leasedResources)
	if err != nil {
		logrus.WithError(err).Errorf("failed to construct resource %s", res.Name)
		return err
	}
	res.Info = *info
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
	// Update Resource with Leased Resource and Info
	if err := m.client.UpdateOne(res.Name, res.State, &res.Info); err != nil {
		logrus.WithError(err).Errorf("failed to update resource %s", res.Name)
		return err
	}
	// Finally return the resource as freeOne
	if err := m.client.ReleaseOne(res.Name, Free); err != nil {
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
				if res, err := m.client.Acquire(r, Dirty, Cleaning); err != nil {
					logrus.WithError(err).Debug("boskos acquire failed!")
				} else if res == nil {
					logrus.Fatal("nil resource was returned")
				} else {
					if req, err := m.recycleOne(res); err != nil {
						logrus.WithError(err).Error("")
						m.client.ReleaseOne(res.Name, Dirty)
					} else {
						m.pending <- *req
					}
				}
			}
		}
	}
}

func (m *Mason) recycleOne(res *common.Resource) (*Requirement, error) {
	configEntry, err := m.client.GetConfig(res.Type)
	if err != nil {
		logrus.WithError(err).Errorf("could not get config for resource type %s", res.Type)
		return nil, err
	}

	// Releasing leased resources
	for _, l := range res.Info.LeasedResources {
		if err := m.client.ReleaseOne(l, Dirty); err != nil {
			logrus.WithError(err).Errorf("could not release resource %s", l)
			return nil, err
		}
	}
	res.Info = common.ResourceInfo{}
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
					if err := m.client.ReleaseOne(res.Name, Free); err != nil {
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
	needs := req.needs
	for rType := range needs {
		if req.IsFulFilled() {
			logrus.Infof("Requirement for release %s is fulfilled", req.resource.Name)
			return nil
		}
		if needs[rType] <= 0 {
			continue
		}
		if res, err := m.client.Acquire(rType, Free, Leased); err != nil {
			logrus.WithError(err).Debug("boskos acquire failed!")
		} else {
			req.fulfillment[rType] = append(req.fulfillment[rType], res)
			needs[rType] -= 1
		}
		time.Sleep(m.sleepTime)
	}
	return nil
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
	m.quit <- true
	m.wg.Wait()
	logrus.Info("Mason stopped")
}
