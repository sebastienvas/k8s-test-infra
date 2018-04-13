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

// Package tide contains a controller for managing a tide pool of PRs. The
// controller will automatically retest PRs in the pool and merge them if they
// pass tests.

package tide

import (
	"reflect"
	"testing"
	"k8s.io/test-infra/prow/config"
)

func TestContextRegisterIgnoreContext(t *testing.T) {
	yes := true
	no := false
	testCases := []struct {
		name          string
		config        *config.Branch
		context       string
		ignoreContext bool
	}{
		{
			name:          "no config no required checks",
			context:       "c1",
			ignoreContext: false,
		},
		{
			name:          "no config tide",
			context:       statusContext,
			ignoreContext: true,
		},
		{
			name:          "empty config existing context",
			config:        &config.Branch{},
			context:       "c1",
			ignoreContext: false,
		},
		{
			name:          "config protect false missing context",
			config:        &config.Branch{Protect: &no},
			context:       "c4",
			ignoreContext: false,
		},
		{
			name:           "config existing context",
			config: &config.Branch{
				Protect: &yes,
				Contexts: []string{"c1", "c2", "c3"},
			},
			context:       "c1",
			ignoreContext: false,
		},
		{
			name:           "config missing context",
			config: &config.Branch{
				Protect: &yes,
				Contexts: []string{"c1", "c2", "c3"},
			},
			context:       "c4",
			ignoreContext: true,
		},

	}

	for _, tc := range testCases {
		rc := newContextRegister()
		rc.Register(newBranchProtectionIC(tc.config))
		if rc.ignoreContext(newContext(tc.context))!= tc.ignoreContext {
			t.Errorf("%s - ignoreContext should return %t", tc.name, tc.ignoreContext)
		}
	}
}

func TestContextRegisterMissingContexts(t *testing.T) {
	testCases := []struct {
		name             string
		config           *config.Branch
		existingContexts []string
		expectedContexts []string
	}{
		{
			name:             "no config no required checks",
			config:           &config.Branch{},
			existingContexts: []string{"c1", "c2"},
		},
		{
			name:             "no config missing context",
			config:           &config.Branch{
				Contexts:[]string{"c1", "c2", "c3"},
			},
			existingContexts: []string{"c1", "c2"},
		},
		{
			name:             "no config no missing context",
			config:           &config.Branch{
				Contexts: []string{"c1", "c2", "c3"},
			},
			existingContexts: []string{"c1", "c2", "c3"},
		},
		{
			name:           "config missing context",
			config: &config.Branch{
				Contexts:[]string{"c1", "c2", "c3"},
			},
			existingContexts: []string{"c1", "c2"},
			expectedContexts: []string{"c3"},
		},
		{
			name:           "config no missing context",
			config: &config.Branch{
				Contexts: []string{"c1", "c2", "c3"},
			},
			existingContexts: []string{"c1", "c2", "c3"},
		},
		{
			name:             "config no required checks",
			config:           &config.Branch{},
			existingContexts: []string{"c1", "c2"},
		},
	}

	for _, tc := range testCases {
		rc := newContextRegister()
		rc.Register(newBranchProtectionIC(tc.config))
		var contexts []Context
		for _, c := range tc.existingContexts {
			contexts = append(contexts, newContext(c))
		}
		missingContexts := rc.missingContexts(contexts)
		if !reflect.DeepEqual(missingContexts, tc.expectedContexts) {
			t.Errorf("%s - expected %v got %v", tc.name, tc.expectedContexts, missingContexts)
		}
	}
}