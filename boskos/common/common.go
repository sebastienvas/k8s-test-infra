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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	Dirty = "dirty"
	Free  = "free"
)

// UserData is a map of Name to user defined interface, serialized into a string
type UserData map[string]string

type LeasedResources []string

// Item interfaces for resources and configs
type Item interface {
	GetName() string
}

type Resource struct {
	Type       string    `json:"type"`
	Name       string    `json:"name"`
	State      string    `json:"state"`
	Owner      string    `json:"owner"`
	LastUpdate time.Time `json:"lastupdate"`
	// Information on how to use the resource
	UserData UserData `json:"userdata,omitempty"`
}

// ResourceEntry is resource config format defined from config.yaml
type ResourceEntry struct {
	Type  string   `json:"type"`
	State string   `json:"state"`
	Names []string `json:"names,flow"`
}

type BoskosConfig struct {
	Resources []ResourceEntry `json:"resources,flow"`
}

type Metric struct {
	Type    string         `json:"type"`
	Current map[string]int `json:"current"`
	Owners  map[string]int `json:"owner"`
	// TODO: Implement state transition metrics
}

func NewResource(name, rtype, state, owner string, t time.Time) Resource {
	return Resource{
		Name:       name,
		Type:       rtype,
		State:      state,
		Owner:      owner,
		LastUpdate: t,
		UserData:   UserData{},
	}
}

func NewResourcesFromConfig(e ResourceEntry) []Resource {
	var resources []Resource
	for _, name := range e.Names {
		resources = append(resources, NewResource(name, e.Type, e.State, "", time.Time{}))
	}
	return resources
}

// ResourceNotFound will be returned if requested resource does not exist.
type UserDataNotFound struct {
	id string
}

func (ud *UserDataNotFound) Error() string {
	return fmt.Sprintf("user data id %s does not exist", ud.id)
}

type ResourceByUpdateTime []Resource

func (ut ResourceByUpdateTime) Len() int           { return len(ut) }
func (ut ResourceByUpdateTime) Swap(i, j int)      { ut[i], ut[j] = ut[j], ut[i] }
func (ut ResourceByUpdateTime) Less(i, j int) bool { return ut[i].LastUpdate.Before(ut[j].LastUpdate) }

type ResourceByName []Resource

func (ut ResourceByName) Len() int           { return len(ut) }
func (ut ResourceByName) Swap(i, j int)      { ut[i], ut[j] = ut[j], ut[i] }
func (ut ResourceByName) Less(i, j int) bool { return ut[i].GetName() < ut[j].GetName() }

type ResTypes []string

func (r *ResTypes) String() string {
	return fmt.Sprint(*r)
}

func (r *ResTypes) Set(value string) error {
	if len(*r) > 0 {
		return errors.New("resTypes flag already set")
	}
	for _, rtype := range strings.Split(value, ",") {
		*r = append(*r, rtype)
	}
	return nil
}

func (res Resource) GetName() string { return res.Name }

func (ud UserData) Extract(id string, out interface{}) error {
	content, ok := ud[id]
	if !ok {
		return &UserDataNotFound{id}
	}
	return json.Unmarshal([]byte(content), out)
}

func (ud UserData) Set(id string, in interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	ud[id] = string(b)
	return nil
}

func (ud UserData) Update(new UserData) {
	if new != nil {
		for id, content := range new {
			if content != "" {
				ud[id] = content
			} else {
				delete(ud, id)
			}
		}
	}
}

func ItemToResource(i interface{}) (Resource, error) {
	res, ok := i.(Resource)
	if !ok {
		return Resource{}, fmt.Errorf("cannot construct Resource from received object %v", i)
	}
	return res, nil
}
