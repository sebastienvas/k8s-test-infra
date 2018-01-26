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
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/ranch"
	"k8s.io/test-infra/boskos/storage"
)

const (
	fakeConfigType = "fakeConfig"
	fakeInfoType   = "fakeInfo"
	emptyContent   = "empty content"
)

type fakeBoskos struct {
	lock      sync.Mutex
	resources map[string]*common.Resource
	configs   map[string]*common.ResourcesConfig
}

type testConfig map[string]struct {
	resourceNeeds *common.ResourceNeeds
	count         int
}

type fakeConfig struct {
}

func fakeConfigConverter(in string) (Masonable, error) {
	return &fakeConfig{}, nil
}

func (fc *fakeConfig) Construct(res *common.Resource, typeToRes common.TypeToResources) (*common.UserData, error) {
	return &common.UserData{"fakeConfig": "unused"}, nil
}

// Create a fake client
func createFakeBoskos(tc testConfig) *fakeBoskos {
	fb := &fakeBoskos{}
	resources := map[string]*common.Resource{}
	configs := map[string]*common.ResourcesConfig{}

	for rtype, c := range tc {
		for i := 0; i < c.count; i++ {
			res := common.Resource{
				Type:  rtype,
				Name:  fmt.Sprintf("%s_%d", rtype, i),
				State: common.Free,
				UserData: common.UserData{},
			}
			if c.resourceNeeds != nil {
				res.State = common.Dirty
				if _, ok := configs[rtype]; !ok {
					configs[rtype] = &common.ResourcesConfig{
						Config: common.TypedContent{
							Type:    fakeConfigType,
							Content: emptyContent,
						},
						Name:  rtype,
						Needs: *c.resourceNeeds,
					}
				}
			}
			resources[res.Name] = &res
		}
	}
	fb.resources = resources
	fb.configs = configs
	return fb
}

func (fb *fakeBoskos) Acquire(rtype, state, dest string) (*common.Resource, error) {
	fb.lock.Lock()
	defer fb.lock.Unlock()

	for _, r := range fb.resources {
		if r.Type == rtype && r.State == state {
			r.State = dest
			logrus.Infof("resource %s state updated from %s to %s", r.Name, state, dest)
			return r, nil
		}
	}

	return nil, fmt.Errorf("could not find resource of type %s", rtype)
}

func (fb *fakeBoskos) ReleaseOne(name, dest string) error {
	fb.lock.Lock()
	defer fb.lock.Unlock()
	res, ok := fb.resources[name]
	if ok {
		logrus.Infof("Released resource %s, state updated from %s to %s", name, res.State, dest)
		res.State = dest
		return nil
	}
	return fmt.Errorf("no resource %v", name)
}

func (fb *fakeBoskos) UpdateOne(name, state string, userData *common.UserData) error {
	fb.lock.Lock()
	defer fb.lock.Unlock()
	res, ok := fb.resources[name]
	if ok {
		res.State = state
		res.UserData.Update(userData)
		return nil
	}
	return fmt.Errorf("no resource %v", name)
}

func (fb *fakeBoskos) GetConfig(name string) (*common.ResourcesConfig, error) {
	fb.lock.Lock()
	defer fb.lock.Unlock()
	config, ok := fb.configs[name]
	if ok {
		return config, nil
	}
	return nil, fmt.Errorf("no resource %v", name)
}

func TestRecycleLeasedResources(t *testing.T) {
	masonTypes := []string{"type2"}
	tc := testConfig{
		"type1": {
			count: 1,
		},
		"type2": {
			resourceNeeds: &common.ResourceNeeds{
				"type1": 1,
			},
			count: 1,
		},
	}

	bclient := createFakeBoskos(tc)
	bclient.resources["type1_0"].State = Leased
	bclient.resources["type2_0"].UserData.Set(LeasedResources, &[]string{"type1_0"})
	m := NewMason(masonTypes, 1, bclient, time.Millisecond)
	m.RegisterConfigConverter(fakeConfigType, fakeConfigConverter)
	m.start(m.recycleAll)
	select {
	case <-m.pending:
		break
	case <-time.After(5 * time.Second):
		t.Errorf("Timeout")
	}
	m.Stop()
	if bclient.resources["type2_0"].State != Cleaning {
		t.Errorf("Resource state should be cleaning")
	}
	if bclient.resources["type1_0"].State != common.Dirty {
		t.Errorf("Resource state should be dirty")
	}
}

func TestRecycleNoLeasedResources(t *testing.T) {
	masonTypes := []string{"type2"}
	tc := testConfig{
		"type1": {
			count: 1,
		},
		"type2": {
			resourceNeeds: &common.ResourceNeeds{
				"type1": 1,
			},
			count: 1,
		},
	}

	bclient := createFakeBoskos(tc)
	m := NewMason(masonTypes, 1, bclient, time.Millisecond)
	m.RegisterConfigConverter(fakeConfigType, fakeConfigConverter)
	m.start(m.recycleAll)
	select {
	case <-m.pending:
		break
	case <-time.After(5 * time.Second):
		t.Errorf("Timeout")
	}
	m.Stop()
	if bclient.resources["type2_0"].State != Cleaning {
		t.Errorf("Resource state should be cleaning")
	}
	if bclient.resources["type1_0"].State != common.Free {
		t.Errorf("Resource state should be untouched, current %s", bclient.resources["type1_0"].State)
	}
}

func TestFulfillOne(t *testing.T) {
	masonTypes := []string{"type2"}
	tc := testConfig{
		"type1": {
			count: 1,
		},
		"type2": {
			resourceNeeds: &common.ResourceNeeds{
				"type1": 1,
			},
			count: 1,
		},
	}

	bclient := createFakeBoskos(tc)
	m := NewMason(masonTypes, 1, bclient, time.Millisecond)
	res := bclient.resources["type2_0"]
	req := Requirement{
		resource:    *res,
		needs:       bclient.configs["type2"].Needs,
		fulfillment: common.TypeToResources{},
	}
	if err := m.fulfillOne(&req); err != nil {
		t.Errorf("could not satisty requirement ")
	}
	if len(req.fulfillment) != 1 {
		t.Errorf("there should be only one type")
	}
	if len(req.fulfillment["type1"]) != 1 {
		t.Errorf("there should be only one resources")
	}
	userRes := req.fulfillment["type1"][0]
	if bclient.resources[userRes.Name].State != Leased {
		t.Errorf("Statys should be Leased")
	}

}

func TestMason(t *testing.T) {
	masonTypes := []string{"type2"}
	tc := testConfig{
		"type1": {
			count: 10,
		},
		"type2": {
			resourceNeeds: &common.ResourceNeeds{
				"type1": 10,
			},
			count: 10,
		},
	}
	bclient := createFakeBoskos(tc)
	m := NewMason(masonTypes, 5, bclient, time.Millisecond)
	m.RegisterConfigConverter(fakeConfigType, fakeConfigConverter)
	m.Start()
	<-time.After(2 * time.Second)
	for _, res := range bclient.resources {
		switch res.Type {
		case "type1":
			if res.State != Leased {
				t.Errorf("resource %v should be leased", res)
			}
		case "type2":
			if res.State != common.Free {
				t.Errorf("resource %v should be freeOne", res)
			}
		default:
			t.Errorf("resource type %s not expected", res.Type)
		}
	}
	res, err := bclient.Acquire("type2", common.Free, "Used")
	if err != nil {
		t.Error("There should be free resources")
	}
	bclient.ReleaseOne(res.Name, common.Dirty)
	m.Stop()
}

func TestMasonStartStop(t *testing.T) {
	masonTypes := []string{"type2"}
	tc := testConfig{
		"type1": {
			count: 10,
		},
		"type2": {
			resourceNeeds: &common.ResourceNeeds{
				"type1": 10,
			},
			count: 10,
		},
	}
	bclient := createFakeBoskos(tc)
	m := NewMason(masonTypes, 5, bclient, time.Millisecond)
	m.RegisterConfigConverter(fakeConfigType, fakeConfigConverter)
	m.Start()
	m.Stop()
}

func TestConfig(t *testing.T) {
	resources, err := ranch.ParseConfig("test-resources.yaml")
	if err != nil {
		t.Error(err)
	}
	configs, err := ParseConfig("test-configs.yaml")
	if err != nil {
		t.Error(err)
	}
	if err := ValidateConfig(configs, resources); err != nil {
		t.Error(err)
	}
}

func makeFakeConfig(name, cType, content string, needs int) common.ResourcesConfig {
	c := common.ResourcesConfig{
		Name:  name,
		Needs: common.ResourceNeeds{},
		Config: common.TypedContent{
			Type:    cType,
			Content: content,
		},
	}
	for i := 0; i < needs; i++ {
		c.Needs[fmt.Sprintf("type_%d", i)] = i
	}
	return c
}

func TestSyncConfig(t *testing.T) {
	var testcases = []struct {
		name      string
		oldConfig []common.ResourcesConfig
		newConfig []common.ResourcesConfig
		expect    []common.ResourcesConfig
	}{
		{
			name: "empty",
		},
		{
			name: "deleteAll",
			oldConfig: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType", "", 2),
				makeFakeConfig("config2", "fakeType", "", 3),
				makeFakeConfig("config3", "fakeType", "", 4),
			},
		},
		{
			name: "new",
			newConfig: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType", "", 2),
				makeFakeConfig("config2", "fakeType", "", 3),
				makeFakeConfig("config3", "fakeType", "", 4),
			},
			expect: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType", "", 2),
				makeFakeConfig("config2", "fakeType", "", 3),
				makeFakeConfig("config3", "fakeType", "", 4),
			},
		},
		{
			name: "noChange",
			oldConfig: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType", "", 2),
				makeFakeConfig("config2", "fakeType", "", 3),
				makeFakeConfig("config3", "fakeType", "", 4),
			},
			newConfig: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType", "", 2),
				makeFakeConfig("config2", "fakeType", "", 3),
				makeFakeConfig("config3", "fakeType", "", 4),
			},
			expect: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType", "", 2),
				makeFakeConfig("config2", "fakeType", "", 3),
				makeFakeConfig("config3", "fakeType", "", 4),
			},
		},
		{
			name: "update",
			oldConfig: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType", "", 2),
				makeFakeConfig("config2", "fakeType", "", 3),
				makeFakeConfig("config3", "fakeType", "", 4),
			},
			newConfig: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType2", "", 2),
				makeFakeConfig("config2", "fakeType", "something", 3),
				makeFakeConfig("config3", "fakeType", "", 5),
			},
			expect: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType2", "", 2),
				makeFakeConfig("config2", "fakeType", "something", 3),
				makeFakeConfig("config3", "fakeType", "", 5),
			},
		},
		{
			name: "delete",
			oldConfig: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType", "", 2),
				makeFakeConfig("config2", "fakeType", "", 3),
				makeFakeConfig("config3", "fakeType", "", 4),
			},
			newConfig: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType2", "", 2),
				makeFakeConfig("config3", "fakeType", "", 5),
			},
			expect: []common.ResourcesConfig{
				makeFakeConfig("config1", "fakeType2", "", 2),
				makeFakeConfig("config3", "fakeType", "", 5),
			},
		},
	}

	for _, tc := range testcases {
		s := NewStorage(storage.NewMemoryStorage())
		s.SyncConfigs(tc.newConfig)
		configs, err := s.GetConfigs()
		if err != nil {
			t.Errorf("failed to get resources")
			continue
		}
		sort.Stable(common.ResourcesConfigByName(configs))
		sort.Stable(common.ResourcesConfigByName(tc.expect))
		if !reflect.DeepEqual(configs, tc.expect) {
			t.Errorf("Test %v: got %v, expect %v", tc.name, configs, tc.expect)
		}
	}
}

func TestGetConfig(t *testing.T) {
	var testcases = []struct {
		name, configName string
		exists           bool
		configs          []common.ResourcesConfig
	}{
		{
			name:       "exists",
			exists:     true,
			configName: "test",
			configs: []common.ResourcesConfig{
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type:    "type3",
						Content: "content",
					},
					Name: "test",
				},
			},
		},
		{
			name:       "noConfig",
			exists:     false,
			configName: "test",
		},
		{
			name:       "existsMultipleConfigs",
			exists:     true,
			configName: "test1",
			configs: []common.ResourcesConfig{
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type:    "type3",
						Content: "content",
					},
					Name: "test",
				},
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type:    "type3",
						Content: "content",
					},
					Name: "test1",
				},
			},
		},
		{
			name:       "noExistMultipleConfigs",
			exists:     false,
			configName: "test2",
			configs: []common.ResourcesConfig{
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type:    "type3",
						Content: "content",
					},
					Name: "test",
				},
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type:    "type3",
						Content: "content",
					},
					Name: "test1",
				},
			},
		},
	}
	for _, tc := range testcases {
		s := NewStorage(storage.NewMemoryStorage())
		for _, config := range tc.configs {
			s.AddConfig(config)
		}
		config, err := s.GetConfig(tc.configName)
		if !tc.exists {
			if err == nil {
				t.Error("client should return an error")
			}
		} else {
			if config.Name != tc.configName {
				t.Error("config name should match")
			}
		}
	}
}
