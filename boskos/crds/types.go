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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/test-infra/boskos/common"
	"time"
)

const (
	ResourcesInstancesKind = "ResourceInstance"
	ResourceInstancePlural = "resourceinstances"
	ResourcesConfigsKind   = "Config"
	ResourceConfigPlural   = "resourceconfigs"
)

var knownTypes = map[string]struct {
	object     BoskosObject
	collection BoskosCollection
}{
	ResourceConfigPlural: {
		object:     &ResourceConfig{},
		collection: &ResourceConfigList{},
	},
	ResourceInstancePlural: {
		object:     &ResourceInstance{},
		collection: &ResourceInstanceList{},
	},
}

type BoskosObject interface {
	runtime.Object
	GetName() string
}

type BoskosCollection interface {
	runtime.Object
	SetItems([]BoskosObject)
	GetItems() []BoskosObject
}

// ResourceInstance holds the Resource Data. This is where the data is persisted.
// The Resource will be generated from this

type ResourceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceInstanceSpec   `json:"spec,omitempty"`
	Status            ResourceInstanceStatus `json:"status,omitempty"`
}

type ResourceInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []*ResourceInstance `json:"items"`
}

type ResourceInstanceSpec struct {
	Type      string `json:"type"`
	UseConfig bool   `json:"useconfig,omitempty"`
}

// ResourceInstanceStatus holds information about on leased resources as well as
// information on how to use the new resource created.

type ResourceInstanceStatus struct {
	State      string              `json:"state,omitempty"`
	Owner      string              `json:"owner"`
	LastUpdate time.Time           `json:"lastupdate,omitempty"`
	Info       common.ResourceInfo `json:"info,omitempty"`
}

// Config holds generalized configuration information about how the
// resource needs to be created.
// There is a one to many relation from Config to ResourceInstance.
// Although some ResourceInstance might not have a Config (Example Project)

type ResourceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceConfigSpec `json:"spec"`
}

type ResourceConfigSpec struct {
	common.TypedContent `json:"spec"`
	Needs               common.ResourceNeeds `json:"resourceneeds"`
}

type ResourceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []*ResourceConfig `json:"items"`
}

func (in *ResourceInstance) GetName() string {
	return in.Name
}

func (in *ResourceInstance) DeepCopyInto(out *ResourceInstance) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

func (in *ResourceInstance) DeepCopy() *ResourceInstance {
	if in == nil {
		return nil
	}
	out := new(ResourceInstance)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourceInstance) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ResourceInstance) ToResource() common.Resource {
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

func (in *ResourceInstance) FromResource(r common.Resource) {
	in.Name = r.Name
	in.Spec.Type = r.Type
	in.Spec.UseConfig = r.UseConfig
	in.Status.Owner = r.Owner
	in.Status.State = r.State
	in.Status.LastUpdate = r.LastUpdate
	in.Status.Info = r.Info
}

func (in *ResourceInstanceList) GetItems() []BoskosObject {
	var items []BoskosObject
	for _, i := range in.Items {
		items = append(items, i)
	}
	return items
}

func (in *ResourceInstanceList) SetItems(objects []BoskosObject) {
	var items []*ResourceInstance
	for _, b := range objects {
		items = append(items, b.(*ResourceInstance))
	}
	in.Items = items
}

func (in *ResourceInstanceList) DeepCopyInto(out *ResourceInstanceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	out.Items = in.Items
}

func (in *ResourceInstanceList) DeepCopy() *ResourceInstanceList {
	if in == nil {
		return nil
	}
	out := new(ResourceInstanceList)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourceInstanceList) DeepCopyObject() runtime.Object {
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

func (in *ResourceConfig) ToResource() common.ConfigEntry {
	return common.ConfigEntry{
		Name:         in.Name,
		TypedContent: in.Spec.TypedContent,
		Needs:        in.Spec.Needs,
	}
}

func (in *ResourceConfig) FromResource(r common.ConfigEntry) {
	in.Name = r.Name
	in.Spec.TypedContent = r.TypedContent
	in.Spec.Needs = r.Needs
}

func (in *ResourceConfigList) GetItems() []BoskosObject {
	var items []BoskosObject
	for _, i := range in.Items {
		items = append(items, i)
	}
	return items
}

func (in *ResourceConfigList) SetItems(objects []BoskosObject) {
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
