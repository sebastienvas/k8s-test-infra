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

package crds

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/test-infra/boskos/common"
	"time"
)

const (
	ResourceKind         = "Resource"
	ResourcePlural       = "resources"
	ResourceConfigKind   = "ResourceConfig"
	ResourceConfigPlural = "resourceconfigs"
)

var knownTypes = map[string]struct {
	object     Object
	collection Collection
}{
	ResourceConfigPlural: {
		object:     &ResourceConfig{},
		collection: &ResourceConfigList{},
	},
	ResourcePlural: {
		object:     &Resource{},
		collection: &ResourceList{},
	},
}

type Object interface {
	runtime.Object
	GetName() string
}

type Collection interface {
	runtime.Object
	SetItems([]Object)
	GetItems() []Object
}

// Resource holds the Resource Data. This is where the data is persisted.
// The Resource will be generated from this

type Resource struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`
	Spec          ResourceSpec   `json:"spec,omitempty"`
	Status        ResourceStatus `json:"status,omitempty"`
}

type ResourceList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []*Resource `json:"items"`
}

type ResourceSpec struct {
	Type      string `json:"type"`
	UseConfig bool   `json:"useconfig,omitempty"`
}

// ResourceStatus holds information about on leased resources as well as
// information on how to use the new resource created.
type ResourceStatus struct {
	State      string              `json:"state,omitempty"`
	Owner      string              `json:"owner"`
	LastUpdate time.Time           `json:"lastupdate,omitempty"`
	Info       common.ResourceInfo `json:"info,omitempty"`
}

// ResourceConfig holds generalized configuration information about how the
// resource needs to be created. Some Resource might not have a ResourceConfig (Example Project)

type ResourceConfig struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`
	Spec          ResourceConfigSpec `json:"spec"`
}

type ResourceConfigSpec struct {
	Config common.TypedContent  `json:"config"`
	Needs  common.ResourceNeeds `json:"resourceneeds"`
}

type ResourceConfigList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []*ResourceConfig `json:"items"`
}

func (in *Resource) GetName() string {
	return in.Name
}

func (in *Resource) DeepCopyInto(out *Resource) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

func (in *Resource) DeepCopy() *Resource {
	if in == nil {
		return nil
	}
	out := new(Resource)
	in.DeepCopyInto(out)
	return out
}

func (in *Resource) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *Resource) ToResource() common.Resource {
	return common.Resource{
		Name:       in.Name,
		Type:       in.Spec.Type,
		UseConfig:  in.Spec.UseConfig,
		Owner:      in.Status.Owner,
		State:      in.Status.State,
		LastUpdate: in.Status.LastUpdate,
		Info:       in.Status.Info,
	}
}

func (in *Resource) FromResource(r common.Resource) {
	in.Name = r.Name
	in.Spec.Type = r.Type
	in.Spec.UseConfig = r.UseConfig
	in.Status.Owner = r.Owner
	in.Status.State = r.State
	in.Status.LastUpdate = r.LastUpdate
	in.Status.Info = r.Info
}

func (in *ResourceList) GetItems() []Object {
	var items []Object
	for _, i := range in.Items {
		items = append(items, i)
	}
	return items
}

func (in *ResourceList) SetItems(objects []Object) {
	var items []*Resource
	for _, b := range objects {
		items = append(items, b.(*Resource))
	}
	in.Items = items
}

func (in *ResourceList) DeepCopyInto(out *ResourceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	out.Items = in.Items
}

func (in *ResourceList) DeepCopy() *ResourceList {
	if in == nil {
		return nil
	}
	out := new(ResourceList)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ResourceConfig) GetName() string {
	return in.Name
}

func (in *ResourceConfig) DeepCopyInto(out *ResourceConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

func (in *ResourceConfig) DeepCopy() *ResourceConfig {
	if in == nil {
		return nil
	}
	out := new(ResourceConfig)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourceConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ResourceConfig) ToResource() common.ResourceConfig {
	return common.ResourceConfig{
		Name:   in.Name,
		Config: in.Spec.Config,
		Needs:  in.Spec.Needs,
	}
}

func (in *ResourceConfig) FromResource(r common.ResourceConfig) {
	in.ObjectMeta.Name = r.Name
	in.Spec.Config = r.Config
	in.Spec.Needs = r.Needs
}

func (in *ResourceConfigList) GetItems() []Object {
	var items []Object
	for _, i := range in.Items {
		items = append(items, i)
	}
	return items
}

func (in *ResourceConfigList) SetItems(objects []Object) {
	var items []*ResourceConfig
	for _, b := range objects {
		items = append(items, b.(*ResourceConfig))
	}
	in.Items = items
}

func (in *ResourceConfigList) GetName() string {
	return "List"
}

func (in *ResourceConfigList) DeepCopyInto(out *ResourceConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	out.Items = in.Items
}

func (in *ResourceConfigList) DeepCopy() *ResourceConfigList {
	if in == nil {
		return nil
	}
	out := new(ResourceConfigList)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourceConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
