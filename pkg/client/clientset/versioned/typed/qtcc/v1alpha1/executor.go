/*
Copyright 2022.

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
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/quanxiang-cloud/qtcc/pkg/apis/qtcc/v1alpha1"
	scheme "github.com/quanxiang-cloud/qtcc/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ExecutorsGetter has a method to return a ExecutorInterface.
// A group's client should implement this interface.
type ExecutorsGetter interface {
	Executors(namespace string) ExecutorInterface
}

// ExecutorInterface has methods to work with Executor resources.
type ExecutorInterface interface {
	Create(ctx context.Context, executor *v1alpha1.Executor, opts v1.CreateOptions) (*v1alpha1.Executor, error)
	Update(ctx context.Context, executor *v1alpha1.Executor, opts v1.UpdateOptions) (*v1alpha1.Executor, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Executor, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ExecutorList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Executor, err error)
	ExecutorExpansion
}

// executors implements ExecutorInterface
type executors struct {
	client rest.Interface
	ns     string
}

// newExecutors returns a Executors
func newExecutors(c *QtccV1alpha1Client, namespace string) *executors {
	return &executors{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the executor, and returns the corresponding executor object, and an error if there is any.
func (c *executors) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Executor, err error) {
	result = &v1alpha1.Executor{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("executors").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Executors that match those selectors.
func (c *executors) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ExecutorList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ExecutorList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("executors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested executors.
func (c *executors) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("executors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a executor and creates it.  Returns the server's representation of the executor, and an error, if there is any.
func (c *executors) Create(ctx context.Context, executor *v1alpha1.Executor, opts v1.CreateOptions) (result *v1alpha1.Executor, err error) {
	result = &v1alpha1.Executor{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("executors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(executor).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a executor and updates it. Returns the server's representation of the executor, and an error, if there is any.
func (c *executors) Update(ctx context.Context, executor *v1alpha1.Executor, opts v1.UpdateOptions) (result *v1alpha1.Executor, err error) {
	result = &v1alpha1.Executor{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("executors").
		Name(executor.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(executor).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the executor and deletes it. Returns an error if one occurs.
func (c *executors) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("executors").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *executors) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("executors").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched executor.
func (c *executors) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Executor, err error) {
	result = &v1alpha1.Executor{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("executors").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
