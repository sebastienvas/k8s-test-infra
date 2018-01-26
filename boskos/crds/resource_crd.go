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
	"time"

	"k8s.io/test-infra/boskos/common"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ResourceKind   = "Resource"
	ResourcePlural = "resources"
)

var (
	ResourceType = Type{
		Kind:       ResourceKind,
		Plural:     ResourcePlural,
		Object:     &Resource{},
		Collection: &ResourceList{},
	}
)

func NewResourceClient() (*Client, error) {
	return NewClientFromFlag(ResourceType)
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
	Type string `json:"type"`
}

// ResourceStatus holds information about on leased resources as well as
// information on how to use the new resource created.
type ResourceStatus struct {
	State      string          `json:"state,omitempty"`
	Owner      string          `json:"owner"`
	LastUpdate time.Time       `json:"lastupdate,omitempty"`
	UserData   common.UserData `json:"userdata,omitempty"`
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
		Owner:      in.Status.Owner,
		State:      in.Status.State,
		LastUpdate: in.Status.LastUpdate,
		UserData:   in.Status.UserData,
	}
}

func (in *Resource) ToItem() common.Item {
	return in.ToResource()
}

func (in *Resource) FromResource(r common.Resource) {
	in.Name = r.Name
	in.Spec.Type = r.Type
	in.Status.Owner = r.Owner
	in.Status.State = r.State
	in.Status.LastUpdate = r.LastUpdate
	in.Status.UserData = r.UserData
}

func (in *Resource) FromItem(i common.Item) {
	r, err := common.ItemToResource(i)
	if err == nil {
		in.FromResource(r)
	}
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
