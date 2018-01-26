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
)

const (
	ResourcesConfigKind   = "ResourcesConfig"
	ResourcesConfigPlural = "resourcesconfigs"
)

var (
	ResourcesConfigType = Type{
		Kind:       ResourcesConfigKind,
		Plural:     ResourcesConfigPlural,
		Object:     &ResourcesConfig{},
		Collection: &ResourcesConfigList{},
	}
)

// ResourceConfig holds generalized configuration information about how the
// resource needs to be created. Some Resource might not have a ResourceConfig (Example Project)

type ResourcesConfig struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`
	Spec          ResourcesConfigSpec `json:"spec"`
}

type ResourcesConfigSpec struct {
	Config common.TypedContent  `json:"config"`
	Needs  common.ResourceNeeds `json:"resourceneeds"`
}

type ResourcesConfigList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []*ResourcesConfig `json:"items"`
}

func (in *ResourcesConfig) GetName() string {
	return in.Name
}

func (in *ResourcesConfig) DeepCopyInto(out *ResourcesConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

func (in *ResourcesConfig) DeepCopy() *ResourcesConfig {
	if in == nil {
		return nil
	}
	out := new(ResourcesConfig)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourcesConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ResourcesConfig) ToConfig() common.ResourcesConfig {
	return common.ResourcesConfig{
		Name:   in.Name,
		Config: in.Spec.Config,
		Needs:  in.Spec.Needs,
	}
}

func (in *ResourcesConfig) ToItem() common.Item {
	return in.ToConfig()
}

func (in *ResourcesConfig) FromConfig(r common.ResourcesConfig) {
	in.ObjectMeta.Name = r.Name
	in.Spec.Config = r.Config
	in.Spec.Needs = r.Needs
}

func (in *ResourcesConfig) FromItem(i common.Item) {
	c, err := common.ItemToResourcesConfig(i)
	if err == nil {
		in.FromConfig(c)
	}
}

func (in *ResourcesConfigList) GetItems() []Object {
	var items []Object
	for _, i := range in.Items {
		items = append(items, i)
	}
	return items
}

func (in *ResourcesConfigList) SetItems(objects []Object) {
	var items []*ResourcesConfig
	for _, b := range objects {
		items = append(items, b.(*ResourcesConfig))
	}
	in.Items = items
}

func (in *ResourcesConfigList) DeepCopyInto(out *ResourcesConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	out.Items = in.Items
}

func (in *ResourcesConfigList) DeepCopy() *ResourcesConfigList {
	if in == nil {
		return nil
	}
	out := new(ResourcesConfigList)
	in.DeepCopyInto(out)
	return out
}

func (in *ResourcesConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
