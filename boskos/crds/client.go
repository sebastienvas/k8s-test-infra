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
	"flag"
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

var (
	kubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeConfig file")
	namespace  = flag.String("namespace", v1.NamespaceDefault, "namespace to install on")
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

// RegisterResource sends a request to create CRDs and waits for them to initialize
func RegisterResource(config *rest.Config, kind, plural string) error {
	c, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return err
	}

	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", plural, Group),
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   Group,
			Version: Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: plural,
				Kind:   kind,
			},
		},
	}
	if _, err := c.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func NewClient(cl *rest.RESTClient, scheme *runtime.Scheme, namespace, plural string) Client {
	return Client{cl: cl, ns: namespace, plural: plural,
		codec: runtime.NewParameterCodec(scheme)}
}

func NewDummyClient(plural string) *DummyClient {
	c := &DummyClient{
		plural:  plural,
		objects: make(map[string]Object),
	}
	return c
}

func NewResourceClient() (*Client, error) {
	return NewClientFromFlag(ResourceKind, ResourcePlural)
}

func NewClientFromFlag(kind, plural string) (*Client, error) {
	config, scheme, err := CreateRESTConfig(*kubeConfig)
	if err != nil {
		return nil, err
	}

	if err = RegisterResource(config, kind, plural); err != nil {
		return nil, err
	}
	// creates the client
	var restClient *rest.RESTClient
	restClient, err = rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}
	rc := NewClient(restClient, scheme, *namespace, plural)
	return &rc, nil
}

type ClientInterface interface {
	NewObject() Object
	NewCollection() Collection
	Create(obj Object) (Object, error)
	Update(obj Object) (Object, error)
	Delete(name string, options *v1.DeleteOptions) error
	Get(name string) (Object, error)
	List(opts v1.ListOptions) (Collection, error)
}

type DummyClient struct {
	objects map[string]Object
	plural  string
}

func (c *DummyClient) NewObject() Object {
	return knownTypes[c.plural].object.DeepCopyObject().(Object)
}

func (c *DummyClient) NewCollection() Collection {
	return knownTypes[c.plural].collection.DeepCopyObject().(Collection)
}

func (c *DummyClient) Create(obj Object) (Object, error) {
	c.objects[obj.GetName()] = obj
	return obj, nil
}

func (c *DummyClient) Update(obj Object) (Object, error) {
	_, ok := c.objects[obj.GetName()]
	if !ok {
		return nil, fmt.Errorf("cannot find object %s", obj.GetName())
	}
	c.objects[obj.GetName()] = obj
	return obj, nil
}

func (c *DummyClient) Delete(name string, options *v1.DeleteOptions) error {
	_, ok := c.objects[name]
	if ok {
		delete(c.objects, name)
		return nil
	}
	return fmt.Errorf("%s does not exist", name)
}

func (c *DummyClient) Get(name string) (Object, error) {
	obj, ok := c.objects[name]
	if ok {
		return obj, nil
	}
	return nil, fmt.Errorf("could not find %s", name)
}

func (c *DummyClient) List(opts v1.ListOptions) (Collection, error) {
	var items []Object
	for _, i := range c.objects {
		items = append(items, i)
	}
	r := c.NewCollection()
	r.SetItems(items)
	return r, nil
}

type Client struct {
	cl     *rest.RESTClient
	ns     string
	plural string
	codec  runtime.ParameterCodec
}

func (c *Client) NewObject() Object {
	return knownTypes[c.plural].object.DeepCopyObject().(Object)
}

func (c *Client) NewCollection() Collection {
	return knownTypes[c.plural].collection.DeepCopyObject().(Collection)
}

func (c *Client) Create(obj Object) (Object, error) {
	result := c.NewObject()
	err := c.cl.Post().
		Namespace(c.ns).
		Resource(c.plural).
		Name(obj.GetName()).
		Body(obj).
		Do().
		Into(result)
	return result, err
}

func (c *Client) Update(obj Object) (Object, error) {
	result := c.NewObject()
	err := c.cl.Put().
		Namespace(c.ns).
		Resource(c.plural).
		Body(obj).
		Name(obj.GetName()).
		Do().
		Into(result)
	return result, err
}

func (c *Client) Delete(name string, options *v1.DeleteOptions) error {
	return c.cl.Delete().
		Namespace(c.ns).
		Resource(c.plural).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (c *Client) Get(name string) (Object, error) {
	result := c.NewObject()
	err := c.cl.Get().
		Namespace(c.ns).
		Resource(c.plural).
		Name(name).
		Do().
		Into(result)
	return result, err
}

func (c *Client) List(opts v1.ListOptions) (Collection, error) {
	result := c.NewCollection()
	err := c.cl.Get().
		Namespace(c.ns).
		Resource(c.plural).
		VersionedParams(&opts, c.codec).
		Do().
		Into(result)
	return result, err
}
