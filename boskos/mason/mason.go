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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/ghodss/yaml"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/storage"
)

const (
	// LeasedResources is a common.UserData entry
	LeasedResources = "leasedResources"
	// Number of queued serialized client operation before starting to block
	maxClientOps = 50
	// Period between syncing all resources, this should be smaller than the reaper timeout
	boskosSyncPeriod = 10 * time.Minute
)

// Masonable should be implemented by all configurations
type Masonable interface {
	Construct(context.Context, *common.Resource, common.TypeToResources) (common.UserData, error)
}

// ConfigConverter converts a string into a Masonable
type ConfigConverter func(string) (Masonable, error)

type boskosClient interface {
	Acquire(rtype, state, dest string) (*common.Resource, error)
	AcquireByState(state, dest string, names []string) ([]common.Resource, error)
	ReleaseOne(name, dest string) error
	UpdateOne(name, state string, userData common.UserData) error
	SyncAll() error
	UpdateAll(dest string) error
	ReleaseAll(dest string) error
}

// Used to serialize client operations and reducing blocking operation on goroutines
type clientOps struct {
	queue  chan func()
	client boskosClient
}

func (c *clientOps) releaseOne(name, dest string) {
	c.queue <- func() {
		if err := c.client.ReleaseOne(name, dest); err != nil {
			logrus.WithError(err).Errorf("failed to release %s to state %s", name, dest)
		}
	}
}

func (c *clientOps) updateOne(name, state string, userData common.UserData) {
	c.queue <- func() {
		if err := c.client.UpdateOne(name, state, userData); err != nil {
			logrus.WithError(err).Errorf("failed to update release %s to state %s", name, state)
		}
	}
}

func (c *clientOps) add(fn func()) {
	c.queue <- fn
}

// Mason uses config to convert dirty resources to usable one
type Mason struct {
	client                             boskosClient
	serializedClientOps                clientOps
	cleanerCount                       int
	storage                            Storage
	pending, fulfilled, cleaned        chan requirements
	boskosWaitPeriod, boskosSyncPeriod time.Duration
	wg                                 sync.WaitGroup
	configConverters                   map[string]ConfigConverter
	cancel                             context.CancelFunc
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
// In: configPath - path to the config file
// Out: A list of ResourceConfig object on success, or nil on error.
func ParseConfig(configPath string) ([]common.ResourcesConfig, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, err
	}
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
// In: configs   - a list of resources configs
//     resources - a list of resources
// Out: nil on success, error on failure
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

// NewMason creates and initialized a new Mason object
// In: rtypes       - A list of resource types to act on
//     channelSize  - Size for all the channel
//     cleanerCount - Number of cleaning threads
//     client       - boskos client
//     boskosWaitPeriod    - time to wait before a retry
// Out: A Pointer to a Mason Object
func NewMason(cleanerCount int, client boskosClient, boskosWaitPeriod time.Duration) *Mason {
	return &Mason{
		client: client,
		serializedClientOps: clientOps{
			client: client,
			queue:  make(chan func(), maxClientOps),
		},
		cleanerCount:     cleanerCount,
		storage:          *newStorage(storage.NewMemoryStorage()),
		pending:          make(chan requirements),
		cleaned:          make(chan requirements, cleanerCount),
		fulfilled:        make(chan requirements, cleanerCount),
		boskosWaitPeriod: boskosWaitPeriod,
		boskosSyncPeriod: boskosSyncPeriod,
		configConverters: map[string]ConfigConverter{},
	}
}

func checkUserData(res common.Resource) (common.LeasedResources, error) {
	var leasedResources common.LeasedResources
	if err := res.UserData.Extract(LeasedResources, &leasedResources); err != nil {
		logrus.WithError(err).Errorf("failed to extract %s", LeasedResources)
		return nil, err
	}
	return leasedResources, nil
}

// RegisterConfigConverter is used to register a new Masonable interface
// In: name - identifier for Masonable implementation
//     fn   - function that will parse the configuration string and return a Masonable interface
//
// Out: nil on success, error otherwise
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

func (m *Mason) garbageCollect(req requirements) {
	names := []string{req.resource.Name}

	for _, resources := range req.fulfillment {
		for _, r := range resources {
			if r != nil {
				names = append(names, r.Name)
			}
		}
	}

	for _, name := range names {
		m.serializedClientOps.releaseOne(name, common.Dirty)
	}

}

func (m *Mason) cleanAll(ctx context.Context) {
	defer func() {
		logrus.Info("Exiting cleanAll Thread")
		m.wg.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-m.fulfilled:
			if err := m.cleanOne(ctx, &req.resource, req.fulfillment); err != nil {
				logrus.WithError(err).Errorf("unable to clean resource %s", req.resource.Name)
				m.garbageCollect(req)
			} else {
				m.cleaned <- req
			}
		}
	}
}

func (m *Mason) cleanOne(ctx context.Context, res *common.Resource, leasedResources common.TypeToResources) error {
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
	userData, err := config.Construct(ctx, res, leasedResources)
	if err != nil {
		logrus.WithError(err).Errorf("failed to construct resource %s", res.Name)
		return err
	}
	res.UserData.Update(userData)

	m.serializedClientOps.updateOne(res.Name, res.State, userData)
	logrus.Infof("Resource %s is cleaned", res.Name)
	return nil
}

func (m *Mason) freeAll(ctx context.Context) {
	defer func() {
		logrus.Info("Exiting freeAll Thread")
		m.wg.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-m.cleaned:
			if err := m.freeOne(req); err != nil {
				logrus.WithError(err).Errorf("failed to free up resource %s", req.resource.Name)
				m.garbageCollect(req)
			}
		}
	}
}

func (m *Mason) freeOne(req requirements) error {
	res := req.resource
	leasedResources, err := checkUserData(res)
	if err != nil {
		return err
	}
	// TODO: Implement a ReleaseMultiple in a transaction to prevent orphans
	m.serializedClientOps.add(func() {
		// Releasing main resource first to limit number of orphans.
		if err := m.client.ReleaseOne(req.resource.Name, common.Free); err != nil {
			logrus.WithError(err).Errorf("failed to release resource %s", req.resource.Name)
			m.garbageCollect(req)
			return
		}
		// And release leased resources as res.Name state
		for _, name := range leasedResources {
			if err := m.client.ReleaseOne(name, res.Name); err != nil {
				logrus.WithError(err).Errorf("unable to release %s to state %s", name, res.Name)
				m.garbageCollect(req)
				return
			}
		}
		logrus.Infof("Resource %s has been freed", res.Name)
	})
	return nil
}

func (m *Mason) recycleAll(ctx context.Context) {
	defer func() {
		logrus.Info("Exiting recycleAll Thread")
		m.wg.Done()
	}()
	tick := time.NewTicker(m.boskosWaitPeriod).C
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			configs, err := m.storage.GetConfigs()
			if err != nil {
				logrus.WithError(err).Error("unable to get configuration")
				continue
			}
			var configTypes []string
			for _, c := range configs {
				configTypes = append(configTypes, c.Name)
			}
			for _, r := range configTypes {
				if res, err := m.client.Acquire(r, common.Dirty, common.Cleaning); err != nil {
					logrus.WithError(err).Debug("boskos acquire failed!")
				} else {
					if req, err := m.recycleOne(res); err != nil {
						logrus.WithError(err).Errorf("unable to recycle resource %s", res.Name)
						m.serializedClientOps.releaseOne(res.Name, common.Dirty)
					} else {
						m.pending <- *req
					}
				}
			}
		}
	}
}

func (m *Mason) recycleOne(res *common.Resource) (*requirements, error) {
	logrus.Infof("Resource %s is being recycled", res.Name)
	configEntry, err := m.storage.GetConfig(res.Type)
	if err != nil {
		logrus.WithError(err).Errorf("could not get config for resource type %s", res.Type)
		return nil, err
	}
	leasedResources, _ := checkUserData(*res)
	if leasedResources != nil {
		resources, err := m.client.AcquireByState(res.Name, common.Leased, leasedResources)
		if err != nil {
			logrus.WithError(err).Warningf("could not acquire any leased resources for %s", res.Name)
		}

		for _, r := range resources {
			m.serializedClientOps.releaseOne(r.Name, common.Dirty)
		}
		// Deleting Leased Resources
		res.UserData.Delete(LeasedResources)
		m.serializedClientOps.updateOne(res.Name, res.State, common.UserDataFromMap(map[string]string{LeasedResources: ""}))
	}

	return &requirements{
		fulfillment: common.TypeToResources{},
		needs:       configEntry.Needs,
		resource:    *res,
	}, nil
}

func (m *Mason) syncAll(ctx context.Context) {
	defer func() {
		logrus.Info("Exiting syncAll Thread")
		m.wg.Done()
	}()
	tick := time.NewTicker(m.boskosWaitPeriod).C
	for {
		select {
		case <-ctx.Done():
			return
		case fn := <-m.serializedClientOps.queue:
			fn()
		case <-tick:
			m.client.SyncAll()
		}
	}
}

func (m *Mason) fulfillAll(ctx context.Context) {
	defer func() {
		logrus.Info("Exiting fulfillAll Thread")
		m.wg.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-m.pending:
			if err := m.fulfillOne(ctx, &req); err != nil {
				m.garbageCollect(req)
			} else {
				m.fulfilled <- req
			}
		}
	}
}

func (m *Mason) fulfillOne(ctx context.Context, req *requirements) error {
	// Making a copy
	tick := time.NewTicker(m.boskosWaitPeriod).C
	needs := common.ResourceNeeds{}
	for k, v := range req.needs {
		needs[k] = v
	}
	for rType := range needs {
		for needs[rType] > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-tick:
				// In case we are waiting for a very long time,
				// we need to make sure that existing resources are being updated
				// TODO: handle update failures to update fullfillment.
				m.updateResources(req)
				if res, err := m.client.Acquire(rType, common.Free, common.Leased); err != nil {
					logrus.WithError(err).Debug("boskos acquire failed!")
				} else {
					req.fulfillment[rType] = append(req.fulfillment[rType], res)
					needs[rType]--
				}
			}
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
			logrus.WithError(err).Errorf("failed to add %s user data", LeasedResources)
			return err
		}
		req.resource.UserData.Update(userData)
		m.serializedClientOps.updateOne(req.resource.Name, req.resource.State, userData)
		logrus.Infof("requirements for release %s is fulfilled", req.resource.Name)
		return nil
	}
	return nil
}

func (m *Mason) updateResources(req *requirements) {
	var resources []*common.Resource
	resources = append(resources, &req.resource)
	for _, leasedResources := range req.fulfillment {
		resources = append(resources, leasedResources...)
	}
	for _, r := range resources {
		m.serializedClientOps.updateOne(r.Name, r.State, common.UserData{})
	}
}

// UpdateConfigs updates configs from storage path
// In: storagePath - the path to read the config file from
// Out: nil on success error otherwise
func (m *Mason) UpdateConfigs(storagePath string) error {
	configs, err := ParseConfig(storagePath)
	if err != nil {
		logrus.WithError(err).Error("unable to parse config")
		return err
	}
	return m.storage.SyncConfigs(configs)
}

func (m *Mason) start(ctx context.Context, fn func(context.Context)) {
	go func() {
		fn(ctx)
	}()
	m.wg.Add(1)
}

// Start Mason
func (m *Mason) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.start(ctx, m.syncAll)
	m.start(ctx, m.recycleAll)
	for i := 0; i < m.cleanerCount; i++ {
		m.start(ctx, m.fulfillAll)
	}
	for i := 0; i < m.cleanerCount; i++ {
		m.start(ctx, m.cleanAll)
	}
	m.start(ctx, m.freeAll)
	logrus.Info("Mason started")
}

// Stop Mason
func (m *Mason) Stop() {
	logrus.Info("Stopping Mason")
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	close(m.pending)
	close(m.cleaned)
	close(m.fulfilled)
	close(m.serializedClientOps.queue)
	// Attempting to sync all client ops
	for fn := range m.serializedClientOps.queue {
		fn()
	}
	m.client.ReleaseAll(common.Dirty)
	logrus.Info("Mason stopped")
}
