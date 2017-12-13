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
	"errors"
	"fmt"
	"strings"
	"time"
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

type Resource struct {
	Type       string       `json:"type"`
	Name       string       `json:"name"`
	State      string       `json:"state"`
	Owner      string       `json:"owner"`
	LastUpdate time.Time    `json:"lastupdate"`
	UseConfig  bool         `json:"useconfig,omitempty"`
	Info       ResourceInfo `json:"info,omitempty"`
}

type ResourceInfo struct {
	LeasedResources []string     `json:"leasedresouces,omitempty"`
	Info            TypedContent `json:"info,omitempty"`
}

// ResourceEntry is resource config format defined from config.yaml
type ResourceEntry struct {
	Type      string   `json:"type"`
	State     string   `json:"state"`
	UseConfig bool     `json:"useConfig"`
	Names     []string `json:"names,flow"`
}

type ConfigEntry struct {
	TypedContent
	Needs ResourceNeeds `json:"needs"`
	Name  string        `json:"name"`
}

type BoskosConfig struct {
	Resources []ResourceEntry `json:"resources,flow"`
	Configs   []ConfigEntry   `json:"configs,flow,omitempty"`
}

type Metric struct {
	Type    string         `json:"type"`
	Current map[string]int `json:"current"`
	Owners  map[string]int `json:"owner"`
	// TODO: Implement state transition metrics
}

type ResTypes []string

func (r *ResTypes) String() string {
	return fmt.Sprint(*r)
}

func (rtypes *ResTypes) Set(value string) error {
	if len(*rtypes) > 0 {
		return errors.New("resTypes flag already set")
	}
	for _, rtype := range strings.Split(value, ",") {
		*rtypes = append(*rtypes, rtype)
	}
	return nil
}
