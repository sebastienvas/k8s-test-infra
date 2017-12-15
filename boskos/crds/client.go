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
	"fmt"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	Group   = "boskos.k8s.io"
	Version = "v1"
)

// CreateRESTConfig for cluster API server, pass empty config file for in-cluster
func CreateRESTConfig(kubeconfig string) (config *rest.Config, types *runtime.Scheme, err error) {
	if kubeconfig == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	if err != nil {
		return
	}

	version := schema.GroupVersion{
		Group:   Group,
		Version: Version,
	}

	config.GroupVersion = &version
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON

	types = runtime.NewScheme()
	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			for _, kind := range knownTypes {
				scheme.AddKnownTypes(version, kind.object, kind.collection)
			}
			v1.AddToGroupVersion(scheme, version)
			return nil
		})
	err = schemeBuilder.AddToScheme(types)
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(types)}

	return
}

// RegisterResources sends a request to create CRDs and waits for them to initialize
func RegisterResources(config *rest.Config) error {
	c, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return err
	}

	for _, s := range []struct{ p, k string }{
		{
			p: ResourceInstancePlural,
			k: ResourcesInstancesKind,
		},
		{
			p: ResourceConfigPlural,
			k: ResourcesConfigsKind,
		}} {
		crd := &apiextensionsv1beta1.CustomResourceDefinition{
			ObjectMeta: v1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", s.p, Group),
			},
			Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
				Group:   Group,
				Version: Version,
				Scope:   apiextensionsv1beta1.NamespaceScoped,
				Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
					Plural: s.p,
					Kind:   s.k,
				},
			},
		}
		if _, err := c.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func NewCRDClient(cl *rest.RESTClient, scheme *runtime.Scheme, namespace, plural string) *CRDclient {
	return &CRDclient{cl: cl, ns: namespace, plural: plural,
		codec: runtime.NewParameterCodec(scheme)}
}

func NewCRDDummyClient(plural string, objects []BoskosObject) *CRDDummyClient {
	c := CRDDummyClient{
		plural:  plural,
		objects: map[string]BoskosObject{},
	}
	for _, o := range objects {
		c.objects[o.GetName()] = o
	}
	return &c
}

type CRDClientInterface interface {
	Create(obj BoskosObject) (runtime.Object, error)
	Update(obj BoskosObject) (runtime.Object, error)
	Delete(name string, options *v1.DeleteOptions) error
	Get(name string) (runtime.Object, error)
	List(opts v1.ListOptions) (runtime.Object, error)
}

type CRDDummyClient struct {
	objects map[string]BoskosObject
	plural  string
}

func (c *CRDDummyClient) Create(obj BoskosObject) (runtime.Object, error) {
	c.objects[obj.GetName()] = obj
	return obj, nil
}

func (c *CRDDummyClient) Update(obj BoskosObject) (runtime.Object, error) {
	c.objects[obj.GetName()] = obj
	return obj, nil
}

func (c *CRDDummyClient) Delete(name string, options *v1.DeleteOptions) error {
	_, ok := c.objects[name]
	if ok {
		delete(c.objects, name)
		return nil
	}
	return fmt.Errorf("%s does not exist", name)
}

func (c *CRDDummyClient) Get(name string) (runtime.Object, error) {
	obj, ok := c.objects[name]
	if ok {
		return obj, nil
	}
	return nil, fmt.Errorf("could not find %s", name)
}

func (c *CRDDummyClient) List(opts v1.ListOptions) (runtime.Object, error) {
	var items []BoskosObject
	for _, i := range c.objects {
		items = append(items, i)
	}
	r := knownTypes[c.plural].collection
	r.SetItems(items)
	return r, nil
}

type CRDclient struct {
	cl     *rest.RESTClient
	ns     string
	plural string
	codec  runtime.ParameterCodec
}

func (c *CRDclient) Create(obj BoskosObject) (runtime.Object, error) {
	result := knownTypes[c.plural].object.DeepCopyObject()
	err := c.cl.Post().
		Namespace(c.ns).Resource(c.plural).
		Body(obj).Do().Into(result)
	return result, err
}

func (c *CRDclient) Update(obj BoskosObject) (runtime.Object, error) {
	result := knownTypes[c.plural].object.DeepCopyObject()
	err := c.cl.Put().
		Namespace(c.ns).Resource(c.plural).
		Body(obj).Do().Into(result)
	return result, err
}

func (c *CRDclient) Delete(name string, options *v1.DeleteOptions) error {
	return c.cl.Delete().
		Namespace(c.ns).Resource(c.plural).
		Name(name).Body(options).Do().
		Error()
}

func (c *CRDclient) Get(name string) (runtime.Object, error) {
	result := knownTypes[c.plural].object.DeepCopyObject()
	err := c.cl.Get().
		Namespace(c.ns).Resource(c.plural).
		Name(name).Do().Into(result)
	return result, err
}

func (c *CRDclient) List(opts v1.ListOptions) (runtime.Object, error) {
	result := knownTypes[c.plural].collection.DeepCopyObject()
	err := c.cl.Get().
		Namespace(c.ns).Resource(c.plural).
		VersionedParams(&opts, c.codec).
		Do().Into(result)
	return result, err
}
