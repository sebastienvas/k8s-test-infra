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
	"testing"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/boskos/common"
	"time"
)

const (
	fakeConfigType = "fakeConfig"
	fakeInfoType   = "fakeInfo"
	emptyContent   = "empty content"
)

type fakeBoskos struct {
	lock      sync.Mutex
	resources map[string]*common.Resource
	configs   map[string]*common.ResourceConfig
}

type testConfig map[string]struct {
	resourceNeeds *common.ResourceNeeds
	count         int
}

type fakeConfig struct {
}

func fakeConfigConverter(in string) (Config, error) {
	return &fakeConfig{}, nil
}

func (fc *fakeConfig) Construct(res *common.Resource, typeToRes common.TypeToResources) (*common.TypedContent, error) {
	return &common.TypedContent{
		Type:    fakeInfoType,
		Content: emptyContent,
	}, nil
}

// Create a fake client
func createFakeBoskos(tc testConfig) *fakeBoskos {
	fb := &fakeBoskos{}
	resources := map[string]*common.Resource{}
	configs := map[string]*common.ResourceConfig{}

	for rtype, c := range tc {
		for i := 0; i < c.count; i++ {
			res := common.Resource{
				Type:  rtype,
				Name:  fmt.Sprintf("%s_%d", rtype, i),
				State: common.Free,
			}
			if c.resourceNeeds != nil {
				res.UseConfig = true
				res.State = common.Dirty
				if _, ok := configs[rtype]; !ok {
					configs[rtype] = &common.ResourceConfig{
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

func (fb *fakeBoskos) UpdateOne(name, state string, info *common.ResourceInfo) error {
	fb.lock.Lock()
	defer fb.lock.Unlock()
	res, ok := fb.resources[name]
	if ok {
		res.State = state
		if info != nil {
			res.UserData = *info
		}
		return nil
	}
	return fmt.Errorf("no resource %v", name)
}

func (fb *fakeBoskos) GetConfig(name string) (*common.ResourceConfig, error) {
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
	bclient.resources["type2_0"].UserData = common.ResourceInfo{
		LeasedResources: []string{"type1_0"},
	}
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
