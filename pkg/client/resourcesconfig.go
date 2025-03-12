package client

import (
	"context"

	oblikv1 "github.com/SocialGouv/oblik/pkg/apis/oblik/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type ResourcesConfigInterface interface {
	List(ctx context.Context, namespace string, opts metav1.ListOptions) (*oblikv1.ResourcesConfigList, error)
	Watch(ctx context.Context, namespace string, opts metav1.ListOptions) (watch.Interface, error)
	Get(ctx context.Context, namespace, name string, opts metav1.GetOptions) (*oblikv1.ResourcesConfig, error)
	Create(ctx context.Context, namespace string, resourcesConfig *oblikv1.ResourcesConfig, opts metav1.CreateOptions) (*oblikv1.ResourcesConfig, error)
	Update(ctx context.Context, namespace string, resourcesConfig *oblikv1.ResourcesConfig, opts metav1.UpdateOptions) (*oblikv1.ResourcesConfig, error)
	UpdateStatus(ctx context.Context, namespace string, resourcesConfig *oblikv1.ResourcesConfig, opts metav1.UpdateOptions) (*oblikv1.ResourcesConfig, error)
	Delete(ctx context.Context, namespace, name string, opts metav1.DeleteOptions) error
}

type resourcesConfigClient struct {
	restClient rest.Interface
}

func (c *resourcesConfigClient) List(ctx context.Context, namespace string, opts metav1.ListOptions) (*oblikv1.ResourcesConfigList, error) {
	result := &oblikv1.ResourcesConfigList{}
	err := c.restClient.
		Get().
		Namespace(namespace).
		Resource("resourcesconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *resourcesConfigClient) Get(ctx context.Context, namespace, name string, opts metav1.GetOptions) (*oblikv1.ResourcesConfig, error) {
	result := &oblikv1.ResourcesConfig{}
	err := c.restClient.
		Get().
		Namespace(namespace).
		Resource("resourcesconfigs").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *resourcesConfigClient) Create(ctx context.Context, namespace string, resourcesConfig *oblikv1.ResourcesConfig, opts metav1.CreateOptions) (*oblikv1.ResourcesConfig, error) {
	result := &oblikv1.ResourcesConfig{}
	err := c.restClient.
		Post().
		Namespace(namespace).
		Resource("resourcesconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(resourcesConfig).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *resourcesConfigClient) Update(ctx context.Context, namespace string, resourcesConfig *oblikv1.ResourcesConfig, opts metav1.UpdateOptions) (*oblikv1.ResourcesConfig, error) {
	result := &oblikv1.ResourcesConfig{}
	err := c.restClient.
		Put().
		Namespace(namespace).
		Resource("resourcesconfigs").
		Name(resourcesConfig.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(resourcesConfig).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *resourcesConfigClient) UpdateStatus(ctx context.Context, namespace string, resourcesConfig *oblikv1.ResourcesConfig, opts metav1.UpdateOptions) (*oblikv1.ResourcesConfig, error) {
	result := &oblikv1.ResourcesConfig{}
	err := c.restClient.
		Put().
		Namespace(namespace).
		Resource("resourcesconfigs").
		Name(resourcesConfig.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(resourcesConfig).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *resourcesConfigClient) Watch(ctx context.Context, namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(namespace).
		Resource("resourcesconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}

func (c *resourcesConfigClient) Delete(ctx context.Context, namespace, name string, opts metav1.DeleteOptions) error {
	return c.restClient.
		Delete().
		Namespace(namespace).
		Resource("resourcesconfigs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// ResourcesConfigClientset is a clientset for ResourcesConfig CRD
type ResourcesConfigClientset struct {
	restClient rest.Interface
}

// OblikV1 returns the OblikV1Client
func (c *ResourcesConfigClientset) OblikV1() ResourcesConfigInterface {
	return &resourcesConfigClient{
		restClient: c.restClient,
	}
}

// NewForConfig creates a new ResourcesConfigClientset for the given config
func NewResourcesConfigClientset(c *rest.Config) (*ResourcesConfigClientset, error) {
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: oblikv1.GroupName, Version: oblikv1.Version}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &ResourcesConfigClientset{restClient: client}, nil
}

// AddToScheme adds the ResourcesConfig types to the scheme
func AddToScheme() {
	// Create a scheme builder
	schemeBuilder := runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
		// Register for external version (v1)
		scheme.AddKnownTypes(
			schema.GroupVersion{Group: oblikv1.GroupName, Version: oblikv1.Version},
			&oblikv1.ResourcesConfig{},
			&oblikv1.ResourcesConfigList{},
		)
		metav1.AddToGroupVersion(scheme, schema.GroupVersion{Group: oblikv1.GroupName, Version: oblikv1.Version})

		// Register for internal version
		internalGV := schema.GroupVersion{Group: oblikv1.GroupName, Version: runtime.APIVersionInternal}
		scheme.AddKnownTypes(internalGV,
			&oblikv1.ResourcesConfig{},
			&oblikv1.ResourcesConfigList{},
		)

		return nil
	})

	// Add to client-go scheme
	if err := schemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
}
