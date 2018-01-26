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

package common

import (
	"fmt"
)

// ResourceNeeds maps the type to count of resources types needed
type ResourceNeeds map[string]int

// TypeToResources stores all the leased resources with the same type f
type TypeToResources map[string][]*Resource

type TypedContent struct {
	// Identifier of the struct this maps back to
	Type string `json:"type,omitempty"`
	// Marshaled JSON content
	Content string `json:"content,omitempty"`
}

type ResourceInfo struct {
	LeasedResources []string     `json:"leasedresouces,omitempty"`
	Info            TypedContent `json:"info,omitempty"`
}

type MasonConfig struct {
	Configs []ResourcesConfig `json:"configs,flow,omitempty"`
}

type ResourcesConfig struct {
	Name   string        `json:"name"`
	Config TypedContent  `json:"config"`
	Needs  ResourceNeeds `json:"needs"`
}

type ResourcesConfigByName []ResourcesConfig

func (ut ResourcesConfigByName) Len() int           { return len(ut) }
func (ut ResourcesConfigByName) Swap(i, j int)      { ut[i], ut[j] = ut[j], ut[i] }
func (ut ResourcesConfigByName) Less(i, j int) bool { return ut[i].GetName() < ut[j].GetName() }

func (conf ResourcesConfig) GetName() string { return conf.Name }

func ItemToResourcesConfig(i interface{}) (ResourcesConfig, error) {
	conf, ok := i.(ResourcesConfig)
	if !ok {
		return ResourcesConfig{}, fmt.Errorf("cannot construct Resource from received object %v", i)
	}
	return conf, nil
}
