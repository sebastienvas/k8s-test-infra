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
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"github.com/shurcooL/githubql"
)

// ignoreContexts abstracts checking for optional contexts for a PR.
type ignoreContexts interface {
	ignoreContext(string) bool
	getRequiredContexts() sets.String
}

func newContext(c string) Context {
	return Context{
		Context:     githubql.String(c),
		State:       githubql.StatusStateExpected,
		Description: githubql.String(""),
	}
}

// tideIC ignore tide context.
type tideIC struct {}

// contextRegister contains all registered ignoreContexts.
type contextRegister struct {
	lock sync.RWMutex
	ics [] ignoreContexts
}

// branchProtectionIC implements ignoreContexts for the branch protection plugin.
type branchProtectionIC struct {
	requiredContexts sets.String
}

// newContextRegister should be instantiated in tide for each combination of org, repo, branch.
// It contains all the registered ignoreContexts
func newContextRegister() contextRegister {
	ic := contextRegister{}
	ic.Register(tideIC{})
	return ic
}

func newBranchProtectionIC(bp *config.Branch) ignoreContexts {
	if bp == nil || !*bp.Protect {
		return nil
	}
	return branchProtectionIC{
		requiredContexts: sets.NewString(bp.Contexts...),
	}
}

func (r contextRegister) ignoreContext(c Context) bool {
	for _, i := range r.ics {
		if i.ignoreContext(string(c.Context)) {
			return true
		}
	}
	return false
}

func (r contextRegister) missingContexts(contexts []Context) []Context {
	missingContextsSet := sets.NewString()
	currentContextsSet := sets.NewString()
	for _, c := range contexts {
		currentContextsSet.Insert(string(c.Context))
	}
	for _, i := range r.ics {
		missingContextsSet.Insert(i.getRequiredContexts().Difference(currentContextsSet).List()...)
	}
	var missingContexts []Context
	for c := range missingContextsSet {
		missingContexts = append(missingContexts, newContext(c))
	}
	return missingContexts
}

func (r contextRegister) Register(i ignoreContexts) {
	if i == nil {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.ics = append(r.ics, i)
}

func (ic tideIC) ignoreContext(c string) bool {
	if c == statusContext {
		return true
	}
	return false
}

func (ic tideIC) getRequiredContexts() sets.String {
	return nil
}

func (ic branchProtectionIC) ignoreContext(c string) bool {
	return !ic.requiredContexts.Has(c)
}

func (ic branchProtectionIC) getRequiredContexts() sets.String {
	return ic.requiredContexts
}
